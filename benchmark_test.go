package hawksdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func BenchmarkNewClient(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = New(WithBaseURL("http://localhost:4590"), WithAPIKey("test-key"))
	}
}

func BenchmarkNewClientWithRetry(b *testing.B) {
	b.ReportAllocs()
	cfg := DefaultRetryConfig()
	for b.Loop() {
		_ = New(WithBaseURL("http://localhost:4590"), WithAPIKey("test-key"), WithRetry(cfg))
	}
}

func BenchmarkHealth(b *testing.B) {
	resp := HealthResponse{Status: "ok", Version: "0.1.0", Uptime: "1h", Sessions: 5, StartedAt: "2024-01-01T00:00:00Z"}
	payload, _ := json.Marshal(resp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := c.Health(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChat(b *testing.B) {
	resp := ChatResponse{SessionID: "s-1", Response: "Hi there!", TokensIn: 10, TokensOut: 5, TurnsTaken: 1, Duration: "0.5s"}
	payload, _ := json.Marshal(resp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	ctx := context.Background()
	req := ChatRequest{Prompt: "hello"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := c.Chat(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChatStream(b *testing.B) {
	ssePayload := "data: Hello\n\ndata: World\n\nevent: done\ndata: {}\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ssePayload)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		stream, err := c.ChatStream(ctx, ChatRequest{Prompt: "hi"})
		if err != nil {
			b.Fatal(err)
		}
		for {
			_, err := stream.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatal(err)
			}
		}
		stream.Close()
	}
}

func BenchmarkCollectText(b *testing.B) {
	chunks := 50
	var sb strings.Builder
	for i := 0; i < chunks; i++ {
		fmt.Fprintf(&sb, "data: {\"content\":\"chunk-%d\"}\n\n", i)
	}
	sb.WriteString("event: done\ndata: {}\n\n")
	ssePayload := sb.String()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ssePayload)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		stream, err := c.ChatStream(ctx, ChatRequest{Prompt: "hi"})
		if err != nil {
			b.Fatal(err)
		}
		_, err = stream.CollectText(ctx)
		if err != nil {
			b.Fatal(err)
		}
		stream.Close()
	}
}

func BenchmarkPaginationParams(b *testing.B) {
	opts := &ListOptions{Offset: 100, Limit: 25}
	b.ReportAllocs()
	for b.Loop() {
		_ = paginationParams(opts)
	}
}

func BenchmarkPaginationParamsNil(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = paginationParams(nil)
	}
}

func BenchmarkParseAPIError(b *testing.B) {
	errBody := `{"error":"not found","code":"not_found","details":"session gone"}`

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		resp := &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader(errBody)),
		}
		_ = parseAPIError(resp)
	}
}

func BenchmarkParseRetryAfter(b *testing.B) {
	b.Run("seconds", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = parseRetryAfter("30")
		}
	})
	b.Run("empty", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = parseRetryAfter("")
		}
	})
}

func BenchmarkRetryBackoff(b *testing.B) {
	cfg := DefaultRetryConfig()
	b.ReportAllocs()
	for b.Loop() {
		_ = cfg.backoffDuration(2)
	}
}

func BenchmarkRetryIsRetryable(b *testing.B) {
	cfg := DefaultRetryConfig()
	b.ReportAllocs()
	for b.Loop() {
		_ = cfg.isRetryable(http.StatusServiceUnavailable)
		_ = cfg.isRetryable(http.StatusOK)
	}
}

func BenchmarkClassifyEvent(b *testing.B) {
	events := []*StreamEvent{
		{Event: "content", Data: `{"content":"hello"}`},
		{Event: "done", Data: "{}"},
		{Event: "error", Data: "timeout"},
		{Event: "tool_call", Data: `{"id":"tc-1","name":"read","arguments":"{}"}`},
		{Event: "", Data: "raw text"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, ev := range events {
			_ = classifyEvent(ev)
		}
	}
}

func BenchmarkWorkflowRun(b *testing.B) {
	wf, _ := NewWorkflow().
		Step("step1", func(_ context.Context, input any) (any, error) {
			return input.(int) + 1, nil
		}).
		Step("step2", func(_ context.Context, input any) (any, error) {
			return input.(int) * 2, nil
		}).
		Step("step3", func(_ context.Context, input any) (any, error) {
			return input.(int) + 10, nil
		}).
		Build()

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := wf.Run(ctx, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChatWithTools(b *testing.B) {
	round := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		round++
		if round%2 == 1 {
			json.NewEncoder(w).Encode(ChatWithToolsResponse{
				ChatResponse: ChatResponse{SessionID: "s-1", Response: ""},
				ToolCalls:    []ToolCall{{ID: "tc-1", Name: "echo", Arguments: map[string]interface{}{"msg": "hi"}}},
				FinishReason: "tool_calls",
			})
		} else {
			json.NewEncoder(w).Encode(ChatWithToolsResponse{
				ChatResponse: ChatResponse{SessionID: "s-1", Response: "done"},
				FinishReason: "stop",
			})
		}
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	tools := []Tool{
		{
			Schema: ToolSchema{Name: "echo", Description: "echo", Parameters: json.RawMessage(`{"type":"object"}`)},
			Run:    func(_ context.Context, args map[string]interface{}) (string, error) { return "echoed", nil },
		},
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := c.ChatWithTools(ctx, ChatRequest{Prompt: "hi"}, tools, 5)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewAgent(b *testing.B) {
	c := New()
	cfg := AgentConfig{
		Model:     "test-model",
		MaxRounds: 10,
		Memory:    &MemoryConfig{Enabled: true, SessionID: "s-1", MaxMessages: 100},
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = NewAgent(c, cfg)
	}
}

func BenchmarkAgentChat(b *testing.B) {
	payload, _ := json.Marshal(ChatResponse{SessionID: "s-1", Response: "ok"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	agent := NewAgent(c, AgentConfig{Model: "test"})
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := agent.Chat(ctx, "hello")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUserAgent(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = userAgent()
	}
}
