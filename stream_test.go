package hawksdk

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestStreamReader creates a StreamReader from a raw SSE string body.
func newTestStreamReader(body string) *StreamReader {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(body)),
	}
	return newStreamReader(resp)
}

func TestStreamReader_BasicEvent(t *testing.T) {
	sr := newTestStreamReader("data: Hello\n\n")
	defer sr.Close()

	ev, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev.Data != "Hello" {
		t.Errorf("Data = %q, want %q", ev.Data, "Hello")
	}
	if ev.Event != "" {
		t.Errorf("Event = %q, want empty", ev.Event)
	}
}

func TestStreamReader_EventWithType(t *testing.T) {
	sr := newTestStreamReader("event: content\ndata: some text\n\n")
	defer sr.Close()

	ev, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev.Event != "content" {
		t.Errorf("Event = %q, want %q", ev.Event, "content")
	}
	if ev.Data != "some text" {
		t.Errorf("Data = %q, want %q", ev.Data, "some text")
	}
}

func TestStreamReader_MultipleEvents(t *testing.T) {
	sr := newTestStreamReader("data: first\n\ndata: second\n\ndata: third\n\n")
	defer sr.Close()

	for i, want := range []string{"first", "second", "third"} {
		ev, err := sr.Next()
		if err != nil {
			t.Fatalf("Next() #%d error: %v", i, err)
		}
		if ev.Data != want {
			t.Errorf("event %d Data = %q, want %q", i, ev.Data, want)
		}
	}

	_, err := sr.Next()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestStreamReader_MultiLineData(t *testing.T) {
	// SSE spec: multiple "data:" lines are joined, but our parser only takes
	// the last one (no newline join). This tests the actual behavior.
	sr := newTestStreamReader("data: line1\ndata: line2\n\n")
	defer sr.Close()

	ev, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	// The parser overwrites; the last data line wins.
	if ev.Data != "line2" {
		t.Errorf("Data = %q, want %q", ev.Data, "line2")
	}
}

func TestStreamReader_EmptyData(t *testing.T) {
	sr := newTestStreamReader("data:\n\n")
	defer sr.Close()

	ev, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev.Data != "" {
		t.Errorf("Data = %q, want empty", ev.Data)
	}
}

func TestStreamReader_EmptyStream(t *testing.T) {
	sr := newTestStreamReader("")
	defer sr.Close()

	_, err := sr.Next()
	if err != io.EOF {
		t.Errorf("expected io.EOF on empty stream, got %v", err)
	}
}

func TestStreamReader_OnlyComments(t *testing.T) {
	// Lines starting with ":" are SSE comments and should be ignored.
	sr := newTestStreamReader(": this is a comment\n: another comment\n\n")
	defer sr.Close()

	_, err := sr.Next()
	if err != io.EOF {
		t.Errorf("expected io.EOF on comment-only stream, got %v", err)
	}
}

func TestStreamReader_EventDone(t *testing.T) {
	sr := newTestStreamReader("event: done\ndata: {}\n\n")
	defer sr.Close()

	ev, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev.Event != "done" {
		t.Errorf("Event = %q, want %q", ev.Event, "done")
	}
	if ev.Data != "{}" {
		t.Errorf("Data = %q, want %q", ev.Data, "{}")
	}
}

func TestStreamReader_EventError(t *testing.T) {
	sr := newTestStreamReader("event: error\ndata: something went wrong\n\n")
	defer sr.Close()

	ev, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev.Event != "error" {
		t.Errorf("Event = %q, want %q", ev.Event, "error")
	}
	if ev.Data != "something went wrong" {
		t.Errorf("Data = %q, want %q", ev.Data, "something went wrong")
	}
}

func TestStreamReader_EventWithoutData(t *testing.T) {
	// An event line followed by an empty line (no data) should not produce an event.
	sr := newTestStreamReader("event: content\n\ndata: actual\n\n")
	defer sr.Close()

	ev, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	// The event-only block has no data, so it's skipped.
	// The second block has data.
	if ev.Data != "actual" {
		t.Errorf("Data = %q, want %q", ev.Data, "actual")
	}
}

func TestStreamReader_CloseNilBody(t *testing.T) {
	sr := &StreamReader{resp: nil}
	err := sr.Close()
	if err != nil {
		t.Errorf("Close() on nil resp error: %v", err)
	}
}

func TestStreamReader_JSONData(t *testing.T) {
	sr := newTestStreamReader("event: tool_call\ndata: {\"id\":\"tc-1\",\"name\":\"search\",\"arguments\":\"{}\"}\n\n")
	defer sr.Close()

	ev, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev.Event != "tool_call" {
		t.Errorf("Event = %q, want %q", ev.Event, "tool_call")
	}
	if !strings.Contains(ev.Data, `"id":"tc-1"`) {
		t.Errorf("Data missing expected JSON: %q", ev.Data)
	}
}

func TestStreamReader_ToolCallDeltaEvent(t *testing.T) {
	sr := newTestStreamReader("event: tool_call_delta\ndata: {\"id\":\"tc-1\",\"arguments\":\"partial\"}\n\n")
	defer sr.Close()

	ev, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev.Event != "tool_call_delta" {
		t.Errorf("Event = %q, want %q", ev.Event, "tool_call_delta")
	}
}

// --- classifyEvent tests ---

func TestClassifyEvent_ContentDefault(t *testing.T) {
	ev := &StreamEvent{Data: "hello world"}
	typed := classifyEvent(ev)
	if typed.Type != StreamEventContent {
		t.Errorf("Type = %v, want StreamEventContent", typed.Type)
	}
	if typed.Content != "hello world" {
		t.Errorf("Content = %q, want %q", typed.Content, "hello world")
	}
}

func TestClassifyEvent_ContentJSON(t *testing.T) {
	ev := &StreamEvent{Event: "content", Data: `{"content":"parsed text"}`}
	typed := classifyEvent(ev)
	if typed.Type != StreamEventContent {
		t.Errorf("Type = %v, want StreamEventContent", typed.Type)
	}
	if typed.Content != "parsed text" {
		t.Errorf("Content = %q, want %q", typed.Content, "parsed text")
	}
}

func TestClassifyEvent_Completion(t *testing.T) {
	for _, eventName := range []string{"done", "complete"} {
		ev := &StreamEvent{Event: eventName, Data: "{}"}
		typed := classifyEvent(ev)
		if typed.Type != StreamEventCompletion {
			t.Errorf("event=%q: Type = %v, want StreamEventCompletion", eventName, typed.Type)
		}
	}
}

func TestClassifyEvent_Error(t *testing.T) {
	ev := &StreamEvent{Event: "error", Data: "bad things happened"}
	typed := classifyEvent(ev)
	if typed.Type != StreamEventError {
		t.Errorf("Type = %v, want StreamEventError", typed.Type)
	}
	if typed.Error == nil {
		t.Fatal("Error is nil")
	}
	if !strings.Contains(typed.Error.Error(), "bad things happened") {
		t.Errorf("Error = %q, want to contain %q", typed.Error.Error(), "bad things happened")
	}
}

func TestClassifyEvent_ToolCall(t *testing.T) {
	ev := &StreamEvent{Event: "tool_call", Data: `{"id":"tc-1","name":"search","arguments":"{}"}`}
	typed := classifyEvent(ev)
	if typed.Type != StreamEventToolCall {
		t.Errorf("Type = %v, want StreamEventToolCall", typed.Type)
	}
	if typed.ToolCall == nil {
		t.Fatal("ToolCall is nil")
	}
	if typed.ToolCall.ID != "tc-1" {
		t.Errorf("ToolCall.ID = %q, want %q", typed.ToolCall.ID, "tc-1")
	}
	if typed.ToolCall.Name != "search" {
		t.Errorf("ToolCall.Name = %q, want %q", typed.ToolCall.Name, "search")
	}
}

func TestClassifyEvent_ToolCallDelta(t *testing.T) {
	ev := &StreamEvent{Event: "tool_call_delta", Data: `{"id":"tc-2","arguments":"partial"}`}
	typed := classifyEvent(ev)
	if typed.Type != StreamEventToolCall {
		t.Errorf("Type = %v, want StreamEventToolCall", typed.Type)
	}
	if typed.ToolCall == nil {
		t.Fatal("ToolCall is nil")
	}
	if typed.ToolCall.ID != "tc-2" {
		t.Errorf("ToolCall.ID = %q, want %q", typed.ToolCall.ID, "tc-2")
	}
}

func TestClassifyEvent_ToolCallInvalidJSON(t *testing.T) {
	// If tool_call event has invalid JSON, it falls back to content.
	ev := &StreamEvent{Event: "tool_call", Data: "not json"}
	typed := classifyEvent(ev)
	if typed.Type != StreamEventContent {
		t.Errorf("Type = %v, want StreamEventContent (fallback)", typed.Type)
	}
	if typed.Content != "not json" {
		t.Errorf("Content = %q, want %q", typed.Content, "not json")
	}
}

func TestClassifyEvent_DeltaEvent(t *testing.T) {
	// "delta" event with JSON content field.
	ev := &StreamEvent{Event: "delta", Data: `{"content":"chunk"}`}
	typed := classifyEvent(ev)
	if typed.Type != StreamEventContent {
		t.Errorf("Type = %v, want StreamEventContent", typed.Type)
	}
	if typed.Content != "chunk" {
		t.Errorf("Content = %q, want %q", typed.Content, "chunk")
	}
}

// --- collectToolCallValues tests ---

func TestCollectToolCallValues_PreservesOrder(t *testing.T) {
	m := map[string]*ToolCallDelta{
		"b": {ID: "b", Name: "second"},
		"a": {ID: "a", Name: "first"},
	}
	order := []string{"a", "b"}
	result := collectToolCallValues(m, order)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0].ID != "a" || result[1].ID != "b" {
		t.Errorf("order wrong: got [%s, %s], want [a, b]", result[0].ID, result[1].ID)
	}
}

