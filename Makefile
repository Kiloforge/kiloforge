.PHONY: build run clean test

BINARY := crelay
PKG := ./cmd/crelay
export GIT_WORK_TREE ?= $(shell git rev-parse --show-toplevel 2>/dev/null)
build:
	go build -o $(BINARY) $(PKG)

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
	go clean

test:
	go test ./...

lint:
	golangci-lint run ./...
