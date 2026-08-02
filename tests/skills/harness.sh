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

skill_test_init() {
  if ! command -v claude >/dev/null 2>&1; then
    echo "SKIP: claude CLI not on PATH"
    exit 0
  fi
  if [ ! -x "$MK" ]; then
    echo "building mk..."
    ( cd "$REPO_ROOT" && make build >/dev/null )
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
}

skill_test_cleanup() {
  rm -rf "$db_dir"
  rm -rf "$work_dir"
}

# run_skill "<prompt>" — runs one agent turn in the temp working directory.
run_skill() {
  ( cd "$work_dir" && MK_DB="$mk_db" claude -p "$1" ) >/dev/null 2>&1
}

# assert_json "<mk args>" "<jq filter>" "<description>"
assert_json() {
  local args=$1 filter=$2 desc=$3
  local out
  out=$(eval "\"$MK\" --json $args" 2>&1)
  if echo "$out" | jq -e "$filter" >/dev/null 2>&1; then
    echo "  ok: $desc"
  else
    echo "  FAIL: $desc"
    echo "    filter: $filter"
    echo "    got:    $(echo "$out" | head -c 200)"
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