func TestCollectToolCallValues_Empty(t *testing.T) {
	result := collectToolCallValues(map[string]*ToolCallDelta{}, nil)
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

// --- StreamReader via httptest (integration-style) ---

func TestStreamReader_ViaHTTPServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		fmt.Fprint(w, "event: content\ndata: Hello\n\n")
		flusher.Flush()
		fmt.Fprint(w, "event: content\ndata: World\n\n")
		flusher.Flush()
		fmt.Fprint(w, "event: done\ndata: {}\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("http.Get() error: %v", err)
	}
	sr := newStreamReader(resp)
	defer sr.Close()

	ev1, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() #1 error: %v", err)
	}
	if ev1.Event != "content" || ev1.Data != "Hello" {
		t.Errorf("event 1 = {%q, %q}, want {content, Hello}", ev1.Event, ev1.Data)
	}

	ev2, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() #2 error: %v", err)
	}
	if ev2.Data != "World" {
		t.Errorf("event 2 Data = %q, want %q", ev2.Data, "World")
	}

	ev3, err := sr.Next()
	if err != nil {
		t.Fatalf("Next() #3 error: %v", err)
	}
	if ev3.Event != "done" {
		t.Errorf("event 3 Event = %q, want %q", ev3.Event, "done")
	}

	_, err = sr.Next()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestStreamReader_ErrorEventViaHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "event: error\ndata: rate limit exceeded\n\n")
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("http.Get() error: %v", err)
	}
	sr := newStreamReader(resp)
	defer sr.Close()

	ctx := context.Background()
	ch := sr.Events(ctx)

	var events []TypedStreamEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) < 2 {
		t.Fatalf("got %d events, want at least 2", len(events))
	}

	if events[0].Type != StreamEventError {
		t.Errorf("events[0].Type = %v, want StreamEventError", events[0].Type)
	}
	if events[0].Error == nil {
		t.Fatal("events[0].Error is nil")
	}
	var apiErr *APIError
	if !errors.As(events[0].Error, &apiErr) {
		t.Errorf("error is not *APIError: %T", events[0].Error)
	}

	if events[1].Type != StreamEventCompletion {
		t.Errorf("events[1].Type = %v, want StreamEventCompletion", events[1].Type)
	}
}

// --- ToolCallDelta struct test ---

func TestToolCallDelta_Fields(t *testing.T) {
	td := ToolCallDelta{
		ID:        "tc-1",
		Name:      "search",
		Arguments: `{"q":"test"}`,
	}
	if td.ID != "tc-1" {
		t.Errorf("ID = %q, want %q", td.ID, "tc-1")
	}
	if td.Name != "search" {
		t.Errorf("Name = %q, want %q", td.Name, "search")
	}
	if td.Arguments != `{"q":"test"}` {
		t.Errorf("Arguments = %q, want %q", td.Arguments, `{"q":"test"}`)
	}
}

// Ensure bufio.Scanner is used (compile-time check).
var _ *bufio.Scanner
