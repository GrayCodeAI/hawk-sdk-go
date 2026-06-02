<p align="center">
  <h1 align="center">Hawk SDK for Go</h1>
  <p align="center">
    <strong>Official Go client for the Hawk daemon API</strong>
  </p>
  <p align="center">
    <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License"></a>
    <a href="https://github.com/GrayCodeAI/hawk-sdk-go/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/GrayCodeAI/hawk-sdk-go/ci.yml?style=flat-square&label=tests" alt="CI"></a>
  </p>
</p>

---

The Hawk SDK for Go provides a type-safe client for interacting with the [Hawk](https://github.com/GrayCodeAI/hawk) daemon API. It supports chat, streaming, session management, and health checks.

## Features

- **Type-safe API** - Strongly typed requests and responses
- **Streaming support** - Real-time response streaming
- **Session management** - Create and manage chat sessions
- **Health checks** - Monitor daemon status
- **Error handling** - Detailed error types and messages

## Installation

```bash
go get github.com/GrayCodeAI/hawk-sdk-go
```

## Quick Start

```go
import hawksdk "github.com/GrayCodeAI/hawk-sdk-go"

client := hawksdk.New()

// Health check
health, err := client.Health(ctx)
fmt.Printf("Daemon version: %s\n", health.Version)

// Chat
resp, err := client.Chat(ctx, hawksdk.ChatRequest{
    Message: "Explain what a closure is in Go",
})
fmt.Println(resp.Response)
```

## Examples

See the [examples/](examples/) directory for complete runnable examples.

## API Reference

### Client Methods

- `Health(ctx)` - Check daemon health and version
- `Chat(ctx, req)` - Send a chat message
- `ChatStream(ctx, req)` - Stream chat responses
- `Sessions(ctx)` - List active sessions
- `Session(ctx, id)` - Get session details

## License

MIT - see [LICENSE](LICENSE) for details.
