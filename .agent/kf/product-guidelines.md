# Product Guidelines

## Voice and Tone

Professional and technical. Documentation and CLI output should be precise, detailed, and assume technical competence from the user.

## Design Principles

1. **Simplicity over features** — One command does the right thing. Minimize required flags and configuration.
2. **Local-first and private** — Everything runs on the user's machine. No external services, no data leaves the workstation.
3. **Convention over configuration** — Sensible defaults for ports, paths, and behavior. Override only when needed.
4. **Reliability and observability** — Clear status output, structured logs, and actionable error messages. The user should always know what's happening.
5. **Schema-first APIs** — All inter-process and client-server communication is defined by machine-readable schemas (OpenAPI for REST/HTTP, AsyncAPI for events/streams) before implementation. Server stubs, client code, and models are generated from schemas, never hand-written. No new HTTP endpoint may be added without first updating the OpenAPI schema. No new event or message type may be added without first updating the AsyncAPI schema.
6. **Makefile as build entry point** — All builds, tests, and code generation go through the Makefile. See `code_styleguides/build.md` for conventions on frontend embedding, output directories, and VCS stamping.
7. **Thin adapters, shared domain logic** — CLI commands and REST handlers are thin adapters that convert input into domain commands/queries and dispatch to the service layer. Business logic lives exclusively in `core/service/`. Adapters never access stores directly — they receive services via constructor injection. This ensures consistent behavior regardless of entry point (CLI, API, webhook). When adding a new operation, implement it as a service method first, then wire it from both CLI and API adapters.
