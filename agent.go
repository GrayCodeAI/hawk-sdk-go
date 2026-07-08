package hawksdk

import (
	"context"
	"sync"
)

// AgentConfig holds the declarative configuration for an Agent.
type AgentConfig struct {
	// Model specifies which LLM model to use.
	Model string

	// Tools are the tools available to this agent.
	Tools []Tool

	// MaxRounds limits the tool execution loop iterations.
	MaxRounds int

	// Memory is an optional configuration for agent memory/context management.
	Memory *MemoryConfig

	// NOTE: Name, SystemPrompt, Temperature, and TopP are not yet supported
	// by the daemon's ChatRequest API. They will be added when the server
	// exposes corresponding fields.
}

// MemoryConfig configures memory behavior for an agent.
type MemoryConfig struct {
	// Enabled controls whether memory is active. If false, SessionID is
	// ignored and each Chat call starts a fresh, session-less conversation.
	Enabled bool

	// SessionID allows resuming a previous session. Only used when Enabled
	// is true.
	SessionID string

	// MaxMessages is reserved for a future client-side history limit; it is
	// not yet enforced (the daemon manages context retention per session).
	MaxMessages int
}

// Agent wraps a Client with declarative configuration, providing a
// simplified interface for conversational AI interactions.
//
// Concurrency: Agent is safe for concurrent use. The session ID is read and
// updated under an internal mutex. Each Chat or ChatStream call captures the
// session ID at the moment the request is built; a stream returned by
// ChatStream continues to use the session ID captured at call time even if a
// concurrent Chat call establishes a new session while the stream is being
// consumed.
type Agent struct {
	client *Client
	config AgentConfig

	// mu protects sessionID from concurrent reads/writes.
	mu sync.Mutex

	// sessionID tracks the current session for continuity.
	sessionID string
}

// NewAgent creates a new Agent with the given client and configuration.
func NewAgent(client *Client, config AgentConfig) *Agent {
	a := &Agent{
		client: client,
		config: config,
	}
	if config.Memory != nil && config.Memory.Enabled && config.Memory.SessionID != "" {
		a.sessionID = config.Memory.SessionID
	}
	return a
}

// Chat sends a message and returns the complete response.
// If the agent has tools configured, it automatically uses ChatWithTools.
func (a *Agent) Chat(ctx context.Context, message string) (*ChatResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	req := a.buildRequest(message)

	// If tools are configured, use the tool execution loop.
	if len(a.config.Tools) > 0 {
		maxRounds := a.config.MaxRounds
		if maxRounds <= 0 {
			maxRounds = 10
		}
		resp, err := a.client.ChatWithTools(ctx, req, a.config.Tools, maxRounds)
		if err != nil {
			return nil, err
		}
		a.updateSession(resp)
		return resp, nil
	}

	resp, err := a.client.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	a.updateSession(resp)
	return resp, nil
}

// ChatStream sends a message and returns a streaming response reader.
// Note: streaming with tools is not automatically looped; use Chat for
// full tool loop support.
//
// The session ID is captured under the agent's lock when the request is
// built, so the entire stream lifecycle uses that snapshot: a concurrent
// Chat call that mutates the agent's session ID does not affect an
// in-flight stream.
func (a *Agent) ChatStream(ctx context.Context, message string) (*StreamReader, error) {
	// Capture the session ID into the request under the lock. req holds the
	// captured value by copy, so the stream is immune to later mutations of
	// a.sessionID by concurrent Chat calls.
	a.mu.Lock()
	req := a.buildRequest(message)
	a.mu.Unlock()
	return a.client.ChatStream(ctx, req)
}

// SessionID returns the current session ID, if established.
func (a *Agent) SessionID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sessionID
}

// buildRequest constructs a ChatRequest from the agent config and message.
func (a *Agent) buildRequest(message string) ChatRequest {
	req := ChatRequest{
		Prompt:    message,
		Model:     a.config.Model,
		SessionID: a.sessionID,
	}

	if a.config.MaxRounds > 0 {
		req.MaxTurns = a.config.MaxRounds
	}

	return req
}

// updateSession stores the session ID from the response for continuity.
func (a *Agent) updateSession(resp *ChatResponse) {
	if resp != nil && resp.SessionID != "" {
		a.sessionID = resp.SessionID
	}
}
