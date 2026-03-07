.PHONY: build run clean test

BINARY := conductor-relay
PKG := ./cmd/conductor-relay

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
