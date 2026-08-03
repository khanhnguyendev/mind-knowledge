#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/harness.sh"
skill_test_init

git -c init.defaultBranch=main init -q "$work_dir"
# Local identity, not global — a machine running this suite may have no
# git identity configured at all, and this test's commits (the seed one
# below, plus whatever the skill makes per step) must not depend on it.
git -C "$work_dir" config user.email "mk-implement-test@example.com"
git -C "$work_dir" config user.name "mk-implement test harness"

# Pre-seed an initial commit — mk-implement commits per step on top of
# real repo history, not into an empty, commit-less working tree.
echo "# scratch project" > "$work_dir/README.md"
git -C "$work_dir" add README.md
git -C "$work_dir" commit -q -m "initial commit"

# Pre-register the project directly through $MK rather than letting Ground
# hand off to mk-init inline — same fixture technique as tests/skills/
# mk-plan.sh, mk-spec.sh, and mk-wiki.sh; mk-init's own naming gate has
# nothing to do with what this test exercises.
"$MK" project add --name test-implement --path "$work_dir" >/dev/null

# Pre-create the epic and story that mk-plan would have handed off in a
# prior turn, already carrying a plan — this skill consumes an
# already-planned story, it doesn't write one. The plan is deliberately
# trivial (create one file, make it executable, verify it, commit) so a
# single non-interactive agent turn can actually execute it in full.
EPIC_ID=$("$MK" epic create -p test-implement --title "Greeting script" --description "Add a tiny greeting script to the repo as a vertical slice worth exercising end to end.")
STORY_ID=$("$MK" story create --epic "$EPIC_ID" --title "Add greet.sh" --description "Add a greet.sh script to the project root that prints a greeting.")

PLAN=$(cat <<'PLAN_EOF'
Step 1 — create greet.sh: in the project root, create a file named
greet.sh containing a POSIX shell shebang and a single line that echoes
"Hello from mk-implement". Commit this step before moving on.

Step 2 — make it executable: chmod +x greet.sh so it can be run directly
as ./greet.sh. Commit this step before moving on.

Step 3 — verify: run ./greet.sh and confirm the output is exactly
"Hello from mk-implement". This step makes no file change, so it needs
no commit of its own — just confirm the script behaves as intended
before moving to the last step.

Step 4 — final commit: confirm the working tree is clean (everything
from steps 1-2 already committed); if anything is still uncommitted,
commit it now.

Interfaces: this story produces one new file, greet.sh, at the project
root, executable, with no inputs consumed from anywhere else.
PLAN_EOF
)
"$MK" story edit "$STORY_ID" --plan "$PLAN" >/dev/null

# Move the story to ready, the status mk-plan would have left it in —
# mk-implement picks up from ready, not from backlog.
"$MK" story mv "$STORY_ID" --to ready >/dev/null

run_skill "/mk-implement Implement story $STORY_ID (Add greet.sh) under epic $EPIC_ID by following its plan exactly. This environment only grants the Bash tool, so create and edit every file with shell commands (cat heredocs, chmod, etc.) rather than an editor tool. Execute each step in order, commit as you go, append your progress to the story's notes, and move the story to review once every step has landed."

# Grounding: project registered at the right path, exactly one.
assert_json "project ls" 'length == 1' "exactly one project registered"
assert_json "project ls" '.[0].repo_path == "'"$work_dir"'"' "project path matches work_dir"

# No story or epic got created or duplicated — this skill implements an
# existing story, it doesn't make a new one.
assert_json "story ls" 'length == 1' "still exactly one story"
assert_json "epic ls" 'length == 1' "still exactly one epic"

# The plan actually got executed: the file it called for exists and is
# executable.
if [ -f "$work_dir/greet.sh" ]; then
  echo "  ok: greet.sh was created"
else
  echo "  FAIL: greet.sh was created"
  echo "    expected file at $work_dir/greet.sh"
  SKILL_TEST_FAILURES=$((SKILL_TEST_FAILURES + 1))
fi

if [ -x "$work_dir/greet.sh" ]; then
  echo "  ok: greet.sh is executable"
else
  echo "  FAIL: greet.sh is executable"
  SKILL_TEST_FAILURES=$((SKILL_TEST_FAILURES + 1))
fi

# Commit per step, not per story: more than just the pre-seeded initial
# commit should exist once the plan's landed.
commit_count=$(git -C "$work_dir" log --oneline | wc -l | tr -d ' ')
if [ "$commit_count" -gt 1 ]; then
  echo "  ok: more than the seed commit exists ($commit_count total)"
else
  echo "  FAIL: more than the seed commit exists ($commit_count total)"
  SKILL_TEST_FAILURES=$((SKILL_TEST_FAILURES + 1))
fi

# Story reached review, not done — mk-verify is what closes it.
assert_json "story view $STORY_ID" '.status == "review"' "story moved to review"

# Notes carry real progress, not a stub — appended as execution went, per
# the skill's own method, not written up once at the end.
assert_json "story view $STORY_ID" '(.notes // "") | length > 20' "notes hold real progress, not a stub"

assert_json "log ls" 'length >= 1' "appended a log entry"
assert_json "log ls" '.[0].kind == "implement"' "log entry kind is implement"

skill_test_done
