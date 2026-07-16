package hawksdk

import "testing"

func TestParseInto(t *testing.T) {
	type result struct {
		Answer int    `json:"answer"`
		Note   string `json:"note"`
	}

	resp := &ChatResponse{Response: `{"answer": 42, "note": "ok"}`}
	got, err := ParseInto[result](resp)
	if err != nil {
		t.Fatalf("ParseInto() error: %v", err)
	}
	if got.Answer != 42 || got.Note != "ok" {
		t.Errorf("ParseInto() = %+v, want {42 ok}", got)
	}
}

func TestParseIntoMalformedJSON(t *testing.T) {
	type result struct {
		Answer int `json:"answer"`
	}

	resp := &ChatResponse{Response: "not json at all"}
	if _, err := ParseInto[result](resp); err == nil {
		t.Fatal("expected an error for malformed JSON, got nil")
	}
}

func TestParseIntoNilResponse(t *testing.T) {
	type result struct{}

	if _, err := ParseInto[result](nil); err == nil {
		t.Fatal("expected an error for nil response, got nil")
	}
}
