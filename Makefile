.PHONY: build test test-skills install clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/khanhnguyendev/mind-knowledge/internal/cli.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o mk ./cmd/mk

test:
	# tests/cli and tests/contract both build and exec a binary at
	# runtime, so Go's test cache can't see that they depend on the rest
	# of the module (see tests/cli/harness_test.go); everything else is
	# safe to cache.
	go test $$(go list ./... | grep -vE '/tests/(cli|contract)')
	go test -count=1 ./tests/cli/... ./tests/contract/...

# test-skills drives a real `claude` agent per test file, so it is slow and
# consumes tokens. Kept out of `test` on purpose — see tests/skills/harness.sh.
test-skills:
	@for t in tests/skills/mk-*.sh; do \
		echo "=== $$t ==="; \
		bash "$$t" || exit 1; \
	done

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/mk

clean:
	rm -f mk
