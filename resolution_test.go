package hawksdk_test

import (
	"encoding/json"
	"testing"

	hawksdk "github.com/GrayCodeAI/hawk-sdk-go"
)

func TestResolutionPhaseConstants(t *testing.T) {
	phases := []hawksdk.ResolutionPhase{
		hawksdk.ResolutionPhaseLocalize,
		hawksdk.ResolutionPhaseRepair,
		hawksdk.ResolutionPhaseValidate,
		hawksdk.ResolutionPhaseReview,
		hawksdk.ResolutionPhasePlanning,
	}
	for _, p := range phases {
		if string(p) == "" {
			t.Errorf("ResolutionPhase constant must not be empty")
		}
	}
	if string(hawksdk.ResolutionPhaseUnknown) != "" {
		t.Error("ResolutionPhaseUnknown should be empty")
	}
}

func TestResolutionRequestJSONRoundTrip(t *testing.T) {
	req := hawksdk.ResolutionRequest{
		SessionID:  "sess-123",
		RootDir:    "/repo",
		Query:      "fix the authentication bug",
		MaxFiles:   10,
		MaxSymbols: 5,
		Language:   "go",
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got hawksdk.ResolutionRequest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Query != req.Query {
		t.Errorf("Query = %q, want %q", got.Query, req.Query)
	}
	if got.SessionID != req.SessionID {
		t.Errorf("SessionID = %q, want %q", got.SessionID, req.SessionID)
	}
}

func TestResolutionResultJSONRoundTrip(t *testing.T) {
	result := hawksdk.ResolutionResult{
		SessionID: "sess-123",
		Candidates: []hawksdk.PatchCandidate{
			{
				FilePath:    "auth.go",
				Symbol:      "ValidateToken",
				PatchedBody: "func ValidateToken(t string) error { return nil }",
				Confidence:  0.9,
			},
		},
		ValidationPassed: true,
		PhaseMetrics: []hawksdk.PhaseMetrics{
			{Phase: hawksdk.ResolutionPhaseLocalize, InputTokens: 1000, OutputTokens: 200, ElapsedMs: 50},
			{Phase: hawksdk.ResolutionPhaseRepair, InputTokens: 3000, OutputTokens: 500, ElapsedMs: 200},
			{Phase: hawksdk.ResolutionPhaseValidate, InputTokens: 500, OutputTokens: 100, ElapsedMs: 20},
		},
		TotalInputTokens:  4500,
		TotalOutputTokens: 800,
	}

	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got hawksdk.ResolutionResult
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !got.ValidationPassed {
		t.Error("ValidationPassed should be true after round-trip")
	}
	if len(got.Candidates) != 1 {
		t.Errorf("Candidates len = %d, want 1", len(got.Candidates))
	}
	if len(got.PhaseMetrics) != 3 {
		t.Errorf("PhaseMetrics len = %d, want 3", len(got.PhaseMetrics))
	}
	if got.PhaseMetrics[0].Phase != hawksdk.ResolutionPhaseLocalize {
		t.Errorf("first phase = %q, want %q", got.PhaseMetrics[0].Phase, hawksdk.ResolutionPhaseLocalize)
	}
}
