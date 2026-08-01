.PHONY: build test install clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/khanhnguyendev/mind-knowledge/internal/cli.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o mk ./cmd/mk

test:
	# tests/cli builds and execs a binary at runtime, so Go's test cache
	# can't see that it depends on the rest of the module (see
	# tests/cli/harness_test.go); everything else is safe to cache.
	go test $$(go list ./... | grep -v /tests/cli)
	go test -count=1 ./tests/cli/...

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/mk

clean:
	rm -f mk
