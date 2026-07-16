package hawksdk

import (
	"encoding/json"
	"fmt"
)

// ParseInto decodes a ChatResponse's Response text as JSON into a caller-
// supplied type T. This is a best-effort client-side convenience: the daemon
// API has no schema-negotiation mechanism (ChatRequest carries no response-
// format field), so the model isn't constrained server-side to emit T-shaped
// JSON. Callers that need that guarantee should instruct the model to emit
// JSON via the prompt and treat ParseInto's error as "the model didn't
// comply," not as a validation failure of the daemon.
func ParseInto[T any](resp *ChatResponse) (T, error) {
	var out T
	if resp == nil {
		return out, fmt.Errorf("hawk-sdk: ParseInto: nil response")
	}
	if err := json.Unmarshal([]byte(resp.Response), &out); err != nil {
		return out, fmt.Errorf("hawk-sdk: ParseInto: response is not valid JSON for the target type: %w", err)
	}
	return out, nil
}
