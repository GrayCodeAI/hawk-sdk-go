---
description: hawk-sdk-go — Go SDK build and test conventions.
globs: "*.go,*.yaml"
alwaysApply: false
---

# hawk-sdk-go Conventions

Official Go client for the Hawk daemon API.

## Build & Test

```bash
go build ./...                    # Build library
go test ./...                     # Run tests
go test -race ./...               # Race detector
go test -coverprofile=c.out ./...  # Coverage
go vet ./...                      # Static analysis
gofumpt -w .                      # Format
go mod tidy                       # Tidy modules
```

## Design Principles

- Dependency-free: zero third-party runtime imports, pure Go standard library
- `oapi-codegen` is a build-time tool only (see `codegen_tools.go`)
- Local-only: designed for developers running Hawk locally

## Ecosystem Boundaries

- Consumer of Hawk public APIs and `hawk-core-contracts`
- Do not import support engine repos (`eyrie`, `yaad`, `tok`, `trace`, `sight`, `inspect`)

For full hawk-eco extension guidelines, see [hawk/AGENTS.md](https://github.com/GrayCodeAI/hawk/blob/main/AGENTS.md).
