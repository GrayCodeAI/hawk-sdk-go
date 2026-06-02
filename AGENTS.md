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

## Naming Conventions

- **Package**: `hawksdk` — all exported symbols live in this single package
- **Types**: PascalCase, noun-based (`Client`, `ChatRequest`, `ChatResponse`, `StreamReader`, `RetryConfig`)
- **Error types**: suffix with `Error` (`APIError`, `NotFoundError`, `RateLimitError`, `InternalServerError`)
- **Constructor options**: `With` prefix, functional options pattern (`WithBaseURL`, `WithHTTPClient`, `WithAPIKey`, `WithRetry`)
- **Methods**: PascalCase for exported, camelCase for unexported (`Health`, `Chat`, `get`, `post`, `doWithRetry`, `setAuth`)
- **JSON tags**: snake_case matching the daemon API (`json:"session_id,omitempty"`, `json:"tokens_in"`)
- **Constants**: camelCase for unexported (`defaultBaseURL`), PascalCase for exported (`Version`)
- **Test functions**: `Test` + method name (`TestHealth`, `TestChat`, `TestChatStream`, `TestHTTPError`, `TestConcurrencyClient`)

## API Patterns

### Functional Options
The `Client` uses the functional options pattern. `ClientOption` is `func(*Client)`. Options like `WithBaseURL`, `WithHTTPClient`, `WithAPIKey`, and `WithRetry` return closures that modify the client:
```go
c := New(
    WithBaseURL("http://localhost:4590"),
    WithAPIKey("sk-..."),
    WithRetry(DefaultRetryConfig()),
)
```

### Error Hierarchy
`errors.go` defines `APIError` as the base struct with `StatusCode`, `Code`, `Message`, `Details`. Each HTTP status gets a wrapper type (`NotFoundError`, `RateLimitError`, etc.) that embeds `APIError`. All error types implement `Error()` and `Unwrap()` for `errors.Is`/`errors.As` compatibility:
```go
var notFound *NotFoundError
if errors.As(err, &notFound) {
    // handle 404
}
```

### Internal Transport
`get()` and `post()` are unexported methods on `*Client` that handle request construction, auth headers, retry, response decoding, and error parsing. All public methods delegate to them. `ChatStream` and `DeleteSession` are exceptions — they need custom response handling.

### Retry with Context
`doWithRetry()` clones the request body for each attempt (using `io.NopCloser(bytes.NewReader(body))`), respects `Retry-After` headers on 429, and uses `sleepWithContext()` to honor context cancellation during backoff. Network errors are retryable; non-retryable status codes pass through immediately.

### Generics
`PaginatedResponse[T any]` uses Go generics. The `ListOptions` struct configures pagination. `Sessions()`, `Messages()` return `*PaginatedResponse[T]`.

### Agent Thread Safety
`Agent` uses `sync.Mutex` to protect `sessionID` from concurrent reads/writes. The mutex is locked in `Chat()` and `ChatStream()`. `SessionID()` also acquires the lock.

## Testing Patterns

### httptest Servers
All tests use `net/http/httptest.NewServer` to spin up local HTTP servers. Each test registers a `HandlerFunc` that checks the request path, method, and body, then returns a JSON-encoded response:
```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/v1/health" { t.Errorf(...) }
    json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}))
defer srv.Close()
c := New(WithBaseURL(srv.URL))
```

### Typed Error Verification
Tests use `errors.As` to verify error types, and check the `Error()` string format:
```go
var notFound *NotFoundError
if !errors.As(err, &notFound) { t.Error("expected NotFoundError") }
```

### Concurrency Tests
`TestConcurrencyClient` and `TestConcurrencyAgent` spawn 20 goroutines hitting the same client/agent simultaneously, using `sync.WaitGroup` and error channels. This verifies thread safety of internal state.

### Network Failure Tests
`TestNetworkFailure` and `TestNetworkFailureAllMethods` use an unreachable port (`127.0.0.1:1`) with `MaxRetries: 0` to verify graceful error handling for all HTTP methods.

### SSE Stream Tests
`TestChatStream` writes SSE-formatted data (`"data: Hello\n\n"`) and verifies `stream.Next()` returns correct events, terminating with `io.EOF`.

## Key File Locations

| What | Where |
|------|-------|
| Client + transport | `client.go` |
| API types + JSON tags | `types.go` |
| Error types + parser | `errors.go` |
| Retry + backoff | `retry.go` |
| SSE streaming | `stream.go` |
| Stream helpers | `stream_helpers.go` |
| Agent abstraction | `agent.go` |
| Tool definitions | `tools.go` |
| Workflow engine | `workflow.go` |
| Version constant | `version.go` |
| Client tests | `client_test.go` |
| Agent tests | `agent_test.go` |
| Stream tests | `stream_test.go` |
| Error tests | `errors_test.go` |
| Retry tests | `retry_test.go` |
| Workflow tests | `workflow_test.go` |
| Tool tests | `tools_test.go` |
| Session tests | `sessions_test.go` |

## Refactoring Guidelines

- **Safe to refactor**: `get()`/`post()` internals — they are unexported and all public methods go through them
- **Do not change**: `APIError.Error()` format string — tests assert exact string output
- **Do not change**: JSON struct tags — they match the daemon's API contract
- **Do not change**: `Unwrap()` implementations — `errors.As` depends on them for type matching
- **Safe to extend**: add new error types by creating a struct embedding `APIError`, adding to `parseAPIError` switch, and implementing `Error()` + `Unwrap()`
- **Safe to extend**: add new client methods by following the `get()`/`post()` delegation pattern
- **When adding streaming endpoints**: follow `ChatStream` pattern — set `Accept: text/event-stream`, check status before wrapping in `newStreamReader`
- **Concurrency**: any new mutable state on `Agent` must be protected by `a.mu`
