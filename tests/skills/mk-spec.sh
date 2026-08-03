#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/harness.sh"
skill_test_init

git -c init.defaultBranch=main init -q "$work_dir"

# Pre-register the project directly through $MK rather than letting Ground
# hand off to mk-init inline: mk-init's own confirmation gate (the project
# name) would stall a single non-interactive turn unless pre-answered too,
# and that gate has nothing to do with what this test is exercising. Same
# fixture technique as tests/skills/mk-wiki.sh.
"$MK" project add --name test-spec --path "$work_dir" >/dev/null

# Pre-create the epic (and a story under it) that mk-brainstorm would have
# produced in a prior turn — this skill consumes an already-approved
# design, it doesn't run its own brainstorm, so the fixture supplies the
# epic mk-spec is supposed to pick up and later link back to.
EPIC_ID=$("$MK" epic create -p test-spec --title "Rate limiter for the API gateway" --description "Add request rate limiting in front of the gateway.")
"$MK" story create --epic "$EPIC_ID" --title "Implement token bucket limiter" >/dev/null

# A single non-interactive `claude -p` turn cannot hold the skill's real
# self-review-then-present-for-review dialogue, so this prompt pre-supplies
# the approved design and pre-answers the review gate, the same pre-answer
# technique tests/skills/mk-brainstorm.sh and tests/skills/mk-init.sh use
# for their own gates.
run_skill "/mk-spec Write up the approved design for epic $EPIC_ID (Rate limiter for the API gateway) as a spec page. The approved design: use a token bucket algorithm, one bucket per API key, refilled at a fixed rate, checked in gateway middleware before a request reaches any backend; rejected requests get a 429. I have already reviewed this and approve it as final — go ahead and file it, link it to the epic, and finish the turn without asking me anything else."

# Grounding: project registered at the right path, exactly one.
assert_json "project ls" 'length == 1' "exactly one project registered"
assert_json "project ls" '.[0].repo_path == "'"$work_dir"'"' "project path matches work_dir"

# Exactly one spec-kind wiki page, with a summary (a page with no summary
# is invisible in `mk wiki index`).
assert_json "wiki ls" 'map(select(.kind == "spec")) | length == 1' "exactly one spec-kind wiki page"
assert_json "wiki ls" 'map(select(.kind == "spec"))[0].summary | length > 0' "spec page has a non-empty summary"

SPEC_ID=$("$MK" wiki ls --json | jq -r 'map(select(.kind == "spec"))[0].id')

# Linked from the epic to the spec page, not the reverse: wiki.orphans
# checks for inbound links, so only a link that names the spec page as
# its target closes the check.
assert_json "link ls" "map(select(.relation == \"implements\" and .from_id == \"$EPIC_ID\" and .to_id == \"$SPEC_ID\")) | length == 1" "epic links to the spec page with relation implements"

assert_json "log ls" 'length >= 1' "appended a log entry"
assert_json "log ls" '.[0].kind == "spec"' "log entry kind is spec"

# The link from step 6 should already have closed out wiki.orphans for
# this specific page.
assert_json "doctor --scope wiki" "map(select(.check == \"wiki.orphans\" and .id == \"$SPEC_ID\")) | length == 0" "wiki.orphans clean for the new spec page"

skill_test_done
