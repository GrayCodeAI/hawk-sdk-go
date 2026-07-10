<p align="center">
  <h1 align="center">Hawk SDK for Go</h1>
  <p align="center">
    <strong>Dependency-free Go client for the Hawk daemon API</strong>
  </p>
  <p align="center">
    <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License"></a>
    <a href="https://github.com/GrayCodeAI/hawk-sdk-go/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/GrayCodeAI/hawk-sdk-go/ci.yml?style=flat-square&label=tests" alt="CI"></a>
    <a href="CONTRIBUTING.md"><img src="https://img.shields.io/badge/Contributing-guidelines-orange?style=flat-square" alt="Contributing"></a>
  </p>
</p>

---

Hawk SDK for Go is the official Go client library for the [Hawk](https://github.com/GrayCodeAI/hawk) daemon API. It provides a dependency-free, type-safe client for interacting with Hawk's HTTP API from Go applications.

## Ecosystem

Hawk SDK for Go is part of the [hawk-eco](https://github.com/GrayCodeAI/hawk-eco) mono-ecosystem:

| Component | Purpose |
|-----------|---------|
| **hawk** | AI-powered coding agent for the terminal |
| **hawk-sdk-go** | Go SDK for the Hawk daemon API |
| **hawk-sdk-python** | Python SDK for the Hawk daemon API |
| **hawk-core-contracts** | Shared cross-repo contracts (types, events, tools) |

## Architecture

The SDK follows a clean separation of concerns:

```
hawk-sdk-go/
├── client.go           # Main Client with HTTP transport and functional options
├── types.go            # API types and request/response structs
├── errors.go           # Typed error hierarchy with status code mapping
├── retry.go            # Exponential backoff with Retry-After support
├── stream.go           # SSE StreamReader for streaming responses
├── stream_helpers.go   # Text and tool call collectors from streams
├── agent.go            # Higher-level Agent abstraction with conversation management
├── tools.go            # Tool definition types
├── workflow.go         # Workflow engine types
├── version.go          # SDK version constant
├── api/
│   └── openapi.yaml    # API surface reference (OpenAPI 3.1)
├── docs/
│   └── architecture.md # Detailed architecture documentation
├── examples/           # Runnable examples
└── *_test.go           # Comprehensive test suite
```

## Key Design Decisions

- **Zero runtime dependencies:** Zero third-party runtime imports, pure Go standard library (build-time `oapi-codegen` tooling is gated behind `//go:build tools` and does not affect the runtime module graph)
- **Idiomatic Go:** Follows Go conventions, error handling patterns, and naming
- **Local-only:** Designed for developers running Hawk locally on their machine
- **Single package:** All exported symbols in the `hawksdk` package
- **Functional options:** Client configuration via `With*()` functions

## Quick Start

```go
import "github.com/GrayCodeAI/hawk-sdk-go"

// Create client with options
client := hawksdk.New(
    hawksdk.WithBaseURL("http://localhost:4590"),
    hawksdk.WithAPIKey("sk-..."),
    hawksdk.WithRetry(hawksdk.DefaultRetryConfig()),
)

// Health check
ctx := context.Background()
health, err := client.Health(ctx)
if err != nil {
    log.Fatalf("health check failed: %v", err)
}
fmt.Printf("Daemon version: %s\n", health.Version)

// Send a chat message
resp, err := client.Chat(ctx, hawksdk.ChatRequest{
    Prompt:   "Explain what a closure is in Go",
    Model:    "claude-opus-4-6",
    MaxTurns: 5,
})
if err != nil {
    log.Fatalf("chat failed: %v", err)
}
fmt.Printf("Response: %s\n", resp.Response)

// Stream a response
stream, err := client.ChatStream(ctx, hawksdk.ChatRequest{
    Prompt:   "Analyze this repository",
    Autonomy: "high",
})
if err != nil {
    log.Fatalf("stream failed: %v", err)
}
defer stream.Close()

for {
    event, err := stream.Next()
    if err != nil {
        break // io.EOF or error
    }
    fmt.Print(event.Data)
}
```

## API Reference

### Client Constructor

```go
hawksdk.New(opts ...ClientOption) *Client
```

#### Options

| Option | Description |
|--------|-------------|
| `WithBaseURL(url string)` | Set daemon base URL (default: `http://127.0.0.1:4590`) |
| `WithHTTPClient(client *http.Client)` | Use custom HTTP client |
| `WithAPIKey(key string)` | Set API key for authentication |
| `WithRetry(cfg RetryConfig)` | Enable automatic retries with backoff |

### Client Methods

| Method | Description |
|--------|-------------|
| `Health(ctx) (*HealthResponse, error)` | Check daemon health and version |
| `Chat(ctx, req ChatRequest) (*ChatResponse, error)` | Send a chat message |
| `ChatStream(ctx, req ChatRequest) (*StreamReader, error)` | Stream chat responses via SSE |
| `Sessions(ctx) ([]SessionSummary, error)` | List active sessions |
| `Session(ctx, id string) (*SessionDetail, error)` | Get session details |
| `Messages(ctx, sessionID, opts *ListOptions) (*PaginatedResponse[Message], error)` | Get paginated messages |
| `DeleteSession(ctx, id string) error` | Delete a session |
| `Stats(ctx) (*StatsResponse, error)` | Get aggregated usage stats |

### API Types

| Type | Description |
|------|-------------|
| `ChatRequest` | Request body for `/v1/chat` |
| `ChatResponse` | Response from `/v1/chat` |
| `HealthResponse` | Response from `/v1/health` |
| `SessionSummary` | Session listing entry |
| `SessionDetail` | Full session information |
| `Message` | Conversation message with tool calls |
| `StatsResponse` | Aggregated usage statistics |

### Error Types

| Type | HTTP Status |
|------|:-----------:|
| `APIError` | Base error type |
| `BadRequestError` | 400 |
| `AuthenticationError` | 401 |
| `ForbiddenError` | 403 |
| `NotFoundError` | 404 |
| `InternalServerError` | 500 |
| `ServiceUnavailableError` | 503 |
| `RateLimitError` | 429 |

### Retry Configuration

```go
type RetryConfig struct {
    MaxRetries        int
    InitialBackoff    time.Duration
    MaxBackoff        time.Duration
    BackoffMultiplier float64
    RetryableStatusCodes []int
}
```

## Ecosystem Boundaries

- `hawk-sdk-go` is a **consumer** of Hawk public APIs and contracts
- **Do not** import support engine repos: `eyrie`, `yaad`, `tok`, `trace`, `sight`, or `inspect`
- **Do not** import `hawk/internal/*` or removed legacy paths
- Cross-repo shared types come from Hawk public surfaces or `hawk-core-contracts`

## Development

### Build & Test

```bash
go build ./...                    # Build library
go test ./...                    # Run tests
go test -race ./...              # Race detector
go test -coverprofile=c.out ./... # Coverage
go vet ./...                     # Static analysis
gofumpt -w .                     # Format
go mod tidy                      # Tidy modules
```

### Running Examples

```bash
cd examples/basic
go run .
```

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

## License

MIT - see [LICENSE](LICENSE) for details.
