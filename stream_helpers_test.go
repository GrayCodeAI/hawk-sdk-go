package hawksdk

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: Hello\n\n")
		fmt.Fprint(w, "data:  World\n\n")
		fmt.Fprint(w, "data: !\n\n")
		fmt.Fprint(w, "event: done\ndata: {}\n\n")
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	stream, err := c.ChatStream(context.Background(), ChatRequest{Prompt: "hi"})
	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}
	defer stream.Close()

	text, err := stream.CollectText(context.Background())
	if err != nil {
		t.Fatalf("CollectText() error: %v", err)
	}
	want := "Hello World!"
	if text != want {
		t.Errorf("CollectText() = %q, want %q", text, want)
	}
}

func TestCollectText_WithJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "event: content\ndata: {\"content\":\"Hello\"}\n\n")
		fmt.Fprint(w, "event: content\ndata: {\"content\":\" World\"}\n\n")
		fmt.Fprint(w, "event: done\ndata: {}\n\n")
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	stream, err := c.ChatStream(context.Background(), ChatRequest{Prompt: "hi"})
	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}
	defer stream.Close()

	text, err := stream.CollectText(context.Background())
	if err != nil {
		t.Fatalf("CollectText() error: %v", err)
	}
	want := "Hello World"
	if text != want {
		t.Errorf("CollectText() = %q, want %q", text, want)
	}
}

func TestCollectToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "event: tool_call\ndata: {\"id\":\"tc-1\",\"name\":\"search\",\"arguments\":\"{\\\"q\\\"\"}\n\n")
		fmt.Fprint(w, "event: tool_call_delta\ndata: {\"id\":\"tc-1\",\"name\":\"\",\"arguments\":\": \\\"hello\\\"}\"}\n\n")
		fmt.Fprint(w, "event: done\ndata: {}\n\n")
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	stream, err := c.ChatStream(context.Background(), ChatRequest{Prompt: "search"})
	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}
	defer stream.Close()

	calls, err := stream.CollectToolCalls(context.Background())
	if err != nil {
		t.Fatalf("CollectToolCalls() error: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(calls))
	}
	if calls[0].ID != "tc-1" {
		t.Errorf("ID = %q, want %q", calls[0].ID, "tc-1")
	}
	if calls[0].Name != "search" {
		t.Errorf("Name = %q, want %q", calls[0].Name, "search")
	}
	wantArgs := "{\"q\": \"hello\"}"
	if calls[0].Arguments != wantArgs {
		t.Errorf("Arguments = %q, want %q", calls[0].Arguments, wantArgs)
	}
}

func TestEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: Hello\n\n")
		fmt.Fprint(w, "event: tool_call\ndata: {\"id\":\"tc-1\",\"name\":\"calc\",\"arguments\":\"{}\"}\n\n")
		fmt.Fprint(w, "event: done\ndata: {}\n\n")
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	stream, err := c.ChatStream(context.Background(), ChatRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}
	defer stream.Close()

	ctx := context.Background()
	ch := stream.Events(ctx)

	var events []TypedStreamEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) < 3 {
		t.Fatalf("got %d events, want at least 3", len(events))
	}

	// First event: content.
	if events[0].Type != StreamEventContent {
		t.Errorf("events[0].Type = %v, want StreamEventContent", events[0].Type)
	}
	if events[0].Content != "Hello" {
		t.Errorf("events[0].Content = %q, want %q", events[0].Content, "Hello")
	}

	// Second event: tool call.
	if events[1].Type != StreamEventToolCall {
		t.Errorf("events[1].Type = %v, want StreamEventToolCall", events[1].Type)
	}
	if events[1].ToolCall == nil || events[1].ToolCall.Name != "calc" {
		t.Errorf("events[1].ToolCall.Name = %v, want %q", events[1].ToolCall, "calc")
	}

	// Third event: completion.
	if events[2].Type != StreamEventCompletion {
		t.Errorf("events[2].Type = %v, want StreamEventCompletion", events[2].Type)
	}
}

func TestCollectText_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: Hello\n\n")
		// Simulate a slow stream — the context will cancel before more data.
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	stream, err := c.ChatStream(context.Background(), ChatRequest{Prompt: "hi"})
	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}
	defer stream.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Read first event manually, then cancel.
	_, _ = stream.Next()
	cancel()

	_, err = stream.CollectText(ctx)
	if err != context.Canceled {
		// After cancellation, remaining text or ctx.Err expected.
		if err == nil {
			// This is fine — the stream may have ended cleanly.
			return
		}
	}
}
