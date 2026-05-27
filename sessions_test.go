package hawksdk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestSessionsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if got := r.URL.Query().Get("offset"); got != "5" {
			t.Errorf("offset = %q, want %q", got, "5")
		}
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Errorf("limit = %q, want %q", got, "2")
		}

		json.NewEncoder(w).Encode(PaginatedResponse[SessionSummary]{
			Data: []SessionSummary{
				{ID: "sess-1", Turns: 5, CWD: "/tmp"},
				{ID: "sess-2", Turns: 10, CWD: "/home"},
			},
			Total:   5,
			Offset:  5,
			Limit:   2,
			HasMore: true,
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := c.Sessions(context.Background(), &ListOptions{Offset: 5, Limit: 2})
	if err != nil {
		t.Fatalf("Sessions() error: %v", err)
	}
	if resp.Total != 5 {
		t.Errorf("Total = %d, want 5", resp.Total)
	}
	if !resp.HasMore {
		t.Error("HasMore = false, want true")
	}
	if len(resp.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(resp.Data))
	}
	if resp.Data[0].ID != "sess-1" {
		t.Errorf("Data[0].ID = %q, want %q", resp.Data[0].ID, "sess-1")
	}
	if resp.Data[1].Turns != 10 {
		t.Errorf("Data[1].Turns = %d, want 10", resp.Data[1].Turns)
	}
}

func TestSessionsListNoPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query params: %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(PaginatedResponse[SessionSummary]{
			Data:  []SessionSummary{},
			Total: 0,
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := c.Sessions(context.Background(), nil)
	if err != nil {
		t.Fatalf("Sessions() error: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("Total = %d, want 0", resp.Total)
	}
	if resp.HasMore {
		t.Error("HasMore = true, want false")
	}
}

func TestSessionGetByID(t *testing.T) {
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/sess-abc" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(SessionDetail{
			ID:           "sess-abc",
			CreatedAt:    now,
			UpdatedAt:    now,
			Model:        "claude-opus-4-6",
			Provider:     "anthropic",
			CWD:          "/home/user",
			Name:         "test-session",
			MessageCount: 15,
			ToolCalls:    7,
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := c.Session(context.Background(), "sess-abc")
	if err != nil {
		t.Fatalf("Session() error: %v", err)
	}
	if resp.ID != "sess-abc" {
		t.Errorf("ID = %q, want %q", resp.ID, "sess-abc")
	}
	if resp.Model != "claude-opus-4-6" {
		t.Errorf("Model = %q, want %q", resp.Model, "claude-opus-4-6")
	}
	if resp.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "anthropic")
	}
	if resp.MessageCount != 15 {
		t.Errorf("MessageCount = %d, want 15", resp.MessageCount)
	}
	if resp.ToolCalls != 7 {
		t.Errorf("ToolCalls = %d, want 7", resp.ToolCalls)
	}
}

func TestSessionGetByIDPathEscape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// r.URL.Path is decoded by the HTTP server, so check RawPath for the encoded form.
		wantRaw := "/v1/sessions/sess%2Fwith%2Fslashes"
		if r.URL.RawPath != "" && r.URL.RawPath != wantRaw {
			t.Errorf("RawPath = %q, want %q", r.URL.RawPath, wantRaw)
		}
		// The decoded path should have the slashes unescaped.
		if r.URL.Path != "/v1/sessions/sess/with/slashes" {
			t.Errorf("Path = %q, want %q", r.URL.Path, "/v1/sessions/sess/with/slashes")
		}
		json.NewEncoder(w).Encode(SessionDetail{ID: "sess/with/slashes"})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := c.Session(context.Background(), "sess/with/slashes")
	if err != nil {
		t.Fatalf("Session() error: %v", err)
	}
	if resp.ID != "sess/with/slashes" {
		t.Errorf("ID = %q, want %q", resp.ID, "sess/with/slashes")
	}
}

func TestMessagesGetWithPagination(t *testing.T) {
	var gotOffset, gotLimit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/sess-42/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		gotOffset = r.URL.Query().Get("offset")
		gotLimit = r.URL.Query().Get("limit")

		json.NewEncoder(w).Encode(PaginatedResponse[Message]{
			Data: []Message{
				{Role: "user", Content: "hello"},
				{Role: "assistant", Content: "hi there"},
			},
			Total:   20,
			Offset:  5,
			Limit:   10,
			HasMore: true,
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := c.Messages(context.Background(), "sess-42", &ListOptions{Offset: 5, Limit: 10})
	if err != nil {
		t.Fatalf("Messages() error: %v", err)
	}
	if gotOffset != "5" {
		t.Errorf("offset param = %q, want %q", gotOffset, "5")
	}
	if gotLimit != "10" {
		t.Errorf("limit param = %q, want %q", gotLimit, "10")
	}
	if resp.Total != 20 {
		t.Errorf("Total = %d, want 20", resp.Total)
	}
	if resp.Offset != 5 {
		t.Errorf("Offset = %d, want 5", resp.Offset)
	}
	if !resp.HasMore {
		t.Error("HasMore = false, want true")
	}
	if len(resp.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(resp.Data))
	}
	if resp.Data[0].Role != "user" {
		t.Errorf("Data[0].Role = %q, want %q", resp.Data[0].Role, "user")
	}
	if resp.Data[1].Content != "hi there" {
		t.Errorf("Data[1].Content = %q, want %q", resp.Data[1].Content, "hi there")
	}
}

func TestMessagesPathEscape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantRaw := "/v1/sessions/id%2F1/messages"
		if r.URL.RawPath != "" && r.URL.RawPath != wantRaw {
			t.Errorf("RawPath = %q, want %q", r.URL.RawPath, wantRaw)
		}
		if r.URL.Path != "/v1/sessions/id/1/messages" {
			t.Errorf("Path = %q, want %q", r.URL.Path, "/v1/sessions/id/1/messages")
		}
		json.NewEncoder(w).Encode(PaginatedResponse[Message]{Data: []Message{}})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Messages(context.Background(), "id/1", nil)
	if err != nil {
		t.Fatalf("Messages() error: %v", err)
	}
}

