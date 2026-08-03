#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/harness.sh"
skill_test_init

git -c init.defaultBranch=main init -q "$work_dir"
cat > "$work_dir/article.md" <<'ARTICLE'
# Rate limiting strategies

Token bucket allows bursts up to the bucket size. Leaky bucket smooths
output to a fixed rate. Sliding window counts requests in a moving
interval and is the most memory-hungry of the three.
ARTICLE

run_skill "/mk-wiki ingest the file article.md in this directory"

assert_json "source ls" 'length == 1' "captured one source"
assert_json "source ls" '.[0].body | test("token bucket"; "i")' "source body captured, not just the title"
assert_json "wiki ls"   'length >= 1' "wrote at least one wiki page"
assert_json "link ls"   'map(select(.relation == "derived-from")) | length >= 1' "linked the page to its source"
assert_json "log ls"    '.[0].kind == "ingest"' "log entry kind is ingest"

# The citation edge is what keeps doctor quiet about this source.
assert_json "doctor --scope wiki" 'map(select(.check == "wiki.unprocessed")) | length == 0' "source is no longer unprocessed"

skill_test_done
