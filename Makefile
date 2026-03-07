.PHONY: build run clean test

BINARY := crelay
PKG := ./cmd/crelay

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
