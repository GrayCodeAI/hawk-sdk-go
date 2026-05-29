# AGENTS.md — hawk-sdk-go

Go SDK for the hawk daemon API. Dependency-free client for chat, streaming, sessions, messages, and stats. Built for solo developers running hawk locally.

## Design Principles

- **Dependency-free** — no third-party imports
- **Idiomatic Go** — follows Go conventions and error handling patterns
- **Local-only** — nothing calls third-party services or phones home

## Build & Test

```bash
go test ./...                    # Run all tests
go test -race ./...              # Race detector
go test -coverprofile=c.out ./... # Coverage
go vet ./...                     # Static analysis
gofumpt -w .                     # Format
```

## Architecture

- `client.go` — Main client with HTTP transport
- `chat.go` — Chat and streaming operations
- `session.go` — Session management
- `stats.go` — Aggregated statistics
- `errors.go` — Typed error categories
- `retry.go` — Exponential backoff with Retry-After honoring

## Conventions

- Go 1.26+, pure Go, no CGO
- Table-driven tests
- Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`
- No `Co-authored-by:` trailers
- `gofumpt` formatting enforced in CI
- `AgentConfig` fields must be wired up (see recent security fixes)

## Common Pitfalls

- `Retry-After: 0` is valid — respect it
- Session ID must be set atomically (race condition fixed)
- Default timeout is required — don't leave it zero
- Scanner buffer size matters for large responses
