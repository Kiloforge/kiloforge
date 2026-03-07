.PHONY: build build-frontend build-backend dev test clean gen-api verify-codegen

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

gen-api:
	cd backend && oapi-codegen -config api/cfg.yaml api/openapi.yaml
	cd backend && oapi-codegen -config api/cfg-client.yaml api/openapi.yaml

verify-codegen:
	@cp backend/internal/adapter/rest/gen/server.gen.go /tmp/server.gen.go.bak
	@cp backend/internal/adapter/rest/gen/client.gen.go /tmp/client.gen.go.bak
	@cd backend && oapi-codegen -config api/cfg.yaml api/openapi.yaml
	@cd backend && oapi-codegen -config api/cfg-client.yaml api/openapi.yaml
	@diff backend/internal/adapter/rest/gen/server.gen.go /tmp/server.gen.go.bak >/dev/null 2>&1 || (cp /tmp/server.gen.go.bak backend/internal/adapter/rest/gen/server.gen.go && echo "server.gen.go is out of date — run 'make gen-api'" && exit 1)
	@diff backend/internal/adapter/rest/gen/client.gen.go /tmp/client.gen.go.bak >/dev/null 2>&1 || (cp /tmp/client.gen.go.bak backend/internal/adapter/rest/gen/client.gen.go && echo "client.gen.go is out of date — run 'make gen-api'" && exit 1)
	@echo "Generated code is up to date."
