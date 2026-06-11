package hawksdk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// APIError is the base error type for all hawk API errors.
// It contains the HTTP status code, an error code string, a human-readable
// message, and optional details.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("hawk-sdk: %s [%s] (status %d)", e.Message, e.Code, e.StatusCode)
	}
	return fmt.Sprintf("hawk-sdk: %s (status %d)", e.Message, e.StatusCode)
}

// Unwrap returns nil for the base APIError.
func (e *APIError) Unwrap() error { return nil }

// BadRequestError represents a 400 Bad Request response.
type BadRequestError struct {
	APIError
}

// Unwrap allows errors.Is/As to match the underlying APIError.
func (e *BadRequestError) Unwrap() error { return &e.APIError }

// AuthenticationError represents a 401 Unauthorized response.
type AuthenticationError struct {
	APIError
}

// Unwrap allows errors.Is/As to match the underlying APIError.
func (e *AuthenticationError) Unwrap() error { return &e.APIError }

// ForbiddenError represents a 403 Forbidden response.
type ForbiddenError struct {
	APIError
}

// Unwrap allows errors.Is/As to match the underlying APIError.
func (e *ForbiddenError) Unwrap() error { return &e.APIError }

// NotFoundError represents a 404 Not Found response.
type NotFoundError struct {
	APIError
}

// Unwrap allows errors.Is/As to match the underlying APIError.
func (e *NotFoundError) Unwrap() error { return &e.APIError }

// RateLimitError represents a 429 Too Many Requests response.
// RetryAfter indicates how long to wait before retrying.
type RateLimitError struct {
	APIError
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// Error implements the error interface.
func (e *RateLimitError) Error() string {
	base := e.APIError.Error()
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%s (retry after %s)", base, e.RetryAfter)
	}
	return base
}

// Unwrap allows errors.Is/As to match the underlying APIError.
func (e *RateLimitError) Unwrap() error { return &e.APIError }

// InternalServerError represents a 500 Internal Server Error response.
type InternalServerError struct {
	APIError
}

// Unwrap allows errors.Is/As to match the underlying APIError.
func (e *InternalServerError) Unwrap() error { return &e.APIError }

// ServiceUnavailableError represents a 503 Service Unavailable response.
type ServiceUnavailableError struct {
	APIError
}

// Unwrap allows errors.Is/As to match the underlying APIError.
func (e *ServiceUnavailableError) Unwrap() error { return &e.APIError }

// parseAPIError reads the response body and returns an appropriate typed error.
func parseAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	var errResp ErrorResponse
	msg := string(body)
	code := ""
	details := ""

	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		msg = errResp.Error
		code = errResp.Code
		details = errResp.Details
	}

	base := APIError{
		StatusCode: resp.StatusCode,
		Code:       code,
		Message:    msg,
		Details:    details,
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		return &BadRequestError{APIError: base}
	case http.StatusUnauthorized:
		return &AuthenticationError{APIError: base}
	case http.StatusForbidden:
		return &ForbiddenError{APIError: base}
	case http.StatusNotFound:
		return &NotFoundError{APIError: base}
	case http.StatusTooManyRequests:
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return &RateLimitError{APIError: base, RetryAfter: retryAfter}
	case http.StatusInternalServerError:
		return &InternalServerError{APIError: base}
	case http.StatusServiceUnavailable:
		return &ServiceUnavailableError{APIError: base}
	default:
		return &base
	}
}

// parseRetryAfter parses the Retry-After header value.
// It supports both seconds (integer) and HTTP-date formats.
func parseRetryAfter(val string) time.Duration {
	if val == "" {
		return 0
	}

	// Try parsing as seconds.
	if secs, err := strconv.Atoi(val); err == nil {
		return time.Duration(secs) * time.Second
	}

	// Try parsing as HTTP-date.
	if t, err := http.ParseTime(val); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}

	return 0
}
