# API Schemas

kiloforge uses a **schema-first** approach for all API development. Schemas are the single source of truth — Go server code, client code, and models are generated from them.

## File Layout

```
api/
  openapi.yaml      — OpenAPI 3.1 schema for REST endpoints
  asyncapi.yaml     — AsyncAPI 3.0 schema for events and streams
  cfg.yaml          — oapi-codegen configuration
  README.md         — This file
```

## REST API (OpenAPI)

All request-response HTTP endpoints are defined in `openapi.yaml` and code is generated using `oapi-codegen`.

### Adding a New REST Endpoint

1. **Edit the schema** — Add the path, parameters, and request/response schemas to `openapi.yaml`
2. **Regenerate code** — Run `make gen-api` from the `backend/` directory
3. **Implement the interface** — The generator produces a strict server interface. Add the handler method to your server struct in `internal/adapter/rest/`
4. **Write tests** — Test the handler using `httptest`
5. **Verify** — Run `make verify-codegen` to ensure generated code matches the schema

### Code Generation

- **Tool**: `oapi-codegen` v2.5.1 (strict server mode)
- **Config**: `api/cfg.yaml`
- **Output**: `*.gen.go` files in `internal/adapter/rest/`
- **Rule**: Never edit `.gen.go` files — they are overwritten on every generation run
- **Strict typing**: Always prefer strongly-typed models. Avoid `interface{}` or `any` in schemas — tighten constraints instead

### Modifying an Existing Endpoint

1. Update the schema in `openapi.yaml`
2. Run `make gen-api`
3. Update the handler implementation to match the new interface
4. Update tests
5. Run `make verify-codegen`

## Events and Streams (AsyncAPI)

Event-driven interfaces (SSE, webhooks, future WebSocket) are documented in `asyncapi.yaml` using AsyncAPI 3.0.

### Current Channels

| Channel | Direction | Protocol | Description |
|---------|-----------|----------|-------------|
| `/-/events` | Server → Client | SSE | Real-time agent status updates |
| `/webhook` | External → Server | HTTP POST | Gitea webhook event payloads |

### Adding a New Event Type

1. **Edit the schema** — Add the channel, message, and payload schema to `asyncapi.yaml`
2. **Implement the handler** — Write Go code matching the documented payload schema
3. **Write tests** — Test event serialization and deserialization

### AsyncAPI Code Generation

Code generation for AsyncAPI is not yet implemented. The schema currently serves as documentation for event contracts. Contributors should ensure their implementations match the schemas defined in `asyncapi.yaml`.

## Non-Standard Responses

Some endpoints return non-JSON content. These are documented in the schemas but implemented manually:

- **SVG badges** (`/-/api/badges/*`): OpenAPI documents these with `content: image/svg+xml`. The SVG rendering is hand-written.
- **SSE streams** (`/-/events`): AsyncAPI documents the channel and message types. The SSE handler uses `text/event-stream` with chunked encoding.
- **Webhook ingestion** (`/webhook`): AsyncAPI documents consumed Gitea event payloads. Parsing uses Gitea's type definitions.

## Quick Reference

```bash
# Regenerate Go code from OpenAPI schema
make gen-api

# Verify generated code matches schema (CI gate)
make verify-codegen

# Run all tests including API handler tests
make test
```
