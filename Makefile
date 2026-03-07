.PHONY: build run clean test

BUILD_DIR := .build
BINARY := $(BUILD_DIR)/crelay
PKG := ./cmd/crelay
export GIT_WORK_TREE ?= $(shell git rev-parse --show-toplevel 2>/dev/null)

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BINARY) $(PKG)

run: build
	./$(BINARY)

clean:
	rm -rf $(BUILD_DIR)
	go clean

test:
	go test ./...

lint:
	golangci-lint run ./...
