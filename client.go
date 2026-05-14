package hawksdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const defaultBaseURL = "http://127.0.0.1:4590"

// Client is a Go SDK client for the hawk daemon API.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	retryConfig *RetryConfig
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithBaseURL sets the daemon base URL (default: http://127.0.0.1:4590).
func WithBaseURL(u string) ClientOption {
	return func(c *Client) { c.baseURL = u }
}

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = hc }
}

// New creates a new hawk SDK client.
func New(opts ...ClientOption) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Health checks daemon connectivity.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var resp HealthResponse
	if err := c.get(ctx, "/v1/health", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Chat sends a prompt and returns the complete response.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	var resp ChatResponse
	if err := c.post(ctx, "/v1/chat", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ChatStream sends a prompt and streams the response via SSE.
func (c *Client) ChatStream(ctx context.Context, req ChatRequest) (*StreamReader, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("hawk-sdk: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("hawk-sdk: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("hawk-sdk: stream request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		return nil, parseAPIError(resp)
	}

	return newStreamReader(resp), nil
}

// Sessions lists all sessions with optional pagination.
func (c *Client) Sessions(ctx context.Context, opts *ListOptions) (*PaginatedResponse[SessionSummary], error) {
	params := paginationParams(opts)
	var resp PaginatedResponse[SessionSummary]
	if err := c.get(ctx, "/v1/sessions", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Session gets a session by ID.
func (c *Client) Session(ctx context.Context, id string) (*SessionDetail, error) {
	var resp SessionDetail
	if err := c.get(ctx, "/v1/sessions/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Messages gets messages for a session with optional pagination.
func (c *Client) Messages(ctx context.Context, sessionID string, opts *ListOptions) (*PaginatedResponse[Message], error) {
	params := paginationParams(opts)
	var resp PaginatedResponse[Message]
	if err := c.get(ctx, "/v1/sessions/"+url.PathEscape(sessionID)+"/messages", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteSession deletes a session by ID.
func (c *Client) DeleteSession(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+"/v1/sessions/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("hawk-sdk: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("hawk-sdk: delete request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return parseAPIError(resp)
	}
	return nil
}

// Stats gets aggregated usage statistics.
func (c *Client) Stats(ctx context.Context) (*StatsResponse, error) {
	var resp StatsResponse
	if err := c.get(ctx, "/v1/stats", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out interface{}) error {
	u := c.baseURL + path
	if params != nil {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return fmt.Errorf("hawk-sdk: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.doWithRetry(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("hawk-sdk: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return parseAPIError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("hawk-sdk: decode response: %w", err)
	}
	return nil
}

func (c *Client) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("hawk-sdk: marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("hawk-sdk: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.doWithRetry(ctx, req, data)
	if err != nil {
		return fmt.Errorf("hawk-sdk: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return parseAPIError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("hawk-sdk: decode response: %w", err)
	}
	return nil
}

func paginationParams(opts *ListOptions) url.Values {
	if opts == nil {
		return nil
	}
	params := url.Values{}
	if opts.Offset > 0 {
		params.Set("offset", strconv.Itoa(opts.Offset))
	}
	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	}
	return params
}

