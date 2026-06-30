package hawksdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func TestAgent_Chat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Model != "claude-opus-4-6" {
			t.Errorf("Model = %q, want %q", req.Model, "claude-opus-4-6")
		}

		json.NewEncoder(w).Encode(ChatResponse{
			SessionID: "agent-sess-1",
			Response:  "I'm your agent!",
			TokensIn:  20,
			TokensOut: 10,
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	agent := NewAgent(c, AgentConfig{
		Model:     "claude-opus-4-6",
		MaxRounds: 5,
	})

	resp, err := agent.Chat(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Response != "I'm your agent!" {
		t.Errorf("Response = %q, want %q", resp.Response, "I'm your agent!")
	}
	if agent.SessionID() != "agent-sess-1" {
		t.Errorf("SessionID = %q, want %q", agent.SessionID(), "agent-sess-1")
	}
}

func TestAgent_ChatWithTools(t *testing.T) {
	var round int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&round, 1)
		if n == 1 {
			json.NewEncoder(w).Encode(ChatWithToolsResponse{
				ChatResponse: ChatResponse{SessionID: "s-tools"},
				ToolCalls: []ToolCall{
					{ID: "tc-1", Name: "greet", Arguments: map[string]interface{}{"name": "world"}},
				},
			})
		} else {
			json.NewEncoder(w).Encode(ChatWithToolsResponse{
				ChatResponse: ChatResponse{
					SessionID: "s-tools",
					Response:  "Hello, world!",
				},
				FinishReason: "stop",
			})
		}
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	agent := NewAgent(c, AgentConfig{
		Model: "claude-opus-4-6",
		Tools: []Tool{
			{
				Schema: ToolSchema{
					Name:        "greet",
					Description: "Greets someone",
					Parameters:  json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
				},
				Run: func(ctx context.Context, args map[string]interface{}) (string, error) {
					var p struct{ Name string }
					b, _ := json.Marshal(args)
					json.Unmarshal(b, &p)
					return "Hello, " + p.Name + "!", nil
				},
			},
		},
		MaxRounds: 5,
	})

	resp, err := agent.Chat(context.Background(), "greet the world")
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Response != "Hello, world!" {
		t.Errorf("Response = %q, want %q", resp.Response, "Hello, world!")
	}
}

func TestAgent_SessionContinuity(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		n := atomic.AddInt32(&calls, 1)
		if n == 2 && req.SessionID != "sess-abc" {
			t.Errorf("second call SessionID = %q, want %q", req.SessionID, "sess-abc")
		}

		json.NewEncoder(w).Encode(ChatResponse{
			SessionID: "sess-abc",
			Response:  "ok",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	agent := NewAgent(c, AgentConfig{Model: "test"})

	// First call establishes session.
	_, err := agent.Chat(context.Background(), "first")
	if err != nil {
		t.Fatalf("Chat(1) error: %v", err)
	}

	// Second call should send the session ID.
	_, err = agent.Chat(context.Background(), "second")
	if err != nil {
		t.Fatalf("Chat(2) error: %v", err)
	}
}

func TestAgent_MemorySessionID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.SessionID != "existing-sess" {
			t.Errorf("SessionID = %q, want %q", req.SessionID, "existing-sess")
		}

		json.NewEncoder(w).Encode(ChatResponse{
			SessionID: "existing-sess",
			Response:  "resumed",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	agent := NewAgent(c, AgentConfig{
		Model: "test",
		Memory: &MemoryConfig{
			Enabled:   true,
			SessionID: "existing-sess",
		},
	})

	resp, err := agent.Chat(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Response != "resumed" {
		t.Errorf("Response = %q, want %q", resp.Response, "resumed")
	}
}

func TestAgent_ChatStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Accept = %q, want text/event-stream", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: streaming\n\n"))
		w.Write([]byte("event: done\ndata: {}\n\n"))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	agent := NewAgent(c, AgentConfig{Model: "test"})

	stream, err := agent.ChatStream(context.Background(), "hello")
	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}
	defer stream.Close()

	ev, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev.Data != "streaming" {
		t.Errorf("Data = %q, want %q", ev.Data, "streaming")
	}
}

// TestAgent_ConcurrentChatAndStream runs Chat and ChatStream concurrently
// (run with -race) to verify that ChatStream's session ID snapshot is not
// affected by concurrent Chat calls mutating a.sessionID, and that no data
// race exists between request building and session updates.
func TestAgent_ConcurrentChatAndStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") == "text/event-stream" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("data: chunk\n\n"))
			w.Write([]byte("event: done\ndata: {}\n\n"))
			return
		}
		json.NewEncoder(w).Encode(ChatResponse{
			SessionID: "race-sess",
			Response:  "ok",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	agent := NewAgent(c, AgentConfig{Model: "test"})

	const iterations = 10
	var wg sync.WaitGroup
	errs := make(chan error, iterations*2)

	for i := 0; i < iterations; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, err := agent.Chat(context.Background(), "hello"); err != nil {
				errs <- err
			}
		}()
		go func() {
			defer wg.Done()
			stream, err := agent.ChatStream(context.Background(), "stream hello")
			if err != nil {
				errs <- err
				return
			}
			defer stream.Close()
			// Consume the whole stream while Chat calls mutate sessionID.
			if _, err := stream.CollectText(context.Background()); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent Chat/ChatStream error: %v", err)
	}

	if got := agent.SessionID(); got != "race-sess" {
		t.Errorf("SessionID = %q, want %q", got, "race-sess")
	}
}

func TestNewAgent_Defaults(t *testing.T) {
	c := New()
	agent := NewAgent(c, AgentConfig{})

	if agent.SessionID() != "" {
		t.Errorf("SessionID should be empty initially, got %q", agent.SessionID())
	}
}
