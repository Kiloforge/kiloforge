.PHONY: build build-frontend build-backend dev test test-coverage test-integration test-smoke test-all clean gen-api verify-codegen

BIN_DIR := .build
BINARY := $(BIN_DIR)/kf
DIST_DIR := backend/internal/adapter/dashboard/dist

# Ensure dist/ has at least a placeholder so //go:embed dist/* succeeds.
# Only creates if dist/ is empty or missing — never overwrites real assets.
ensure-dist:
	@if [ ! -f $(DIST_DIR)/index.html ]; then \
		mkdir -p $(DIST_DIR) && \
		echo '<!DOCTYPE html><html><body><p>Frontend not built. Run <code>make build</code>.</p></body></html>' > $(DIST_DIR)/index.html; \
	fi

build:
	$(MAKE) build-frontend
	$(MAKE) build-backend

build-frontend:
	cd frontend && npm ci && npm run build

build-backend: ensure-dist
	@mkdir -p $(BIN_DIR)
	$(eval GIT_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev"))
	$(eval GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none"))
	$(eval BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ))
	cd backend && GIT_DIR=$$(git rev-parse --git-common-dir) GIT_WORK_TREE=$$(cd .. && pwd) \
		go build -ldflags "-s -w -X main.version=$(GIT_VERSION) -X main.commit=$(GIT_COMMIT) -X main.date=$(BUILD_DATE)" \
		-o ../$(BINARY) ./cmd/kf

dev: ensure-dist
	@trap 'kill 0' INT TERM; \
	cd backend && go run ./cmd/kf up & \
	cd frontend && npm run dev & \
	wait

test: ensure-dist
	cd backend && go test -race ./...

test-smoke: ensure-dist
	cd backend && go test -race -run "TestBinaryBuilds|TestRouteRegistration|TestAllCommandsRegistered|TestCommandHelp" ./...

test-integration: ensure-dist
	cd backend && go test -race -tags=integration ./...

test-all: ensure-dist
	cd backend && go test -race -tags=integration ./...

test-coverage: ensure-dist
	cd backend && go test -race -coverprofile=coverage.out ./...
	cd backend && go tool cover -func=coverage.out
	@echo "HTML report: go tool cover -html=backend/coverage.out"

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(DIST_DIR)

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
