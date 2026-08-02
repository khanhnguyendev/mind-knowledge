#!/usr/bin/env bash
# Shared harness for skill state tests.
#
# These tests drive a real agent, so they are slow and consume tokens. They
# run under `make test-skills`, never inside `make test`.
#
# A skill test asserts the *database* landed correctly. It cannot assert the
# skill followed its method — that is what tests/skills/PRESSURE.md covers.

set -uo pipefail

SKILL_TEST_FAILURES=0
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MK="$REPO_ROOT/mk"

# Claude Code resolves "/skill-name" against personal skills in
# ~/.claude/skills/<name>/SKILL.md (confirmed via `claude --help`'s --bare
# note: "Skills still resolve via /skill-name") or against an installed
# plugin's skills/ — this repo is neither until Task 14 packages it as an
# installed plugin. Until then, a `claude -p` subprocess run from a scratch
# work_dir has no way to discover skills/<name>/SKILL.md on its own, so the
# harness symlinks every skill this repo currently ships into the personal
# skills directory for the duration of the run and removes exactly the
# links it added on cleanup. Once Task 14 lands, this whole mechanism can be
# deleted — the plugin install will make the skills discoverable on its own.
PERSONAL_SKILLS_DIR="$HOME/.claude/skills"
SKILL_TEST_LINKS=()

