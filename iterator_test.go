package hawksdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestAllMessages(t *testing.T) {
	// Three pages of 2 messages each, offset advancing by limit=2.
	total := 6
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit != 2 {
			t.Errorf("limit = %d, want 2", limit)
		}
		var data []Message
		for i := offset; i < offset+limit && i < total; i++ {
			data = append(data, Message{Role: "user", Content: fmt.Sprintf("msg-%d", i)})
		}
		json.NewEncoder(w).Encode(PaginatedResponse[Message]{
			Data:    data,
			Total:   total,
			Offset:  offset,
			Limit:   limit,
			HasMore: offset+limit < total,
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))

	var got []string
	for msg, err := range AllMessages(context.Background(), c, "sess-1", &ListOptions{Limit: 2}) {
		if err != nil {
			t.Fatalf("AllMessages() error: %v", err)
		}
		got = append(got, msg.Content)
	}

	if len(got) != total {
		t.Fatalf("got %d messages, want %d: %v", len(got), total, got)
	}
	for i, content := range got {
		want := fmt.Sprintf("msg-%d", i)
		if content != want {
			t.Errorf("got[%d] = %q, want %q", i, content, want)
		}
	}
}

func TestAllMessagesStopsEarly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PaginatedResponse[Message]{
			Data:    []Message{{Content: "a"}, {Content: "b"}},
			HasMore: true,
		})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))

	count := 0
	for range AllMessages(context.Background(), c, "sess-1", &ListOptions{Limit: 2}) {
		count++
		if count == 1 {
			break
		}
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (should stop on caller break)", count)
	}
}

func TestAllMessagesPropagatesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "boom"})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))

	var gotErr error
	for _, err := range AllMessages(context.Background(), c, "sess-1", nil) {
		gotErr = err
		break
	}
	if gotErr == nil {
		t.Fatal("expected an error, got nil")
	}
}
