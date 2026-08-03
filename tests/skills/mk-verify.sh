#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/harness.sh"
skill_test_init

git -c init.defaultBranch=main init -q "$work_dir"
# Local identity, not global — a machine running this suite may have no
# git identity configured at all. Same fixture technique as tests/skills/
# mk-implement.sh and tests/skills/mk-review.sh, even though this test
# doesn't itself expect the skill to commit anything.
git -C "$work_dir" config user.email "mk-verify-test@example.com"
git -C "$work_dir" config user.name "mk-verify test harness"

# Pre-seed an initial commit and the reviewed change this story is
# sitting in review for.
echo "# scratch project" > "$work_dir/README.md"
git -C "$work_dir" add README.md
git -C "$work_dir" commit -q -m "initial commit"

# Pre-seed the project's own verification tooling — mk-verify's whole
# reason for existing is running this for real, right now, rather than
# trusting a claim that it already ran. It exits 0 and prints real,
# specific, greppable output, the way a real test runner would.
cat > "$work_dir/test.sh" <<'SCRIPT_EOF'
#!/bin/sh
echo "running suite..."
echo "2 tests passed"
exit 0
SCRIPT_EOF
chmod +x "$work_dir/test.sh"
git -C "$work_dir" add test.sh
git -C "$work_dir" commit -q -m "Add test.sh verification script"

# Pre-register the project directly through $MK rather than letting Ground
# hand off to mk-init inline — same fixture technique as tests/skills/
# mk-implement.sh, mk-plan.sh, mk-review.sh, mk-spec.sh, and mk-wiki.sh;
# mk-init's own naming gate has nothing to do with what this test
# exercises.
"$MK" project add --name test-verify --path "$work_dir" >/dev/null

# Pre-create the epic and story that mk-review would have handed off in a
# prior turn, carrying the plan mk-plan would have written and the notes
# mk-implement and mk-review would have left — this skill consumes an
# already-reviewed story, it doesn't plan, implement, or review anything
# itself.
EPIC_ID=$("$MK" epic create -p test-verify --title "Test.sh verification script" --description "Add a test.sh script the project uses to verify itself.")
STORY_ID=$("$MK" story create --epic "$EPIC_ID" --title "Add test.sh" --description "Add a test.sh script to the project root that verifies the project and prints its result.")
"$MK" story edit "$STORY_ID" --plan "Step 1 — create test.sh in the project root: a POSIX shell script that prints its result and exits 0 on success. Commit it.

Interfaces: this story produces one new file, test.sh, at the project root, executable, with no inputs consumed from anywhere else." >/dev/null
"$MK" story edit "$STORY_ID" --append-notes "Step 1 — added test.sh, printing a result line and exiting 0. Committed." >/dev/null
"$MK" story edit "$STORY_ID" --append-notes "Review: no findings. Diff is a single small script, exits 0, does what the plan called for." >/dev/null

# Move the story to review, the status mk-review would have left it in —
# mk-verify's whole entry point.
"$MK" story mv "$STORY_ID" --to review >/dev/null

run_skill "/mk-verify Verify story $STORY_ID (Add test.sh) under epic $EPIC_ID. It has been implemented and reviewed and is sitting in review. This project's own verification tooling is ./test.sh in the project root — run it now, for real, capture its actual output, and append that real output to the story's notes as evidence. If it passes, move the story to done. Finish the turn without asking me anything else."

# Grounding: project registered at the right path, exactly one.
assert_json "project ls" 'length == 1' "exactly one project registered"
assert_json "project ls" '.[0].repo_path == "'"$work_dir"'"' "project path matches work_dir"

# No story or epic got created or duplicated — this skill verifies an
# existing story, it doesn't make a new one.
assert_json "story ls" 'length == 1' "still exactly one story"
assert_json "epic ls" 'length == 1' "still exactly one epic"

# The test.sh run passed, so the story closes — this is the one skill in
# the whole suite allowed to make this transition.
assert_json "story view $STORY_ID" '.status == "done"' "story moved to done"

# The real output of test.sh — not a summary of it — landed in the notes,
# on top of what mk-implement and mk-review already left there.
assert_json "story view $STORY_ID" '(.notes // "") | test("passed"; "i")' "notes carry the real verification output"
assert_json "story view $STORY_ID" '(.notes // "") | test("Step 1")' "pre-existing implementation notes were appended to, not overwritten"
assert_json "story view $STORY_ID" '(.notes // "") | test("Review")' "pre-existing review notes were appended to, not overwritten"

assert_json "log ls" 'length >= 1' "appended a log entry"
assert_json "log ls" '.[0].kind == "done"' "log entry kind is done, not verify"

# The one doctor check this skill's own write can trip: story.planless
# fires on a done story with no plan. This story carries a real plan, so
# it should be clean — a passing run here is confirmation the skill
# actually ran doctor and reported honestly rather than skipping the
# check on the story it just closed.
assert_json "doctor --scope stories" "map(select(.check == \"story.planless\" and .id == \"$STORY_ID\")) | length == 0" "story.planless is clean — the plan mk-plan wrote is still there"

skill_test_done
