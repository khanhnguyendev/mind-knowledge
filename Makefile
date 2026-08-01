.PHONY: build test install clean

build:
	go build -o mk ./cmd/mk

test:
	go test ./...

install:
	go install ./cmd/mk

clean:
	rm -f mk
