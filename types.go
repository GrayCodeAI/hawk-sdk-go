package hawksdk

import "time"

// ChatRequest is the request body for POST /v1/chat.
type ChatRequest struct {
	Prompt    string `json:"prompt"`
	SessionID string `json:"session_id,omitempty"`
	Model     string `json:"model,omitempty"`
	MaxTurns  int    `json:"max_turns,omitempty"`
	Autonomy  string `json:"autonomy,omitempty"`
	CWD       string `json:"cwd,omitempty"`
	Agent     string `json:"agent,omitempty"`
}

// ChatResponse is the response from POST /v1/chat.
type ChatResponse struct {
	SessionID  string `json:"session_id"`
	Response   string `json:"response"`
	TokensIn   int    `json:"tokens_in"`
	TokensOut  int    `json:"tokens_out"`
	TurnsTaken int    `json:"turns_taken"`
	Duration   string `json:"duration"`
}

// HealthResponse is the response from GET /v1/health.
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	Sessions  int    `json:"active_sessions"`
	StartedAt string `json:"started_at"`
}

// SessionSummary is a session entry in the list response.
type SessionSummary struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
	Turns     int       `json:"turns"`
	CWD       string    `json:"cwd"`
}

// SessionDetail is the full session detail from GET /v1/sessions/{id}.
type SessionDetail struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Model        string    `json:"model"`
	Provider     string    `json:"provider"`
	CWD          string    `json:"cwd"`
	Name         string    `json:"name"`
	MessageCount int       `json:"message_count"`
	ToolCalls    int       `json:"tool_calls"`
}

// Message is a conversation message.
type Message struct {
	Role       string      `json:"role"`
	Content    string      `json:"content,omitempty"`
	ToolUse    interface{} `json:"tool_use,omitempty"`
	ToolResult interface{} `json:"tool_result,omitempty"`
}

// StatsResponse is the response from GET /v1/stats.
type StatsResponse struct {
	TotalSessions  int         `json:"total_sessions"`
	TotalMessages  int         `json:"total_messages"`
	TotalToolCalls int         `json:"total_tool_calls"`
	TotalCostUSD   float64     `json:"total_cost_usd"`
	ActiveDays     int         `json:"active_days"`
	Models         []ModelStat `json:"models"`
}

// ModelStat is per-model usage in StatsResponse.
type ModelStat struct {
	Model    string  `json:"model"`
	Requests int     `json:"requests"`
	CostUSD  float64 `json:"cost_usd"`
}

// PaginatedResponse wraps paginated list results.
type PaginatedResponse[T any] struct {
	Data    []T  `json:"data"`
	Total   int  `json:"total"`
	Offset  int  `json:"offset"`
	Limit   int  `json:"limit"`
	HasMore bool `json:"has_more"`
}

// ListOptions configures pagination for list endpoints.
type ListOptions struct {
	Offset int
	Limit  int
}

// ErrorResponse is the standard error envelope from the daemon.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
