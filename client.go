package hawksdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "http://127.0.0.1:4590"

// Client is a Go SDK client for the hawk daemon API.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	retryConfig *RetryConfig
	apiKey      string
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithBaseURL sets the daemon base URL (default: http://127.0.0.1:4590).
func WithBaseURL(u string) ClientOption {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = hc }
}

// WithAPIKey sets an API key for authentication. The key is sent as
// an Authorization: Bearer header on every request.
func WithAPIKey(key string) ClientOption {
	return func(c *Client) { c.apiKey = key }
}

// New creates a new hawk SDK client.
//
// Note: the client performs no retries by default. Pass
// WithRetry(DefaultRetryConfig()) for production use to enable automatic
// retries with exponential backoff on transient failures.
func New(opts ...ClientOption) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
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
	httpReq.Header.Set("User-Agent", userAgent())
	c.setAuth(httpReq)

	resp, err := c.doWithRetry(ctx, httpReq, body)
	if err != nil {
		return nil, fmt.Errorf("hawk-sdk: stream request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		return nil, parseAPIError(resp)
	}

	return newStreamReader(resp), nil
}

// Sessions lists the daemon's active sessions. The daemon returns a plain
// array with no pagination envelope, so no list options are accepted.
func (c *Client) Sessions(ctx context.Context) ([]SessionSummary, error) {
	var resp []SessionSummary
	if err := c.get(ctx, "/v1/sessions", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
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
	req.Header.Set("User-Agent", userAgent())
	c.setAuth(req)

	resp, err := c.doWithRetry(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("hawk-sdk: delete request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// The daemon returns 204 No Content on delete, but older daemon versions
	// and intermediary proxies may respond with 200 OK instead. Accepting any
	// 2xx keeps this defensive and consistent with post()'s success handling.
	if resp.StatusCode/100 != 2 {
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

// setAuth sets the Authorization header if an API key is configured.
func (c *Client) setAuth(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
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
	req.Header.Set("User-Agent", userAgent())
	c.setAuth(req)

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
	req.Header.Set("User-Agent", userAgent())
	c.setAuth(req)

	resp, err := c.doWithRetry(ctx, req, data)
	if err != nil {
		return fmt.Errorf("hawk-sdk: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Accept any 2xx status: creation endpoints may return 201 Created
	// and future endpoints may use other success codes.
	if resp.StatusCode/100 != 2 {
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
