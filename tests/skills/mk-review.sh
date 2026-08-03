#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/harness.sh"
skill_test_init

git -c init.defaultBranch=main init -q "$work_dir"
# Local identity, not global — a machine running this suite may have no
# git identity configured at all, and this test's commits must not depend
# on it. Same fixture technique as tests/skills/mk-implement.sh.
git -C "$work_dir" config user.email "mk-review-test@example.com"
git -C "$work_dir" config user.name "mk-review test harness"

# Pre-seed an initial commit — mk-review reads the diff of commits made
# *on top of* real repo history, not a lone commit in an empty repo.
echo "# scratch project" > "$work_dir/README.md"
git -C "$work_dir" add README.md
git -C "$work_dir" commit -q -m "initial commit"

# Pre-seed the "already implemented" change this story is sitting in
# review for: a divide.sh with an obvious, uncaught divide-by-zero risk —
# no argument validation, no zero check, straight into integer division.
# This is the defect mk-review's method (step 2: find the commits, read
# the diff; step 3: report findings graded by severity) exists to catch.
cat > "$work_dir/divide.sh" <<'SCRIPT_EOF'
#!/bin/sh
# divide.sh — divide two integers given as command-line arguments.
a="$1"
b="$2"
echo $((a / b))
SCRIPT_EOF
chmod +x "$work_dir/divide.sh"
git -C "$work_dir" add divide.sh
git -C "$work_dir" commit -q -m "Add divide.sh command-line calculator"
DEFECT_SHA=$(git -C "$work_dir" rev-parse HEAD)

# Pre-register the project directly through $MK rather than letting Ground
# hand off to mk-init inline — same fixture technique as tests/skills/
# mk-implement.sh, mk-plan.sh, mk-spec.sh, and mk-wiki.sh; mk-init's own
# naming gate has nothing to do with what this test exercises.
"$MK" project add --name test-review --path "$work_dir" >/dev/null

# Pre-create the epic and story that mk-implement would have handed off in
# a prior turn — this skill consumes a story already sitting in review
# with real commits behind it, it doesn't implement anything itself.
EPIC_ID=$("$MK" epic create -p test-review --title "Command-line divide utility" --description "Add a small divide.sh utility to the repo.")
STORY_ID=$("$MK" story create --epic "$EPIC_ID" --title "Add divide.sh" --description "Add a divide.sh script that divides two arguments and prints the result.")

# Pre-populate notes the way mk-implement's own per-step trail would have
# left them — this skill's method leans on the notes to help pin down
# which commits belong to the story.
"$MK" story edit "$STORY_ID" --append-notes "Step 1 — added divide.sh, dividing two command-line arguments and printing the result. Committed as $DEFECT_SHA." >/dev/null

# Move the story to review, the status mk-implement would have left it
# in — mk-review's whole entry point.
"$MK" story mv "$STORY_ID" --to review >/dev/null

run_skill "/mk-review Review story $STORY_ID (Add divide.sh) under epic $EPIC_ID. It is sitting in review. Its implementation is a single commit on top of the initial commit, sha $DEFECT_SHA, in this project's git repository. Read the diff of that commit, grade any findings by severity, and append them to the story's notes. If anything durable enough to warrant a wiki page turned up, write it and link it; otherwise just log the review. Finish the turn without asking me anything else."

# Grounding: project registered at the right path, exactly one.
assert_json "project ls" 'length == 1' "exactly one project registered"
assert_json "project ls" '.[0].repo_path == "'"$work_dir"'"' "project path matches work_dir"

# No story or epic got created or duplicated — this skill reviews an
# existing story, it doesn't make a new one.
assert_json "story ls" 'length == 1' "still exactly one story"
assert_json "epic ls" 'length == 1' "still exactly one epic"

# The story stays in review — mk-review never closes it, that's
# mk-verify's job.
assert_json "story view $STORY_ID" '.status == "review"' "story stays in review, not closed"

# Findings actually landed in the notes: real content naming the defect,
# graded by severity, appended on top of the pre-seeded implementation
# note rather than replacing it.
assert_json "story view $STORY_ID" '(.notes // "") | length > 60' "notes carry real review content, not a stub"
assert_json "story view $STORY_ID" '(.notes // "") | test("divide"; "i")' "notes name the divide.sh defect"
assert_json "story view $STORY_ID" '(.notes // "") | test("critical|important|minor"; "i")' "notes grade the finding by severity"

# The pre-seeded implementation note is still there — appended to, not
# clobbered by a plain --notes overwrite.
assert_json "story view $STORY_ID" '(.notes // "") | test("Step 1")' "pre-existing notes were appended to, not overwritten"

assert_json "log ls" 'length >= 1' "appended a log entry"
assert_json "log ls" '.[0].kind == "review"' "log entry kind is review"

# If a decision page turned up (optional — most reviews won't produce
# one), it must actually be linked from the story so wiki.orphans stays
# quiet on it; an unlinked page would mean step 5 started and didn't
# finish.
DECISION_COUNT=$("$MK" --json wiki ls | jq '[.[] | select(.kind == "decision")] | length')
if [ "$DECISION_COUNT" -gt 0 ]; then
  DECISION_ID=$("$MK" --json wiki ls | jq -r '[.[] | select(.kind == "decision")][0].id')
  assert_json "link ls" "map(select(.from_id == \"$STORY_ID\" and .to_id == \"$DECISION_ID\")) | length == 1" "decision page is linked from the story"
  assert_json "doctor --scope wiki" "map(select(.check == \"wiki.orphans\" and .id == \"$DECISION_ID\")) | length == 0" "wiki.orphans clean for the decision page"
else
  echo "  ok: no decision page written this turn (a legitimate outcome — most reviews won't clear that bar)"
fi

skill_test_done
