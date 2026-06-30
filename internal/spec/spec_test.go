package spec

import (
	"encoding/json"
	"testing"
)

func TestSpecTypesCompileAndJSONRoundTrip(t *testing.T) {
	// Verify ChatRequest JSON round-trip matches OpenAPI schema expectations.
	req := ChatRequest{
		Message: "hello",
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal ChatRequest: %v", err)
	}
	var got ChatRequest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal ChatRequest: %v", err)
	}
	if got.Message != "hello" {
		t.Errorf("Message = %q, want %q", got.Message, "hello")
	}
}

func TestSpecEnumValues(t *testing.T) {
	if User != "user" {
		t.Errorf("User = %q, want %q", User, "user")
	}
	if Assistant != "assistant" {
		t.Errorf("Assistant = %q, want %q", Assistant, "assistant")
	}
	if Tool != "tool" {
		t.Errorf("Tool = %q, want %q", Tool, "tool")
	}
}

func TestSpecTypesNotEmpty(t *testing.T) {
	var chatReq ChatRequest
	if chatReq.Message != "" {
		t.Error("zero value ChatRequest.Message should be empty")
	}
	var sess Session
	if sess.Id != nil {
		t.Error("zero value Session.Id should be nil")
	}
	var msg Message
	if msg.Role != nil {
		t.Error("zero value Message.Role should be nil")
	}
}

func TestSpecMessageRoleValid(t *testing.T) {
	if !User.Valid() {
		t.Error("User.Valid() should be true")
	}
	if !Assistant.Valid() {
		t.Error("Assistant.Valid() should be true")
	}
	if !Tool.Valid() {
		t.Error("Tool.Valid() should be true")
	}
	if MessageRole("bogus").Valid() {
		t.Error("bogus role Valid() should be false")
	}
}
