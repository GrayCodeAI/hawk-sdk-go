package hawksdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryOnServerError(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "server error"})
			return
		}
		json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	}))
	defer srv.Close()

	cfg := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableStatuses: []int{500},
	}
	c := New(WithBaseURL(srv.URL), WithRetry(cfg))

	resp, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("Status = %q, want %q", resp.Status, "ok")
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestRetryExhausted(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "unavailable"})
	}))
	defer srv.Close()

	cfg := RetryConfig{
		MaxRetries:        2,
		InitialBackoff:    5 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableStatuses: []int{503},
	}
	c := New(WithBaseURL(srv.URL), WithRetry(cfg))

	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	// Should have made 1 initial + 2 retries = 3 attempts.
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestRetryRespectsContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "error"})
	}))
	defer srv.Close()

	cfg := RetryConfig{
		MaxRetries:        10,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableStatuses: []int{500},
	}
	c := New(WithBaseURL(srv.URL), WithRetry(cfg))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.Health(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestRetryRespectsRetryAfterHeader(t *testing.T) {
	var attempts int32
	start := time.Now()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "0") // 0 seconds for test speed
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "rate limited"})
			return
		}
		json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	}))
	defer srv.Close()

	cfg := DefaultRetryConfig()
	cfg.InitialBackoff = 5 * time.Second // Large default, but Retry-After should override
	c := New(WithBaseURL(srv.URL), WithRetry(cfg))

	resp, err := c.Health(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("Status = %q, want %q", resp.Status, "ok")
	}
	// Should have completed quickly because Retry-After was 0.
	if elapsed > 2*time.Second {
		t.Errorf("took %v, expected to be fast with Retry-After: 0", elapsed)
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "bad request"})
	}))
	defer srv.Close()

	cfg := DefaultRetryConfig()
	cfg.InitialBackoff = 10 * time.Millisecond
	c := New(WithBaseURL(srv.URL), WithRetry(cfg))

	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	// 400 is not retryable by default, so only 1 attempt.
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1", got)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != 1*time.Second {
		t.Errorf("InitialBackoff = %v, want 1s", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want 30s", cfg.MaxBackoff)
	}
	if cfg.BackoffMultiplier != 2.0 {
		t.Errorf("BackoffMultiplier = %v, want 2.0", cfg.BackoffMultiplier)
	}
	if len(cfg.RetryableStatuses) != 5 {
		t.Errorf("RetryableStatuses length = %d, want 5", len(cfg.RetryableStatuses))
	}
}

func TestBackoffDuration(t *testing.T) {
	cfg := RetryConfig{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	// Test multiple attempts — backoff should be bounded.
	for i := 0; i < 100; i++ {
		d := cfg.backoffDuration(i)
		if d > cfg.MaxBackoff {
			t.Errorf("attempt %d: backoff %v exceeds max %v", i, d, cfg.MaxBackoff)
		}
		if d < 0 {
			t.Errorf("attempt %d: backoff %v is negative", i, d)
		}
	}
}
