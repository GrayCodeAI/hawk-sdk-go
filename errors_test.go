package hawksdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// 1. Error message formatting for each type
// ---------------------------------------------------------------------------

func TestAPIError_ErrorMessage(t *testing.T) {
	t.Run("with_code", func(t *testing.T) {
		e := &APIError{StatusCode: 400, Code: "bad_request", Message: "invalid input"}
		want := "hawk-sdk: invalid input [bad_request] (status 400)"
		if got := e.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without_code", func(t *testing.T) {
		e := &APIError{StatusCode: 500, Message: "server error"}
		want := "hawk-sdk: server error (status 500)"
		if got := e.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

func TestSubtypeErrorMessages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "BadRequestError",
			err:  &BadRequestError{APIError: APIError{StatusCode: 400, Code: "bad_request", Message: "bad input"}},
			want: "hawk-sdk: bad input [bad_request] (status 400)",
		},
		{
			name: "AuthenticationError",
			err:  &AuthenticationError{APIError: APIError{StatusCode: 401, Code: "unauthorized", Message: "no key"}},
			want: "hawk-sdk: no key [unauthorized] (status 401)",
		},
		{
			name: "ForbiddenError",
			err:  &ForbiddenError{APIError: APIError{StatusCode: 403, Code: "forbidden", Message: "denied"}},
			want: "hawk-sdk: denied [forbidden] (status 403)",
		},
		{
			name: "NotFoundError",
			err:  &NotFoundError{APIError: APIError{StatusCode: 404, Code: "not_found", Message: "gone"}},
			want: "hawk-sdk: gone [not_found] (status 404)",
		},
		{
			name: "InternalServerError",
			err:  &InternalServerError{APIError: APIError{StatusCode: 500, Code: "internal", Message: "boom"}},
			want: "hawk-sdk: boom [internal] (status 500)",
		},
		{
			name: "ServiceUnavailableError",
			err:  &ServiceUnavailableError{APIError: APIError{StatusCode: 503, Code: "unavailable", Message: "down"}},
			want: "hawk-sdk: down [unavailable] (status 503)",
		},
		{
			name: "RateLimitError_without_retry",
			err:  &RateLimitError{APIError: APIError{StatusCode: 429, Code: "rate_limited", Message: "slow down"}},
			want: "hawk-sdk: slow down [rate_limited] (status 429)",
		},
		{
			name: "RateLimitError_with_retry",
			err:  &RateLimitError{APIError: APIError{StatusCode: 429, Code: "rate_limited", Message: "slow down"}, RetryAfter: 30 * time.Second},
			want: "hawk-sdk: slow down [rate_limited] (status 429) (retry after 30s)",
		},
		{
			name: "RateLimitError_without_code_with_retry",
			err:  &RateLimitError{APIError: APIError{StatusCode: 429, Message: "rate limited"}, RetryAfter: 2 * time.Minute},
			want: "hawk-sdk: rate limited (status 429) (retry after 2m0s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. Unwrap behavior
// ---------------------------------------------------------------------------

func TestAPIError_Unwrap(t *testing.T) {
	e := &APIError{StatusCode: 500, Message: "err"}
	if e.Unwrap() != nil {
		t.Error("APIError.Unwrap() should return nil")
	}
}

func TestSubtypeUnwrap(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect func(error) bool
	}{
		{
			name:   "BadRequestError",
			err:    &BadRequestError{APIError: APIError{StatusCode: 400, Message: "bad"}},
			expect: func(e error) bool { _, ok := e.(*APIError); return ok },
		},
		{
			name:   "AuthenticationError",
			err:    &AuthenticationError{APIError: APIError{StatusCode: 401, Message: "auth"}},
			expect: func(e error) bool { _, ok := e.(*APIError); return ok },
		},
		{
			name:   "ForbiddenError",
			err:    &ForbiddenError{APIError: APIError{StatusCode: 403, Message: "forbidden"}},
			expect: func(e error) bool { _, ok := e.(*APIError); return ok },
		},
		{
			name:   "NotFoundError",
			err:    &NotFoundError{APIError: APIError{StatusCode: 404, Message: "not found"}},
			expect: func(e error) bool { _, ok := e.(*APIError); return ok },
		},
		{
			name:   "RateLimitError",
			err:    &RateLimitError{APIError: APIError{StatusCode: 429, Message: "limited"}},
			expect: func(e error) bool { _, ok := e.(*APIError); return ok },
		},
		{
			name:   "InternalServerError",
			err:    &InternalServerError{APIError: APIError{StatusCode: 500, Message: "boom"}},
			expect: func(e error) bool { _, ok := e.(*APIError); return ok },
		},
		{
			name:   "ServiceUnavailableError",
			err:    &ServiceUnavailableError{APIError: APIError{StatusCode: 503, Message: "down"}},
			expect: func(e error) bool { _, ok := e.(*APIError); return ok },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uw := errors.Unwrap(tt.err)
			if uw == nil {
				t.Fatal("Unwrap returned nil")
			}
			if !tt.expect(uw) {
				t.Errorf("Unwrap returned %T, want *APIError", uw)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. parseAPIError for each status code (JSON body + non-JSON fallback)
// ---------------------------------------------------------------------------

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

			// Every typed error must unwrap to APIError.
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Errorf("expected error to wrap APIError, got %T: %v", err, err)
			}
			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
			if apiErr.Message != tt.body.Error {
				t.Errorf("Message = %q, want %q", apiErr.Message, tt.body.Error)
			}
			if apiErr.Code != tt.body.Code {
				t.Errorf("Code = %q, want %q", apiErr.Code, tt.body.Code)
			}

			// Assert the concrete type.
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

func TestParseAPIError_NonJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal Server Error")
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
	// When JSON parsing fails, the raw body becomes the message.
	if apiErr.Message != "Internal Server Error" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Internal Server Error")
	}
	if apiErr.Code != "" {
		t.Errorf("Code = %q, want empty", apiErr.Code)
	}
}

func TestParseAPIError_UnknownStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(418) // I'm a teapot
		json.NewEncoder(w).Encode(ErrorResponse{Error: "teapot", Code: "teapot"})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Unknown status codes should produce a plain *APIError.
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 418 {
		t.Errorf("StatusCode = %d, want 418", apiErr.StatusCode)
	}

	// Must NOT match any specific error type.
	var badReq *BadRequestError
	if errors.As(err, &badReq) {
		t.Error("unknown status code should not match BadRequestError")
	}
}

func TestParseAPIError_WithDetailsField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"validation failed","code":"invalid_field","details":"field 'prompt' is required"}`))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Details != "field 'prompt' is required" {
		t.Errorf("Details = %q, want %q", apiErr.Details, "field 'prompt' is required")
	}
}

// ---------------------------------------------------------------------------
// 4. RetryAfter parsing
// ---------------------------------------------------------------------------

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want time.Duration
	}{
		{name: "empty", val: "", want: 0},
		{name: "seconds", val: "10", want: 10 * time.Second},
		{name: "zero_seconds", val: "0", want: 0},
		{name: "large_value", val: "300", want: 300 * time.Second},
		{name: "invalid_string", val: "abc", want: 0},
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

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	// Use a future time so the duration is positive.
	future := time.Now().Add(60 * time.Second).UTC()
	dateStr := future.Format(http.TimeFormat)

	got := parseRetryAfter(dateStr)
	// Should be approximately 60s; allow 5s tolerance for test execution time.
	if got < 55*time.Second || got > 65*time.Second {
		t.Errorf("parseRetryAfter(HTTP-date) = %v, want ~60s", got)
	}
}

func TestParseRetryAfter_HTTPDatePast(t *testing.T) {
	// A past date should yield 0 (negative duration treated as 0).
	past := time.Now().Add(-60 * time.Second).UTC()
	dateStr := past.Format(http.TimeFormat)

	got := parseRetryAfter(dateStr)
	if got != 0 {
		t.Errorf("parseRetryAfter(past HTTP-date) = %v, want 0", got)
	}
}

func TestRateLimitError_RetryAfter(t *testing.T) {
	t.Run("seconds_header", func(t *testing.T) {
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
	})

	t.Run("no_header", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			t.Fatalf("expected RateLimitError, got %T", err)
		}
		if rlErr.RetryAfter != 0 {
			t.Errorf("RetryAfter = %v, want 0", rlErr.RetryAfter)
		}
		// Message should NOT contain "retry after" when duration is 0.
		if strings.Contains(rlErr.Error(), "retry after") {
			t.Errorf("Error() should not mention retry when RetryAfter=0: %q", rlErr.Error())
		}
	})

	t.Run("http_date_header", func(t *testing.T) {
		future := time.Now().Add(120 * time.Second).UTC()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", future.Format(http.TimeFormat))
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
			t.Fatalf("expected RateLimitError, got %T", err)
		}
		// Allow tolerance for test execution time.
		if rlErr.RetryAfter < 115*time.Second || rlErr.RetryAfter > 125*time.Second {
			t.Errorf("RetryAfter = %v, want ~120s", rlErr.RetryAfter)
		}
	})
}

// ---------------------------------------------------------------------------
// 5. Error type assertions (errors.Is / errors.As)
// ---------------------------------------------------------------------------

func TestErrorsAs_AllTypes(t *testing.T) {
	// errors.As should match the concrete type and also *APIError via Unwrap.
	errs := []struct {
		name string
		err  error
	}{
		{"BadRequestError", &BadRequestError{APIError: APIError{StatusCode: 400, Message: "bad"}}},
		{"AuthenticationError", &AuthenticationError{APIError: APIError{StatusCode: 401, Message: "auth"}}},
		{"ForbiddenError", &ForbiddenError{APIError: APIError{StatusCode: 403, Message: "denied"}}},
		{"NotFoundError", &NotFoundError{APIError: APIError{StatusCode: 404, Message: "missing"}}},
		{"RateLimitError", &RateLimitError{APIError: APIError{StatusCode: 429, Message: "slow"}}},
		{"InternalServerError", &InternalServerError{APIError: APIError{StatusCode: 500, Message: "boom"}}},
		{"ServiceUnavailableError", &ServiceUnavailableError{APIError: APIError{StatusCode: 503, Message: "down"}}},
	}

	for _, tt := range errs {
		t.Run(tt.name+"/as_concrete_type", func(t *testing.T) {
			// Each concrete type should match itself.
			switch e := tt.err.(type) {
			case *BadRequestError:
				var target *BadRequestError
				if !errors.As(tt.err, &target) {
					t.Error("errors.As failed for *BadRequestError")
				}
				_ = e
			case *AuthenticationError:
				var target *AuthenticationError
				if !errors.As(tt.err, &target) {
					t.Error("errors.As failed for *AuthenticationError")
				}
			case *ForbiddenError:
				var target *ForbiddenError
				if !errors.As(tt.err, &target) {
					t.Error("errors.As failed for *ForbiddenError")
				}
			case *NotFoundError:
				var target *NotFoundError
				if !errors.As(tt.err, &target) {
					t.Error("errors.As failed for *NotFoundError")
				}
			case *RateLimitError:
				var target *RateLimitError
				if !errors.As(tt.err, &target) {
					t.Error("errors.As failed for *RateLimitError")
				}
			case *InternalServerError:
				var target *InternalServerError
				if !errors.As(tt.err, &target) {
					t.Error("errors.As failed for *InternalServerError")
				}
			case *ServiceUnavailableError:
				var target *ServiceUnavailableError
				if !errors.As(tt.err, &target) {
					t.Error("errors.As failed for *ServiceUnavailableError")
				}
			}
		})

		t.Run(tt.name+"/as_APIError_via_Unwrap", func(t *testing.T) {
			// errors.As should also find *APIError through Unwrap.
			var apiErr *APIError
			if !errors.As(tt.err, &apiErr) {
				t.Errorf("errors.As(*APIError) failed for %T via Unwrap", tt.err)
			}
			if apiErr.Message == "" {
				t.Error("APIError.Message should not be empty")
			}
		})
	}
}

func TestErrorsAs_DoesNotMatchWrongType(t *testing.T) {
	err := &NotFoundError{APIError: APIError{StatusCode: 404, Message: "missing"}}

	var badReq *BadRequestError
	if errors.As(err, &badReq) {
		t.Error("NotFoundError should not match BadRequestError")
	}

	var rateLimit *RateLimitError
	if errors.As(err, &rateLimit) {
		t.Error("NotFoundError should not match RateLimitError")
	}

	var auth *AuthenticationError
	if errors.As(err, &auth) {
		t.Error("NotFoundError should not match AuthenticationError")
	}
}

func TestErrorsIs_WrappedAPIError(t *testing.T) {
	// Since APIError has no sentinel value, errors.Is with pointer equality
	// tests the Unwrap chain. Verify that errors.Is works for the nil-case
	// (APIError.Unwrap returns nil, so errors.Is(err, nil) walks the chain).
	base := &APIError{StatusCode: 500, Message: "boom"}
	if errors.Is(base, nil) {
		t.Error("errors.Is(APIError, nil) should be false")
	}

	// errors.Is with a specific *APIError pointer should match only if same pointer.
	same := base
	if !errors.Is(base, same) {
		t.Error("errors.Is should match identical pointer")
	}

	other := &APIError{StatusCode: 500, Message: "boom"}
	if errors.Is(base, other) {
		t.Error("errors.Is should not match different *APIError with same fields")
	}
}

func TestErrorsAs_APIErrorDirectly(t *testing.T) {
	// A bare *APIError (not a subtype) should also match via errors.As.
	err := &APIError{StatusCode: 422, Code: "unprocessable", Message: "bad payload"}
	var target *APIError
	if !errors.As(err, &target) {
		t.Fatal("errors.As(*APIError) failed for direct *APIError")
	}
	if target.StatusCode != 422 {
		t.Errorf("StatusCode = %d, want 422", target.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Integration: parseAPIError via full HTTP round-trip
// ---------------------------------------------------------------------------

func TestParseAPIError_JSONFieldsPopulated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"insufficient scope","code":"forbidden","details":"need admin role"}`))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var forbidden *ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Fatalf("expected *ForbiddenError, got %T", err)
	}
	if forbidden.Message != "insufficient scope" {
		t.Errorf("Message = %q, want %q", forbidden.Message, "insufficient scope")
	}
	if forbidden.Code != "forbidden" {
		t.Errorf("Code = %q, want %q", forbidden.Code, "forbidden")
	}
	if forbidden.Details != "need admin role" {
		t.Errorf("Details = %q, want %q", forbidden.Details, "need admin role")
	}
}
