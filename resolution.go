package hawksdk

// ResolutionPhase identifies which phase of the Agentless pipeline produced
// a given result. Matches the Phase type in hawk/internal/pipeline and
// hawk-core-contracts/sessions.
type ResolutionPhase string

const (
	ResolutionPhaseLocalize ResolutionPhase = "localize"
	ResolutionPhaseRepair   ResolutionPhase = "repair"
	ResolutionPhaseValidate ResolutionPhase = "validate"
	ResolutionPhaseReview   ResolutionPhase = "review"
	ResolutionPhasePlanning ResolutionPhase = "planning"
	ResolutionPhaseUnknown  ResolutionPhase = ""
)

// ResolutionRequest is the SDK-level request to run a multi-phase
// localize → repair → validate (+ review, planning) resolution against a repository.
type ResolutionRequest struct {
	// SessionID links this request to an ongoing session for cost attribution.
	SessionID string `json:"session_id,omitempty"`
	// RootDir is the absolute path to the repository root on the server.
	RootDir string `json:"root_dir"`
	// Query is the natural-language problem description (bug report, task).
	Query string `json:"query"`
	// MaxFiles limits how many files the localize phase considers.
	MaxFiles int `json:"max_files,omitempty"`
	// MaxSymbols limits how many symbols per file the localize phase returns.
	MaxSymbols int `json:"max_symbols,omitempty"`
	// Language restricts localization to one language ("go", "python", etc.).
	Language string `json:"language,omitempty"`
}

// PatchCandidate is a single proposed code change returned by the repair phase.
type PatchCandidate struct {
	// FilePath is the repository-relative path to the file to change.
	FilePath string `json:"file_path"`
	// Symbol is the function or type containing the change.
	Symbol string `json:"symbol,omitempty"`
	// OriginalBody is the existing source of the symbol.
	OriginalBody string `json:"original_body,omitempty"`
	// PatchedBody is the proposed replacement source.
	PatchedBody string `json:"patched_body,omitempty"`
	// Confidence is in [0, 1]; higher means the candidate is more likely correct.
	Confidence float64 `json:"confidence"`
}

// PhaseMetrics records token spend and timing for a single pipeline phase.
type PhaseMetrics struct {
	Phase        ResolutionPhase `json:"phase"`
	InputTokens  int             `json:"input_tokens"`
	OutputTokens int             `json:"output_tokens"`
	ElapsedMs    int64           `json:"elapsed_ms"`
}

// ResolutionResult is the complete output of a three-phase resolution run.
type ResolutionResult struct {
	// SessionID echoes the request SessionID for correlation.
	SessionID string `json:"session_id,omitempty"`
	// Candidates are the patch candidates produced by the repair phase.
	Candidates []PatchCandidate `json:"candidates"`
	// ValidationPassed is true when the applied patch passes all checks.
	ValidationPassed bool `json:"validation_passed"`
	// ValidationFailures lists test or lint failures; empty when ValidationPassed is true.
	ValidationFailures []string `json:"validation_failures,omitempty"`
	// PhaseMetrics records per-phase token spend.
	PhaseMetrics []PhaseMetrics `json:"phase_metrics"`
	// TotalInputTokens is the sum of InputTokens across all phases.
	TotalInputTokens int `json:"total_input_tokens"`
	// TotalOutputTokens is the sum of OutputTokens across all phases.
	TotalOutputTokens int `json:"total_output_tokens"`
}
