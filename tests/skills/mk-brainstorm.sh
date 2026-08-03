#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/harness.sh"
skill_test_init

git -c init.defaultBranch=main init -q "$work_dir"

# A single non-interactive `claude -p` turn cannot hold the skill's real
# Socratic dialogue (one question per message, across several replies), so
# this prompt pre-supplies everything the skill would otherwise draw out
# over multiple turns: a topic, an already-approved direction, and an
# explicit instruction to skip straight to decomposition. Same pre-answer
# technique as tests/skills/mk-init.sh and tests/skills/mk-sync.sh applied
# to a different skill's own gate — here, "approve a direction before any
# epic or story gets created" rather than a name or a fix confirmation.
#
# The working directory is unregistered, so Ground runs mk-init inline
# first — and mk-init's own method has its own real confirmation gate (the
# project name), independent of this skill's approval gate. Pre-answering
# only the brainstorm gate and leaving the name gate unanswered reproduces
# exactly the failure this comment now documents: the turn stops at
# mk-init's question and never reaches the brainstorm dialogue at all, so
# the project name has to be pre-supplied too, same as mk-init.sh does.
run_skill "/mk-brainstorm I want to add a CSV export feature to this project. If this directory isn't registered yet, name the project csv-export-test and proceed without asking anything else about that. Skip the questions and the menu of approaches — I already approve this direction: add a straightforward 'mk export csv' command that reads the existing data model and writes rows to stdout. Go ahead and break it into an epic with at least two stories and file it now."

assert_json "epic ls"  'length == 1' "created exactly one epic"
assert_json "story ls" 'length >= 2' "broke the epic into at least two stories"

EPIC_ID=$("$MK" epic ls --json | jq -r '.[0].id')
assert_json "story ls" "map(.epic_id == \"$EPIC_ID\") | all" "every story hangs off the epic this skill created"

assert_json "log ls" 'length >= 1' "appended a log entry"
assert_json "log ls" '.[0].kind == "brainstorm"' "log entry kind is brainstorm"

# Brainstorming writes no document of its own — that split belongs to
# mk-spec, which picks up a story from here.
assert_json "wiki ls" 'length == 0' "wrote no spec page"

# The epic this skill created has stories under it, so epic.empty must
# not fire for it.
assert_json "doctor --scope stories" "map(select(.check == \"epic.empty\" and .id == \"$EPIC_ID\")) | length == 0" "epic.empty check clean for the new epic"

skill_test_done
