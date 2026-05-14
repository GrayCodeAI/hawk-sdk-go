package hawksdk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseAPIError_TypedErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       ErrorResponse
		wantType   string
	}{
		{
			name:       "400 BadRequest",
			statusCode: http.StatusBadRequest,
			body:       ErrorResponse{Error: "invalid prompt", Code: "bad_request"},
			wantType:   "BadRequestError",
		},
		{
			name:       "401 Authentication",
			statusCode: http.StatusUnauthorized,
			body:       ErrorResponse{Error: "invalid api key", Code: "unauthorized"},
			wantType:   "AuthenticationError",
		},
		{
			name:       "403 Forbidden",
			statusCode: http.StatusForbidden,
			body:       ErrorResponse{Error: "access denied", Code: "forbidden"},
			wantType:   "ForbiddenError",
		},
		{
			name:       "404 NotFound",
			statusCode: http.StatusNotFound,
			body:       ErrorResponse{Error: "resource not found", Code: "not_found"},
			wantType:   "NotFoundError",
		},
		{
			name:       "429 RateLimit",
			statusCode: http.StatusTooManyRequests,
			body:       ErrorResponse{Error: "rate limit exceeded", Code: "rate_limited"},
			wantType:   "RateLimitError",
		},
		{
			name:       "500 InternalServer",
			statusCode: http.StatusInternalServerError,
			body:       ErrorResponse{Error: "internal error", Code: "internal"},
			wantType:   "InternalServerError",
		},
		{
			name:       "503 ServiceUnavailable",
			statusCode: http.StatusServiceUnavailable,
			body:       ErrorResponse{Error: "service down", Code: "unavailable"},
			wantType:   "ServiceUnavailableError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.body)
			}))
			defer srv.Close()

			c := New(WithBaseURL(srv.URL))
			_, err := c.Health(context.Background())
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Check the error wraps APIError.
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Errorf("expected error to wrap APIError, got %T: %v", err, err)
			}

			switch tt.wantType {
			case "BadRequestError":
				var e *BadRequestError
				if !errors.As(err, &e) {
					t.Errorf("expected BadRequestError, got %T", err)
				}
			case "AuthenticationError":
				var e *AuthenticationError
				if !errors.As(err, &e) {
					t.Errorf("expected AuthenticationError, got %T", err)
				}
			case "ForbiddenError":
				var e *ForbiddenError
				if !errors.As(err, &e) {
					t.Errorf("expected ForbiddenError, got %T", err)
				}
			case "NotFoundError":
				var e *NotFoundError
				if !errors.As(err, &e) {
					t.Errorf("expected NotFoundError, got %T", err)
				}
			case "RateLimitError":
				var e *RateLimitError
				if !errors.As(err, &e) {
					t.Errorf("expected RateLimitError, got %T", err)
				}
			case "InternalServerError":
				var e *InternalServerError
				if !errors.As(err, &e) {
					t.Errorf("expected InternalServerError, got %T", err)
				}
			case "ServiceUnavailableError":
				var e *ServiceUnavailableError
				if !errors.As(err, &e) {
					t.Errorf("expected ServiceUnavailableError, got %T", err)
				}
			}
		})
	}
}

func TestRateLimitError_RetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "rate limited"})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if rlErr.RetryAfter != 5*time.Second {
		t.Errorf("RetryAfter = %v, want 5s", rlErr.RetryAfter)
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want time.Duration
	}{
		{name: "empty", val: "", want: 0},
		{name: "seconds", val: "10", want: 10 * time.Second},
		{name: "invalid", val: "abc", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRetryAfter(tt.val)
			if got != tt.want {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestAPIError_ErrorMessage(t *testing.T) {
	e := &APIError{StatusCode: 400, Code: "bad_request", Message: "invalid input"}
	want := "hawk-sdk: invalid input [bad_request] (status 400)"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	// Without code.
	e2 := &APIError{StatusCode: 500, Message: "server error"}
	want2 := "hawk-sdk: server error (status 500)"
	if got := e2.Error(); got != want2 {
		t.Errorf("Error() = %q, want %q", got, want2)
	}
}