link_skills_for_test() {
  mkdir -p "$PERSONAL_SKILLS_DIR"
  local dir name target link existing
  for dir in "$REPO_ROOT"/skills/*/; do
    name="$(basename "$dir")"
    [ "$name" = "references" ] && continue
    [ -f "$dir/SKILL.md" ] || continue
    target="$REPO_ROOT/skills/$name"
    link="$PERSONAL_SKILLS_DIR/$name"
    if [ -L "$link" ]; then
      # Already a symlink — either a previous crashed run's leftover (safe
      # to leave; it points at the same place we would point it) or a real
      # personal skill of the same name pointed elsewhere (never touch
      # someone else's skill). Either way, don't re-link and don't queue it
      # for removal — a link we didn't create is not ours to delete.
      continue
    elif [ -e "$link" ]; then
      # A real file or directory, not a symlink: something the user placed
      # there on purpose. Never clobber it.
      continue
    fi
    ln -s "$target" "$link"
    SKILL_TEST_LINKS+=("$link")
  done
}

unlink_skills_for_test() {
  local link
  for link in "${SKILL_TEST_LINKS[@]:-}"; do
    [ -n "$link" ] && [ -L "$link" ] && rm -f "$link"
  done
  SKILL_TEST_LINKS=()
}

skill_test_init() {
  if ! command -v claude >/dev/null 2>&1; then
    echo "SKIP: claude CLI not on PATH"
    exit 0
  fi
  if ! command -v jq >/dev/null 2>&1; then
    echo "SKIP: jq not on PATH"
    exit 0
  fi
  # Always rebuild rather than only building when $MK is missing: `go
  # build` no-ops (fast) when nothing changed, but a plain `[ ! -x "$MK" ]`
  # guard would happily keep running a stale binary forever once one
  # exists, silently testing old `mk` behaviour against new skill files.
  if ! ( cd "$REPO_ROOT" && make build >/dev/null ); then
    echo "FAIL: mk build failed"
    exit 1
  fi
  if [ ! -x "$MK" ]; then
    echo "FAIL: mk binary missing after build"
    exit 1
  fi
  # mktemp's `X` substitution is only guaranteed at the end of the
  # template — a suffix after the `X` run (e.g. `.db`) is silently not
  # substituted on BSD mktemp (macOS), so every call would return the same
  # literal path. Randomize a directory instead, which both BSD and GNU
  # mktemp handle correctly, and put the database inside it.
  db_dir=$(mktemp -d /tmp/mkskill-XXXXXX)
  mk_db="$db_dir/mk.db"
  export MK_DB="$mk_db"
  work_dir=$(mktemp -d /tmp/mkwork-XXXXXX)
  link_skills_for_test
  trap skill_test_cleanup EXIT
}

skill_test_cleanup() {
  unlink_skills_for_test
  rm -rf "$db_dir"
  rm -rf "$work_dir"
}

# run_skill "<prompt>" — runs one agent turn in the temp working directory.
#
# Most tests call this as a bare statement and never look at its return
# value, so a crashed, rate-limited, or timed-out agent turn must not be
# able to pass for "the skill correctly did nothing": on a nonzero exit,
# run_skill prints a loud CRASH line (with a slice of the agent's combined
# output) and counts it as a failed assertion via $SKILL_TEST_FAILURES, so
# skill_test_done fails the run even if nobody checked the return value. A
# caller that does want to branch on it directly still can — the real
# `claude` exit status is returned.
#
# $work_dir is a scratch directory `claude` has never seen before, so it
# starts every run untrusted: the agent's Bash calls (every `mk` and `git`
# invocation a skill makes) are denied outright rather than prompted for,
# since there is no terminal to prompt on non-interactively — confirmed by
# capturing a run's `permission_denials` under `--output-format json`.
# `--allowedTools "Bash"` grants it for this run only; it is safe to grant
# broadly here because $work_dir and $mk_db are both disposable scratch
# state the harness tears down afterward. Prepending $REPO_ROOT to PATH
# makes the freshly-built `./mk` (not some stale copy elsewhere on the
# machine) the one the agent's bare `mk` calls resolve to, so a test
# provably exercises the binary `skill_test_init` just built.
run_skill() {
  local out status
  out=$(cd "$work_dir" && PATH="$REPO_ROOT:$PATH" MK_DB="$mk_db" claude -p "$1" --allowedTools "Bash" 2>&1)
  status=$?
  if [ "$status" -ne 0 ]; then
    echo "  CRASH: run_skill exited $status for prompt: $1"
    echo "    output: $(echo "$out" | head -c 500)"
    SKILL_TEST_FAILURES=$((SKILL_TEST_FAILURES + 1))
  fi
  return "$status"
}

# assert_json "<mk args>" "<jq filter>" "<description>"
#
# <mk args> is tokenized word-by-word (whitespace-separated, with simple
# '/" quoting understood — via `xargs`) rather than eval'd, so a value
# round-tripped out of `mk` (an id, a title) landing in <mk args> can't be
# read as shell syntax. It is still not a real shell: no globs, no `$()`,
# no variable expansion — keep <mk args> to flags and literal values.
assert_json() {
  local args=$1 filter=$2 desc=$3
  local raw xstatus
  raw=$(xargs -n1 <<<"$args" 2>&1)
  xstatus=$?
  if [ "$xstatus" -ne 0 ]; then
    echo "  FAIL: $desc"
    echo "    error: could not parse mk args \"$args\": $raw"
    SKILL_TEST_FAILURES=$((SKILL_TEST_FAILURES + 1))
    return
  fi
  local -a argv
  readarray -t argv <<<"$raw"
  local out
  out=$("$MK" --json "${argv[@]}" 2>&1)
  local jq_err jq_status
  jq_err=$(echo "$out" | jq -e "$filter" 2>&1 1>/dev/null)
  jq_status=$?
  if [ "$jq_status" -eq 0 ]; then
    echo "  ok: $desc"
  else
    echo "  FAIL: $desc"
    echo "    filter: $filter"
    echo "    got:    $(echo "$out" | head -c 200)"
    if [ -n "$jq_err" ]; then
      echo "    jq:     $jq_err"
    fi
    SKILL_TEST_FAILURES=$((SKILL_TEST_FAILURES + 1))
  fi
}

skill_test_done() {
  skill_test_cleanup
  if [ "$SKILL_TEST_FAILURES" -gt 0 ]; then
    echo "$SKILL_TEST_FAILURES assertion(s) failed"
    exit 1
  fi
  echo "all assertions passed"
}
