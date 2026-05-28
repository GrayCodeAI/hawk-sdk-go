package hawksdk

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// StreamReader reads SSE events from a streaming chat response.
type StreamReader struct {
	scanner *bufio.Scanner
	resp    *http.Response
}

// StreamEvent is a single SSE event from the chat stream.
type StreamEvent struct {
	Event string
	Data  string
}

func newStreamReader(resp *http.Response) *StreamReader {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB max line
	return &StreamReader{
		scanner: scanner,
		resp:    resp,
	}
}

// Next reads the next SSE event from the stream.
// Returns io.EOF when the stream is complete.
func (sr *StreamReader) Next() (*StreamEvent, error) {
	var event StreamEvent
	var hasData bool

	for sr.scanner.Scan() {
		line := sr.scanner.Text()

		if line == "" {
			if hasData {
				return &event, nil
			}
			continue
		}

		switch {
		case strings.HasPrefix(line, "event: "):
			event.Event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			if event.Data != "" {
				event.Data += "\n"
			}
			event.Data += strings.TrimPrefix(line, "data: ")
			hasData = true
		case line == "data:":
			if event.Data != "" {
				event.Data += "\n"
			}
			hasData = true
		}
	}

	if err := sr.scanner.Err(); err != nil {
		return nil, fmt.Errorf("hawk-sdk: stream read error: %w", err)
	}

	if hasData {
		return &event, nil
	}

	return nil, io.EOF
}

// Close closes the underlying HTTP response body.
func (sr *StreamReader) Close() error {
	if sr.resp != nil && sr.resp.Body != nil {
		return sr.resp.Body.Close()
	}
	return nil
}
