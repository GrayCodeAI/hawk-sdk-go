# hawk-sdk-go

Go SDK for the [hawk](https://github.com/GrayCodeAI/hawk) daemon API.

`hawk-sdk-go` is a small, dependency-free client for the local hawk daemon
HTTP API. It exposes idiomatic Go types for chat, streaming, sessions,
messages, and aggregated stats; supports automatic retry with exponential
backoff and `Retry-After` honouring; and provides typed error categories
for clean error handling.

It is built for **solo developers** running their own hawk daemon locally.
Nothing in this SDK calls a third-party service or phones home.

## Install

```bash
go get github.com/GrayCodeAI/hawk-sdk-go
```

## Usage

### Health check

```go
package main

import (
    "context"
    "fmt"
    "log"

    hawksdk "github.com/GrayCodeAI/hawk-sdk-go"
)

func main() {
    c := hawksdk.New()
    h, err := c.Health(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("daemon ok — version=%s sessions=%d uptime=%s\n",
        h.Version, h.Sessions, h.Uptime)
}
```

### Chat

```go
resp, err := c.Chat(ctx, hawksdk.ChatRequest{
    Message: "summarise this commit",
    Model:   "claude-opus-4-20250514",
})
if err != nil {
    return err
}
fmt.Println(resp.Response)
```

### Streaming

```go
stream, err := c.ChatStream(ctx, hawksdk.ChatRequest{Message: "..."})
if err != nil {
    return err
}
defer stream.Close()

for {
    chunk, err := stream.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    fmt.Print(chunk.Delta)
}
```

### Custom base URL & retry

```go
c := hawksdk.New(
    hawksdk.WithBaseURL("http://hawk.local:4590"),
    hawksdk.WithRetry(hawksdk.DefaultRetryConfig()),
)
```

### Typed errors

```go
resp, err := c.Chat(ctx, req)
if err != nil {
    var apiErr *hawksdk.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.Category {
        case hawksdk.ErrCategoryRateLimited:
            // back off
        case hawksdk.ErrCategoryUnauthorized:
            // re-auth
        case hawksdk.ErrCategoryNotFound:
            // ...
        }
    }
    return err
}
```

## API surface

- `New(opts ...ClientOption) *Client` — constructor with `WithBaseURL`,
  `WithHTTPClient`, `WithRetry`.
- `Client.Health(ctx) (*HealthResponse, error)` — daemon connectivity check.
- `Client.Chat(ctx, ChatRequest) (*ChatResponse, error)` — non-streaming.
- `Client.ChatStream(ctx, ChatRequest) (*StreamReader, error)` — SSE.
- `Client.Sessions(ctx, *ListOptions) (*PaginatedResponse[SessionSummary], error)` — list.
- `Client.Session(ctx, id) (*SessionDetail, error)` — get one.
- `Client.Messages(ctx, sessionID, *ListOptions) (*PaginatedResponse[Message], error)` — list messages.
- `Client.DeleteSession(ctx, id) error` — delete.
- `Client.Stats(ctx) (*StatsResponse, error)` — aggregated stats.
- `Agent` / `Tool` / `Workflow` — higher-level orchestration helpers.

The full reference is on [pkg.go.dev](https://pkg.go.dev/github.com/GrayCodeAI/hawk-sdk-go).

## Versioning & compatibility

This SDK adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
The `Version` constant is reported in the `User-Agent` header
(`hawk-sdk-go/0.2.0`) so daemon operators can identify SDK clients in logs.

The SDK targets daemon API version `v1`. Breaking changes to the daemon API
will be tracked in `CHANGELOG.md` with a clear migration path.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). All commits must follow
[Conventional Commits](https://www.conventionalcommits.org/) and must not
include `Co-authored-by:` trailers.

## License

[MIT](LICENSE)
