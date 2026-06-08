<div align="center">

# 📦 hawk-sdk-go Architecture

**Go SDK for the Hawk Daemon API**

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![Dependency](https://img.shields.io/badge/Dependencies-Zero-green)]()

</div>

---

## 🎯 Overview

Dependency-free Go client for the hawk daemon HTTP API. Exposes idiomatic Go types for **chat**, **streaming**, **sessions**, **messages**, and **stats** with automatic retry and typed errors.

---

## 🧱 Components

```
hawk-sdk-go/
├── api/openapi.yaml     📜 SDK method surface reference
├── client.go            🔌 Client, New(), With* options, HTTP transport
├── types.go             📋 API types (ChatRequest, ChatResponse, Session, Message, Stats)
├── errors.go            ❌ APIError base + typed subclasses
├── retry.go             🔄 RetryConfig, doWithRetry(), sleepWithContext()
├── stream.go            📡 StreamReader, SSE parsing
├── stream_helpers.go    🛠️ collect_text(), collect_tool_calls()
├── agent.go             🤖 Agent (conversation history, mutex-safe session ID)
├── tools.go             🛠️ Tool definitions
├── workflow.go          🔧 Workflow engine
├── version.go           🏷️ Version constant
└── *_test.go            🧪 Test files (httptest-based)
```

---

## 📤 Client Usage

```go
import hawksdk "github.com/GrayCodeAI/hawk-sdk-go"

c := hawksdk.New(
    hawksdk.WithBaseURL("http://localhost:4590"),
    hawksdk.WithAPIKey("sk-..."),
    hawksdk.WithRetry(hawksdk.DefaultRetryConfig()),
)

// 🩺 Health check
health, err := c.Health(ctx)

// 💬 Non-streaming chat
resp, err := c.Chat(ctx, hawksdk.ChatRequest{Message: "list files"})

// 📡 Streaming chat
stream, err := c.ChatStream(ctx, hawksdk.ChatRequest{Message: "explain this code"})
defer stream.Close()
for { ev, err := stream.Next(); if err != nil { break }; fmt.Print(ev.Data) }

// 📋 Sessions
sessions, _ := c.Sessions(ctx, hawksdk.ListOptions{Limit: 10})
msgs, _     := c.Messages(ctx, sessionID, hawksdk.ListOptions{})
_            = c.DeleteSession(ctx, sessionID)

// 📊 Stats
stats, _ := c.Stats(ctx)
```

---

## 🤖 Agent (Higher-Level)

```go
agent := hawksdk.NewAgent(c, hawksdk.AgentConfig{SystemPrompt: "You are a Go expert"})
resp, _ := agent.Chat(ctx, "refactor this function")
// Subsequent calls automatically continue the same session
```

---

## ❌ Error Handling

```go
var notFound *hawksdk.NotFoundError
var rateLimit *hawksdk.RateLimitError

if errors.As(err, &notFound)   { /* handle 404 */ }
if errors.As(err, &rateLimit)  { time.Sleep(rateLimit.RetryAfter) }
```

| Error Type | HTTP Status |
|------------|:-----------:|
| `NotFoundError` | 404 |
| `RateLimitError` | 429 |
| `InternalServerError` | 500 |

---

## 🔌 Connecting

The daemon must be running: `hawk daemon start`

| | |
|---|---|
| **Default URL** | `http://127.0.0.1:4590` |
| **Override** | `WithBaseURL()` or `HAWK_BASE_URL` env |
