package hawksdk

import (
	"bytes"
	"context"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// RetryConfig configures the automatic retry behavior for API requests.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int

	// InitialBackoff is the base delay before the first retry.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum delay between retries.
	MaxBackoff time.Duration

	// BackoffMultiplier controls exponential growth of the backoff.
	BackoffMultiplier float64

	// RetryableStatuses lists HTTP status codes that should trigger a retry.
	RetryableStatuses []int
}

// DefaultRetryConfig returns the default retry configuration:
// 3 retries, 1s initial backoff, 30s max, 2x multiplier, retry on 429/500/502/503.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableStatuses: []int{
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
		},
	}
}

// WithRetry sets a retry configuration on the client.
func WithRetry(config RetryConfig) ClientOption {
	return func(c *Client) {
		c.retryConfig = &config
	}
}

// isRetryable checks if the given status code is in the retryable set.
func (cfg *RetryConfig) isRetryable(statusCode int) bool {
	for _, s := range cfg.RetryableStatuses {
		if s == statusCode {
			return true
		}
	}
	return false
}

// backoffDuration computes the backoff duration for the given attempt (0-indexed).
// It uses exponential backoff with full jitter.
func (cfg *RetryConfig) backoffDuration(attempt int) time.Duration {
	backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffMultiplier, float64(attempt))
	if backoff > float64(cfg.MaxBackoff) {
		backoff = float64(cfg.MaxBackoff)
	}
	// Full jitter: random value between 0 and calculated backoff.
	jittered := time.Duration(rand.Float64() * backoff)
	return jittered
}

// sleepWithContext sleeps for the specified duration, respecting context cancellation.
// Returns the context error if the context is cancelled during the sleep.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// doWithRetry executes the given HTTP request with retry logic.
// It returns the response from the first successful attempt or the last failed attempt.
func (c *Client) doWithRetry(ctx context.Context, req *http.Request, body []byte) (*http.Response, error) {
	cfg := c.retryConfig
	if cfg == nil {
		return c.httpClient.Do(req)
	}

	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Clone the request for each attempt (body needs to be re-readable).
		attemptReq := req.Clone(ctx)
		if body != nil {
			attemptReq.Body = io.NopCloser(bytes.NewReader(body))
			attemptReq.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(body)), nil
			}
		}

		resp, err := c.httpClient.Do(attemptReq)
		if err != nil {
			// Network error — retryable.
			lastErr = err
			if attempt < cfg.MaxRetries {
				if sleepErr := sleepWithContext(ctx, cfg.backoffDuration(attempt)); sleepErr != nil {
					return nil, sleepErr
				}
				continue
			}
			return nil, lastErr
		}

		// Success — not retryable.
		if !cfg.isRetryable(resp.StatusCode) {
			return resp, nil
		}

		// Retryable status — determine backoff.
		lastResp = resp
		lastErr = nil

		if attempt < cfg.MaxRetries {
			// For 429, respect Retry-After header if present.
			backoff := cfg.backoffDuration(attempt)
			if resp.StatusCode == http.StatusTooManyRequests {
				if raHeader := resp.Header.Get("Retry-After"); raHeader != "" {
					backoff = parseRetryAfter(raHeader)
				}
			}

			// Drain body before retry to allow connection reuse.
			resp.Body.Close()

			if sleepErr := sleepWithContext(ctx, backoff); sleepErr != nil {
				return nil, sleepErr
			}
			continue
		}
	}

	// All retries exhausted — return the last response.
	if lastResp != nil {
		return lastResp, nil
	}
	return nil, lastErr
}

