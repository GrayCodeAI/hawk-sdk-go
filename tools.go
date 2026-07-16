package hawksdk

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolSchema describes a tool's function signature in the OpenAI function calling format.
type ToolSchema struct {
	// Name is the function name.
	Name string `json:"name"`

	// Description explains what the tool does.
	Description string `json:"description"`

	// Parameters is a JSON Schema object describing the function's parameters.
	Parameters json.RawMessage `json:"parameters"`
}

// Tool represents a callable tool with its schema and execution function.
type Tool struct {
	// Schema describes the tool for the model.
	Schema ToolSchema

	// Run executes the tool with the given arguments and returns a result string.
	Run func(ctx context.Context, arguments map[string]any) (string, error)
}

// ToolCall represents a tool invocation requested by the model.
type ToolCall struct {
	// ID is the unique identifier for this tool call.
	ID string `json:"id"`

	// Name is the function name to invoke.
	Name string `json:"name"`

	// Arguments is the arguments object for the tool call.
	Arguments map[string]any `json:"arguments"`
}

// ToolResult holds the result of executing a tool call.
type ToolResult struct {
	// ToolUseID is the ID of the tool use this result corresponds to.
	// Hawk's daemon keys tool results by the `tool_use_id` field it emitted
	// in the assistant's tool_use block (Anthropic/MCP convention), not OpenAI's
	// `tool_call_id` — the daemon would otherwise drop unmatched results.
	ToolUseID string `json:"tool_use_id"`

	// Content is the string result from the tool execution.
	Content string `json:"content"`

	// IsError indicates whether the tool execution failed.
	IsError bool `json:"is_error,omitempty"`
}

// ChatWithToolsRequest extends ChatRequest with tool definitions for the model.
type ChatWithToolsRequest struct {
	ChatRequest

	// Tools provides the tool schemas to the model.
	Tools []ToolSchema `json:"tools,omitempty"`

	// ToolResults provides results from previous tool calls.
	ToolResults []ToolResult `json:"tool_results,omitempty"`

	// Messages holds the conversation history for multi-turn tool use.
	Messages []Message `json:"messages,omitempty"`
}

// ChatWithToolsResponse extends ChatResponse with tool call information.
type ChatWithToolsResponse struct {
	ChatResponse

	// ToolCalls contains any tool calls the model wants to make.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// FinishReason indicates why the model stopped generating.
	FinishReason string `json:"finish_reason,omitempty"`
}

// ChatWithTools implements the tool execution loop. It sends a chat request,
// checks for tool_calls in the response, executes matching tools, appends
// results to the conversation, and repeats until either no more tool calls
// are requested or maxRounds is reached.
func (c *Client) ChatWithTools(ctx context.Context, req ChatRequest, tools []Tool, maxRounds int) (*ChatResponse, error) {
	if maxRounds <= 0 {
		maxRounds = 10
	}

	// Build tool schemas.
	schemas := make([]ToolSchema, len(tools))
	for i, t := range tools {
		schemas[i] = t.Schema
	}

	// Build tool lookup map.
	toolMap := make(map[string]Tool, len(tools))
	for _, t := range tools {
		toolMap[t.Schema.Name] = t
	}

	// Conversation state.
	var messages []Message
	var toolResults []ToolResult

	for round := 0; round < maxRounds; round++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Build the request for this round.
		toolReq := ChatWithToolsRequest{
			ChatRequest: req,
			Tools:       schemas,
			ToolResults: toolResults,
			Messages:    messages,
		}

		var resp ChatWithToolsResponse
		if err := c.post(ctx, "/v1/chat", toolReq, &resp); err != nil {
			return nil, fmt.Errorf("hawk-sdk: chat with tools round %d: %w", round+1, err)
		}

		// No tool calls — we're done.
		if len(resp.ToolCalls) == 0 || resp.FinishReason == "stop" {
			return &resp.ChatResponse, nil
		}

		// Execute each tool call.
		toolResults = make([]ToolResult, 0, len(resp.ToolCalls))
		for _, tc := range resp.ToolCalls {
			tool, ok := toolMap[tc.Name]
			if !ok {
				toolResults = append(toolResults, ToolResult{
					ToolUseID: tc.ID,
					Content:   fmt.Sprintf("error: unknown tool %q", tc.Name),
					IsError:   true,
				})
				continue
			}

			result, err := tool.Run(ctx, tc.Arguments)
			if err != nil {
				toolResults = append(toolResults, ToolResult{
					ToolUseID: tc.ID,
					Content:   fmt.Sprintf("error: %s", err.Error()),
					IsError:   true,
				})
				continue
			}

			toolResults = append(toolResults, ToolResult{
				ToolUseID: tc.ID,
				Content:   result,
			})
		}

		// Append assistant response and tool results to message history.
		messages = append(messages, Message{
			Role:    "assistant",
			Content: resp.Response,
			ToolUse: resp.ToolCalls,
		})
		messages = append(messages, Message{
			Role:       "tool",
			ToolResult: toolResults,
		})

		// Clear the prompt for subsequent rounds — the messages carry context.
		req.Prompt = ""
	}

	return nil, fmt.Errorf("hawk-sdk: tool execution loop exceeded max rounds (%d)", maxRounds)
}
