package hawksdk

import "context"

// AgentConfig holds the declarative configuration for an Agent.
type AgentConfig struct {
	// Name identifies this agent.
	Name string

	// Model specifies which LLM model to use.
	Model string

	// SystemPrompt is the system instruction prepended to conversations.
	SystemPrompt string

	// Tools are the tools available to this agent.
	Tools []Tool

	// MaxRounds limits the tool execution loop iterations.
	MaxRounds int

	// Temperature controls randomness in the model's output (0.0-2.0).
	Temperature *float64

	// TopP controls nucleus sampling (0.0-1.0).
	TopP *float64

	// Memory is an optional configuration for agent memory/context management.
	Memory *MemoryConfig
}

// MemoryConfig configures memory behavior for an agent.
type MemoryConfig struct {
	// Enabled controls whether memory is active.
	Enabled bool

	// SessionID allows resuming a previous session.
	SessionID string

	// MaxMessages limits how many messages are retained in context.
	MaxMessages int
}

// Agent wraps a Client with declarative configuration, providing a
// simplified interface for conversational AI interactions.
type Agent struct {
	client *Client
	config AgentConfig

	// sessionID tracks the current session for continuity.
	sessionID string
}

// NewAgent creates a new Agent with the given client and configuration.
func NewAgent(client *Client, config AgentConfig) *Agent {
	a := &Agent{
		client: client,
		config: config,
	}
	if config.Memory != nil && config.Memory.SessionID != "" {
		a.sessionID = config.Memory.SessionID
	}
	return a
}

// Chat sends a message and returns the complete response.
// If the agent has tools configured, it automatically uses ChatWithTools.
func (a *Agent) Chat(ctx context.Context, message string) (*ChatResponse, error) {
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
func (a *Agent) ChatStream(ctx context.Context, message string) (*StreamReader, error) {
	req := a.buildRequest(message)
	return a.client.ChatStream(ctx, req)
}

// SessionID returns the current session ID, if established.
func (a *Agent) SessionID() string {
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
