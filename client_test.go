package hawksdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(HealthResponse{
			Status:    "ok",
			Version:   "0.3.0",
			Uptime:    "1h30m",
			Sessions:  2,
			StartedAt: "2024-01-01T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("Status = %q, want %q", resp.Status, "ok")
	}
	if resp.Sessions != 2 {
		t.Errorf("Sessions = %d, want 2", resp.Sessions)
	}
}

func TestChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Prompt != "hello" {
			t.Errorf("Prompt = %q, want %q", req.Prompt, "hello")
		}

		json.NewEncoder(w).Encode(ChatResponse{
			SessionID:  "sess-123",
			Response:   "Hi there!",
			TokensIn:   10,
			TokensOut:  5,
			TurnsTaken: 1,
			Duration:   "1.2s",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := c.Chat(context.Background(), ChatRequest{Prompt: "hello"})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Response != "Hi there!" {
		t.Errorf("Response = %q, want %q", resp.Response, "Hi there!")
	}
	if resp.SessionID != "sess-123" {
		t.Errorf("SessionID = %q, want %q", resp.SessionID, "sess-123")
	}
}

func TestChatStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Accept header = %q, want text/event-stream", r.Header.Get("Accept"))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: Hello\n\n")
		fmt.Fprint(w, "data: World\n\n")
		fmt.Fprint(w, "event: done\ndata: {}\n\n")
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	stream, err := c.ChatStream(context.Background(), ChatRequest{Prompt: "hi"})
	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}
	defer stream.Close()

	ev1, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev1.Data != "Hello" {
		t.Errorf("event 1 Data = %q, want %q", ev1.Data, "Hello")
	}

	ev2, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev2.Data != "World" {
		t.Errorf("event 2 Data = %q, want %q", ev2.Data, "World")
	}

	ev3, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev3.Event != "done" {
		t.Errorf("event 3 Event = %q, want %q", ev3.Event, "done")
	}

	_, err = stream.Next()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/abc-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(SessionDetail{
			ID:           "abc-123",
			Model:        "claude-opus-4-6",
			Provider:     "anthropic",
			MessageCount: 42,
			ToolCalls:    10,
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := c.Session(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("Session() error: %v", err)
	}
	if resp.ID != "abc-123" {
		t.Errorf("ID = %q, want %q", resp.ID, "abc-123")
	}
	if resp.MessageCount != 42 {
		t.Errorf("MessageCount = %d, want 42", resp.MessageCount)
	}
}

func TestStats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/stats" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(StatsResponse{
			TotalSessions:  100,
			TotalMessages:  500,
			TotalToolCalls: 200,
			TotalCostUSD:   12.50,
			ActiveDays:     15,
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := c.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error: %v", err)
	}
	if resp.TotalSessions != 100 {
		t.Errorf("TotalSessions = %d, want 100", resp.TotalSessions)
	}
	if resp.TotalCostUSD != 12.50 {
		t.Errorf("TotalCostUSD = %f, want 12.50", resp.TotalCostUSD)
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "session not found", Code: "not_found"})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Session(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expected := "hawk-sdk: session not found [not_found] (status 404)"
	if got := err.Error(); got != expected {
		t.Errorf("error = %q, want %q", got, expected)
	}

	// Verify typed error via errors.As.
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Error("expected error to be NotFoundError")
	}
}

func TestDeleteSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/sessions/sess-456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	err := c.DeleteSession(context.Background(), "sess-456")
	if err != nil {
		t.Fatalf("DeleteSession() error: %v", err)
	}
}

func TestPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		limit := r.URL.Query().Get("limit")
		if offset != "10" {
			t.Errorf("offset = %q, want %q", offset, "10")
		}
		if limit != "5" {
			t.Errorf("limit = %q, want %q", limit, "5")
		}
		json.NewEncoder(w).Encode(PaginatedResponse[Message]{
			Data:    []Message{{Role: "user", Content: "hi"}},
			Total:   50,
			Offset:  10,
			Limit:   5,
			HasMore: true,
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := c.Messages(context.Background(), "sess-1", &ListOptions{Offset: 10, Limit: 5})
	if err != nil {
		t.Fatalf("Messages() error: %v", err)
	}
	if resp.Total != 50 {
		t.Errorf("Total = %d, want 50", resp.Total)
	}
	if !resp.HasMore {
		t.Error("HasMore = false, want true")
	}
	if len(resp.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].Content != "hi" {
		t.Errorf("Data[0].Content = %q, want %q", resp.Data[0].Content, "hi")
	}
}
