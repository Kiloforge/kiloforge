.PHONY: build build-frontend build-backend dev test clean

BIN_DIR := bin
BINARY := $(BIN_DIR)/crelay

build: build-frontend build-backend

build-frontend:
	cd frontend && npm ci && npm run build

build-backend:
	@mkdir -p $(BIN_DIR)
	cd backend && go build -buildvcs=false -o ../$(BINARY) ./cmd/crelay

dev:
	@trap 'kill 0' INT TERM; \
	cd backend && go run -buildvcs=false ./cmd/crelay up & \
	cd frontend && npm run dev & \
	wait

test:
	cd backend && go test -buildvcs=false -race ./...

clean:
	rm -rf $(BIN_DIR)
	rm -rf backend/internal/adapter/dashboard/dist/*
	touch backend/internal/adapter/dashboard/dist/.gitkeep

lint:
	cd backend && golangci-lint run ./...
	cd frontend && npm run lint
