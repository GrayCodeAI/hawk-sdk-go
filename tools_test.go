package hawksdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestChatWithTools_NoToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ChatWithToolsResponse{
			ChatResponse: ChatResponse{
				SessionID: "s-1",
				Response:  "Hello!",
			},
			FinishReason: "stop",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	tools := []Tool{
		{
			Schema: ToolSchema{
				Name:        "greet",
				Description: "Says hello",
				Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
			},
			Run: func(ctx context.Context, args string) (string, error) {
				return "hi", nil
			},
		},
	}

	resp, err := c.ChatWithTools(context.Background(), ChatRequest{Prompt: "hello"}, tools, 5)
	if err != nil {
		t.Fatalf("ChatWithTools() error: %v", err)
	}
	if resp.Response != "Hello!" {
		t.Errorf("Response = %q, want %q", resp.Response, "Hello!")
	}
}

func TestChatWithTools_ExecutesTools(t *testing.T) {
	var round int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&round, 1)
		if n == 1 {
			// First round: request tool call.
			json.NewEncoder(w).Encode(ChatWithToolsResponse{
				ChatResponse: ChatResponse{SessionID: "s-1", Response: ""},
				ToolCalls: []ToolCall{
					{ID: "tc-1", Name: "add", Arguments: `{"a":1,"b":2}`},
				},
				FinishReason: "tool_calls",
			})
		} else {
			// Second round: return final answer.
			json.NewEncoder(w).Encode(ChatWithToolsResponse{
				ChatResponse: ChatResponse{SessionID: "s-1", Response: "The sum is 3"},
				FinishReason: "stop",
			})
		}
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	tools := []Tool{
		{
			Schema: ToolSchema{
				Name:        "add",
				Description: "Adds two numbers",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}}}`),
			},
			Run: func(ctx context.Context, args string) (string, error) {
				var params struct {
					A int `json:"a"`
					B int `json:"b"`
				}
				json.Unmarshal([]byte(args), &params)
				return "3", nil
			},
		},
	}

	resp, err := c.ChatWithTools(context.Background(), ChatRequest{Prompt: "what is 1+2?"}, tools, 5)
	if err != nil {
		t.Fatalf("ChatWithTools() error: %v", err)
	}
	if resp.Response != "The sum is 3" {
		t.Errorf("Response = %q, want %q", resp.Response, "The sum is 3")
	}
	if got := atomic.LoadInt32(&round); got != 2 {
		t.Errorf("rounds = %d, want 2", got)
	}
}

func TestChatWithTools_MaxRoundsExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always request a tool call.
		json.NewEncoder(w).Encode(ChatWithToolsResponse{
			ChatResponse: ChatResponse{SessionID: "s-1", Response: ""},
			ToolCalls: []ToolCall{
				{ID: "tc-1", Name: "loop", Arguments: `{}`},
			},
			FinishReason: "tool_calls",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	tools := []Tool{
		{
			Schema: ToolSchema{Name: "loop", Description: "loops"},
			Run: func(ctx context.Context, args string) (string, error) {
				return "again", nil
			},
		},
	}

	_, err := c.ChatWithTools(context.Background(), ChatRequest{Prompt: "go"}, tools, 3)
	if err == nil {
		t.Fatal("expected max rounds error")
	}
}

func TestChatWithTools_UnknownTool(t *testing.T) {
	var round int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&round, 1)
		if n == 1 {
			json.NewEncoder(w).Encode(ChatWithToolsResponse{
				ChatResponse: ChatResponse{SessionID: "s-1"},
				ToolCalls: []ToolCall{
					{ID: "tc-1", Name: "unknown_tool", Arguments: `{}`},
				},
				FinishReason: "tool_calls",
			})
		} else {
			json.NewEncoder(w).Encode(ChatWithToolsResponse{
				ChatResponse: ChatResponse{SessionID: "s-1", Response: "handled error"},
				FinishReason: "stop",
			})
		}
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	tools := []Tool{
		{
			Schema: ToolSchema{Name: "known", Description: "a known tool"},
			Run: func(ctx context.Context, args string) (string, error) {
				return "ok", nil
			},
		},
	}

	resp, err := c.ChatWithTools(context.Background(), ChatRequest{Prompt: "test"}, tools, 5)
	if err != nil {
		t.Fatalf("ChatWithTools() error: %v", err)
	}
	if resp.Response != "handled error" {
		t.Errorf("Response = %q, want %q", resp.Response, "handled error")
	}
}

func TestChatWithTools_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ChatWithToolsResponse{
			ChatResponse: ChatResponse{SessionID: "s-1"},
			ToolCalls: []ToolCall{
				{ID: "tc-1", Name: "slow", Arguments: `{}`},
			},
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	tools := []Tool{
		{
			Schema: ToolSchema{Name: "slow"},
			Run: func(ctx context.Context, args string) (string, error) {
				return "done", nil
			},
		},
	}

	_, err := c.ChatWithTools(ctx, ChatRequest{Prompt: "test"}, tools, 5)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
