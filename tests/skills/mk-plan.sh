#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/harness.sh"
skill_test_init

git -c init.defaultBranch=main init -q "$work_dir"

# Pre-register the project directly through $MK rather than letting Ground
# hand off to mk-init inline: mk-init's own naming gate would stall a
# single non-interactive turn unless pre-answered too, and that gate has
# nothing to do with what this test is exercising. Same fixture technique
# as tests/skills/mk-spec.sh and tests/skills/mk-wiki.sh.
"$MK" project add --name test-plan --path "$work_dir" >/dev/null

# Pre-create the epic and story that mk-brainstorm would have produced in
# a prior turn — this skill consumes an already-existing story, it doesn't
# create one, so the fixture supplies both what mk-plan is supposed to
# pick up and enough real description text to plan against.
EPIC_ID=$("$MK" epic create -p test-plan --title "CSV export for reports" --description "Let a user export any saved report as a CSV file from the CLI.")
STORY_ID=$("$MK" story create --epic "$EPIC_ID" --title "Add mk report export --csv" --description "Add a --csv flag to the existing 'mk report export' command. When set, write the report's rows to stdout as CSV instead of the current JSON, reusing the same row iterator the JSON path already uses so the two formats can never drift out of sync with each other.")

# A single non-interactive `claude -p` turn cannot hold a multi-message
# dialogue, so the story is named up front the same way tests/skills/
# mk-spec.sh pre-supplies the approved design — this skill has no review
# gate of its own to pre-answer, unlike mk-spec or mk-brainstorm, so
# naming the story is the only thing this prompt needs to supply.
run_skill "/mk-plan Write an implementation plan for story $STORY_ID (Add mk report export --csv) under epic $EPIC_ID. Break it into ordered, file-level steps and state the interfaces it consumes and produces, then file it."

# Grounding: project registered at the right path, exactly one.
assert_json "project ls" 'length == 1' "exactly one project registered"
assert_json "project ls" '.[0].repo_path == "'"$work_dir"'"' "project path matches work_dir"

# No story got created or duplicated — this skill plans an existing story,
# it doesn't make a new one.
assert_json "story ls" 'length == 1' "still exactly one story"

# The plan field holds real, substantial content — not a stub.
assert_json "story view $STORY_ID" '.plan != null' "story has a plan field set"
assert_json "story view $STORY_ID" '(.plan | length) > 200' "plan has real content, not a stub"

assert_json "log ls" 'length >= 1' "appended a log entry"
assert_json "log ls" '.[0].kind == "plan"' "log entry kind is plan"

# Move the story to done, the same status transition mk-implement /
# mk-review would eventually make, and confirm the plan this skill wrote
# is substantial enough that story.planless does not fire for it — that
# check only fires on a done story with no plan, and this is the one
# doctor check a plan-writing turn can ever affect (only ever clearing it,
# per mk-plan's Settle reasoning; this skill itself never touches status,
# so the transition is made here in the test, not by the skill).
"$MK" story mv "$STORY_ID" --to done >/dev/null
assert_json "doctor --scope stories" "map(select(.check == \"story.planless\" and .id == \"$STORY_ID\")) | length == 0" "story.planless does not fire — the plan mk-plan wrote is non-empty"

skill_test_done
