#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/harness.sh"
skill_test_init

git -c init.defaultBranch=main init -q "$work_dir"
"$MK" project add --name present --path "$work_dir" >/dev/null
"$MK" project add --name vanished --path /tmp/definitely-not-here-99 >/dev/null

assert_json "sync" 'map(.state) | index("missing") != null' "fixture produces a missing project"

run_skill "/mk-sync"

# Reporting only: a drifting project must not be silently archived or re-pointed.
assert_json "project ls" 'length == 2' "no project removed without asking"
assert_json "project ls" 'map(select(.name == "vanished"))[0].status == "active"' "vanished project not silently archived"
assert_json "log ls" '.[0].kind == "sync"' "log entry kind is sync"

skill_test_done
