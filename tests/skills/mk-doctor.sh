#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/harness.sh"
skill_test_init

git -c init.defaultBranch=main init -q "$work_dir"
P=$("$MK" project add --name drifty --path "$work_dir")
E=$("$MK" epic create -p "$P" --title "empty epic")
"$MK" source add --title "unprocessed source" --body "nothing derives from this" >/dev/null

# epic.empty and wiki.unprocessed should both be findable.
assert_json "doctor" 'map(.check) | index("epic.empty") != null' "fixture produces epic.empty"
assert_json "doctor" 'map(.check) | index("wiki.unprocessed") != null' "fixture produces wiki.unprocessed"

run_skill "/mk-doctor Give me the report only. Do not apply any fixes — I have not approved any yet."

# The skill reports; it must not silently repair.
assert_json "doctor" 'map(.check) | index("epic.empty") != null' "epic.empty still present — nothing auto-repaired"
assert_json "log ls" 'length >= 1' "appended a log entry"
assert_json "log ls" '.[0].kind == "lint"' "log entry kind is lint"

skill_test_done