func TestDeleteSessionSuccess(t *testing.T) {
	var method atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method.Store(r.Method)
		if r.URL.Path != "/v1/sessions/sess-to-delete" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	err := c.DeleteSession(context.Background(), "sess-to-delete")
	if err != nil {
		t.Fatalf("DeleteSession() error: %v", err)
	}
	if m := method.Load().(string); m != "DELETE" {
		t.Errorf("method = %q, want DELETE", m)
	}
}

func TestDeleteSessionOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	err := c.DeleteSession(context.Background(), "sess-ok")
	if err != nil {
		t.Fatalf("DeleteSession() error: %v", err)
	}
}

func TestDeleteSessionPathEscape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantRaw := "/v1/sessions/sess%2F123"
		if r.URL.RawPath != "" && r.URL.RawPath != wantRaw {
			t.Errorf("RawPath = %q, want %q", r.URL.RawPath, wantRaw)
		}
		if r.URL.Path != "/v1/sessions/sess/123" {
			t.Errorf("Path = %q, want %q", r.URL.Path, "/v1/sessions/sess/123")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	err := c.DeleteSession(context.Background(), "sess/123")
	if err != nil {
		t.Fatalf("DeleteSession() error: %v", err)
	}
}

func TestCreateSession(t *testing.T) {
	var gotBody CreateSessionRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		json.NewDecoder(r.Body).Decode(&gotBody)

		json.NewEncoder(w).Encode(SessionSummary{
			ID:        "new-sess-1",
			CreatedAt: time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC),
			Turns:     0,
			CWD:       "/home/user/project",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	req := CreateSessionRequest{
		Model: "claude-opus-4-6",
		CWD:   "/home/user/project",
		Name:  "my-session",
	}
	resp, err := c.CreateSession(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateSession() error: %v", err)
	}
	if gotBody.Model != "claude-opus-4-6" {
		t.Errorf("request Model = %q, want %q", gotBody.Model, "claude-opus-4-6")
	}
	if gotBody.CWD != "/home/user/project" {
		t.Errorf("request CWD = %q, want %q", gotBody.CWD, "/home/user/project")
	}
	if gotBody.Name != "my-session" {
		t.Errorf("request Name = %q, want %q", gotBody.Name, "my-session")
	}
	if resp.ID != "new-sess-1" {
		t.Errorf("ID = %q, want %q", resp.ID, "new-sess-1")
	}
	if resp.CWD != "/home/user/project" {
		t.Errorf("CWD = %q, want %q", resp.CWD, "/home/user/project")
	}
}

func TestCreateSessionEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CreateSessionRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "" || req.CWD != "" || req.Name != "" {
			t.Errorf("expected empty request body, got %+v", req)
		}
		json.NewEncoder(w).Encode(SessionSummary{ID: "auto-sess"})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := c.CreateSession(context.Background(), CreateSessionRequest{})
	if err != nil {
		t.Fatalf("CreateSession() error: %v", err)
	}
	if resp.ID != "auto-sess" {
		t.Errorf("ID = %q, want %q", resp.ID, "auto-sess")
	}
}

func TestSessionError404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "session not found",
			Code:  "not_found",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Session(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
	if got := notFound.Code; got != "not_found" {
		t.Errorf("Code = %q, want %q", got, "not_found")
	}
}

func TestSessionsListError404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "endpoint not found",
			Code:  "not_found",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Sessions(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestSessionError500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "database connection failed",
			Code:  "internal",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Session(context.Background(), "sess-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var internal *InternalServerError
	if !errors.As(err, &internal) {
		t.Errorf("expected InternalServerError, got %T: %v", err, err)
	}
	if got := internal.Message; got != "database connection failed" {
		t.Errorf("Message = %q, want %q", got, "database connection failed")
	}
}

func TestCreateSessionError500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "failed to create session",
			Code:  "internal",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.CreateSession(context.Background(), CreateSessionRequest{Model: "test"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var internal *InternalServerError
	if !errors.As(err, &internal) {
		t.Errorf("expected InternalServerError, got %T: %v", err, err)
	}
}

func TestDeleteSessionError404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "session not found",
			Code:  "not_found",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	err := c.DeleteSession(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestDeleteSessionError500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "cleanup failed",
			Code:  "internal",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	err := c.DeleteSession(context.Background(), "sess-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var internal *InternalServerError
	if !errors.As(err, &internal) {
		t.Errorf("expected InternalServerError, got %T: %v", err, err)
	}
}

func TestMessagesError404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "session not found",
			Code:  "not_found",
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Messages(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestSessionError500PlainBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unexpected failure"))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Session(context.Background(), "sess-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var internal *InternalServerError
	if !errors.As(err, &internal) {
		t.Errorf("expected InternalServerError, got %T: %v", err, err)
	}
	if got := internal.Message; got != "unexpected failure" {
		t.Errorf("Message = %q, want %q", got, "unexpected failure")
	}
}
