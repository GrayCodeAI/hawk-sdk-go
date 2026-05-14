package hawksdk

import (
	"context"
	"encoding/json"
	"io"
	"strings"
)

// StreamEventType identifies the kind of stream event.
type StreamEventType int

const (
	// StreamEventContent indicates a text content chunk.
	StreamEventContent StreamEventType = iota
	// StreamEventToolCall indicates a tool call delta or completion.
	StreamEventToolCall
	// StreamEventCompletion indicates the stream is complete.
	StreamEventCompletion
	// StreamEventError indicates an error event from the stream.
	StreamEventError
)

// TypedStreamEvent is a structured event from a chat stream, providing
// typed access to content, tool calls, or errors.
type TypedStreamEvent struct {
	// Type indicates what kind of event this is.
	Type StreamEventType

	// Content holds the text content for StreamEventContent events.
	Content string

	// ToolCall holds the tool call data for StreamEventToolCall events.
	ToolCall *ToolCallDelta

	// Error holds the error for StreamEventError events.
	Error error
}

// ToolCallDelta represents a fragment of a tool call received from the stream.
type ToolCallDelta struct {
	// ID is the unique identifier for the tool call.
	ID string `json:"id"`

	// Name is the function/tool name.
	Name string `json:"name"`

	// Arguments holds the accumulated JSON arguments (may be partial during streaming).
	Arguments string `json:"arguments"`
}

// CollectText consumes the entire stream and returns the concatenated text content.
// It blocks until the stream ends or the context is cancelled.
func (sr *StreamReader) CollectText(ctx context.Context) (string, error) {
	var sb strings.Builder

	for {
		select {
		case <-ctx.Done():
			return sb.String(), ctx.Err()
		default:
		}

		ev, err := sr.Next()
		if err == io.EOF {
			return sb.String(), nil
		}
		if err != nil {
			return sb.String(), err
		}

		// Skip non-data events and completion events.
		if ev.Event == "done" || ev.Event == "error" {
			continue
		}

		// Try to parse as JSON with a content field; fall back to raw data.
		var parsed struct {
			Content string `json:"content"`
			Type    string `json:"type"`
		}
		if json.Unmarshal([]byte(ev.Data), &parsed) == nil && parsed.Content != "" {
			sb.WriteString(parsed.Content)
		} else if ev.Event == "" || ev.Event == "content" || ev.Event == "delta" {
			sb.WriteString(ev.Data)
		}
	}
}

// CollectToolCalls consumes the stream and assembles complete tool calls
// from fragmented deltas. It returns all tool calls found in the stream.
func (sr *StreamReader) CollectToolCalls(ctx context.Context) ([]ToolCallDelta, error) {
	// Map tool call ID to accumulated delta.
	calls := make(map[string]*ToolCallDelta)
	var order []string

	for {
		select {
		case <-ctx.Done():
			return collectToolCallValues(calls, order), ctx.Err()
		default:
		}

		ev, err := sr.Next()
		if err == io.EOF {
			return collectToolCallValues(calls, order), nil
		}
		if err != nil {
			return collectToolCallValues(calls, order), err
		}

		if ev.Event != "tool_call" && ev.Event != "tool_call_delta" {
			continue
		}

		var delta ToolCallDelta
		if json.Unmarshal([]byte(ev.Data), &delta) != nil {
			continue
		}

		if existing, ok := calls[delta.ID]; ok {
			// Append arguments fragment.
			existing.Arguments += delta.Arguments
			if delta.Name != "" {
				existing.Name = delta.Name
			}
		} else {
			tc := delta
			calls[delta.ID] = &tc
			order = append(order, delta.ID)
		}
	}
}

// Events returns a channel that emits typed stream events.
// The channel is closed when the stream ends. The caller should read from
// the channel until it is closed. Context cancellation stops emission.
func (sr *StreamReader) Events(ctx context.Context) <-chan TypedStreamEvent {
	ch := make(chan TypedStreamEvent, 16)

	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				ch <- TypedStreamEvent{Type: StreamEventError, Error: ctx.Err()}
				return
			default:
			}

			ev, err := sr.Next()
			if err == io.EOF {
				ch <- TypedStreamEvent{Type: StreamEventCompletion}
				return
			}
			if err != nil {
				ch <- TypedStreamEvent{Type: StreamEventError, Error: err}
				return
			}

			typed := classifyEvent(ev)
			ch <- typed
		}
	}()

	return ch
}

// classifyEvent converts a raw SSE event into a TypedStreamEvent.
func classifyEvent(ev *StreamEvent) TypedStreamEvent {
	switch ev.Event {
	case "done", "complete":
		return TypedStreamEvent{Type: StreamEventCompletion}
	case "error":
		return TypedStreamEvent{Type: StreamEventError, Error: &APIError{Message: ev.Data}}
	case "tool_call", "tool_call_delta":
		var delta ToolCallDelta
		if json.Unmarshal([]byte(ev.Data), &delta) == nil {
			return TypedStreamEvent{Type: StreamEventToolCall, ToolCall: &delta}
		}
		return TypedStreamEvent{Type: StreamEventContent, Content: ev.Data}
	default:
		// Default: content event.
		var parsed struct {
			Content string `json:"content"`
		}
		if json.Unmarshal([]byte(ev.Data), &parsed) == nil && parsed.Content != "" {
			return TypedStreamEvent{Type: StreamEventContent, Content: parsed.Content}
		}
		return TypedStreamEvent{Type: StreamEventContent, Content: ev.Data}
	}
}

func collectToolCallValues(m map[string]*ToolCallDelta, order []string) []ToolCallDelta {
	result := make([]ToolCallDelta, 0, len(order))
	for _, id := range order {
		if tc, ok := m[id]; ok {
			result = append(result, *tc)
		}
	}
	return result
}
