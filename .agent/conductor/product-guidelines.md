# Product Guidelines

## Voice and Tone

Professional and technical. Documentation and CLI output should be precise, detailed, and assume technical competence from the user.

## Design Principles

1. **Simplicity over features** — One command does the right thing. Minimize required flags and configuration.
2. **Local-first and private** — Everything runs on the user's machine. No external services, no data leaves the workstation.
3. **Convention over configuration** — Sensible defaults for ports, paths, and behavior. Override only when needed.
4. **Reliability and observability** — Clear status output, structured logs, and actionable error messages. The user should always know what's happening.
5. **Schema-first APIs** — All inter-process and client-server communication is defined by machine-readable schemas (OpenAPI for REST/HTTP, AsyncAPI for events/streams) before implementation. Server stubs, client code, and models are generated from schemas, never hand-written. No new HTTP endpoint may be added without first updating the OpenAPI schema. No new event or message type may be added without first updating the AsyncAPI schema.
