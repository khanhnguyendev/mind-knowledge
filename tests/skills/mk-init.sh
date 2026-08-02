#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/harness.sh"
skill_test_init

# A fresh, unregistered directory — pre-initialized as a git repo so the
# skill's Method step 2 ("is this already a git repository?") finds yes and
# skips straight past its own confirmation gate. That gate is real,
# required behaviour (never run `git init` unasked), but this is a single
# non-interactive turn with no way to answer it, so the fixture avoids
# triggering it rather than the prompt working around it — a second,
# separate technique from the name pre-answer below, for a second,
# separate confirmation point in the same skill.
git -c init.defaultBranch=main init -q "$work_dir"

# run_skill is one non-interactive turn (`claude -p`) with no way to answer
# a follow-up question, but the skill's method correctly asks the user to
# confirm the project name before registering (Method step 3 in
# skills/mk-init/SKILL.md) — a real, required part of its behaviour, not a
# defect. Pre-answering both the name and "don't ask, proceed" inside the
# prompt itself is how every scripted test in this suite exercises a skill
# whose method includes a confirmation step; see Task 8's mk-brainstorm
# test for the same pattern. Do not weaken the skill's prose to make a
# single-turn test pass — supply the answer instead.
run_skill "/mk-init Name the project mk-init-test. Proceed without asking me anything else."

assert_json "project ls" 'length == 1' "one project registered"
assert_json "project ls" ".[0].repo_path == \"$work_dir\"" "registered at the working directory"
assert_json "project ls" '.[0].status == "active"' "project is active"
assert_json "wiki ls"    'length >= 1'  "seeded a project wiki page"
assert_json "log ls"     'length >= 1'  "appended a log entry"
assert_json "log ls"     '.[0].kind == "init"' "log entry kind is init"

# Running it again must not duplicate.
run_skill "/mk-init"
assert_json "project ls" 'length == 1' "re-running does not duplicate the project"

skill_test_done
