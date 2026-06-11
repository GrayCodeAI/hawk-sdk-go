package hawksdk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestWorkflow_BasicSequence(t *testing.T) {
	wf, err := NewWorkflow().
		Step("double", func(ctx context.Context, input any) (any, error) {
			n := input.(int)
			return n * 2, nil
		}).
		Step("add10", func(ctx context.Context, input any) (any, error) {
			n := input.(int)
			return n + 10, nil
		}).
		Step("toString", func(ctx context.Context, input any) (any, error) {
			n := input.(int)
			return fmt.Sprintf("result: %d", n), nil
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	result, err := wf.Run(context.Background(), 5)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	want := "result: 20"
	if result != want {
		t.Errorf("Run() = %v, want %v", result, want)
	}
}

func TestWorkflow_StepRetry(t *testing.T) {
	attempts := 0

	wf, err := NewWorkflow().
		Step("flaky", func(ctx context.Context, input any) (any, error) {
			attempts++
			if attempts < 3 {
				return nil, errors.New("transient error")
			}
			return "success", nil
		}).
		WithRetry(RetryConfig{
			MaxRetries:        5,
			InitialBackoff:    5 * time.Millisecond,
			MaxBackoff:        50 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	result, err := wf.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result != "success" {
		t.Errorf("Run() = %v, want %q", result, "success")
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestWorkflow_StepTimeout(t *testing.T) {
	wf, err := NewWorkflow().
		Step("slow", func(ctx context.Context, input any) (any, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				return "done", nil
			}
		}).
		WithTimeout(50 * time.Millisecond).
		Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	_, err = wf.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestWorkflow_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	wf, err := NewWorkflow().
		Step("noop", func(ctx context.Context, input any) (any, error) {
			return "done", nil
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	_, err = wf.Run(ctx, nil)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestWorkflow_StepFailure(t *testing.T) {
	wf, err := NewWorkflow().
		Step("ok", func(ctx context.Context, input any) (any, error) {
			return "step1", nil
		}).
		Step("fail", func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("step2 failed")
		}).
		Step("never", func(ctx context.Context, input any) (any, error) {
			t.Error("step 3 should not be reached")
			return nil, nil
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	_, err = wf.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("expected step failure error")
	}
	if !errors.Is(err, errors.New("")) {
		// Just check the error contains useful info.
		if got := err.Error(); got == "" {
			t.Error("error message should not be empty")
		}
	}
}

// TestWorkflow_RetryBackoffContextCancelled verifies that cancelling the
// context while a step is sleeping between retries aborts the retry loop
// and returns the last step error (not a hang or a nil error).
func TestWorkflow_RetryBackoffContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	wf, err := NewWorkflow().
		Step("always-fails", func(ctx context.Context, input any) (any, error) {
			attempts++
			if attempts == 1 {
				// Cancel during the backoff that follows this failure.
				go func() {
					time.Sleep(20 * time.Millisecond)
					cancel()
				}()
			}
			return nil, errors.New("persistent failure")
		}).
		WithRetry(RetryConfig{
			MaxRetries:        10,
			InitialBackoff:    10 * time.Second, // long enough that cancel wins
			MaxBackoff:        10 * time.Second,
			BackoffMultiplier: 1.0,
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	start := time.Now()
	_, err = wf.Run(ctx, nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error after cancellation during backoff")
	}
	// The retry loop returns lastErr when the backoff sleep is interrupted.
	if !strings.Contains(err.Error(), "persistent failure") {
		t.Errorf("error = %v, want it to wrap the step error", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (cancel should stop retries)", attempts)
	}
	if elapsed > 5*time.Second {
		t.Errorf("Run() took %v, should abort promptly on cancellation", elapsed)
	}
}

// TestWorkflow_RetryRespectsStepTimeout verifies the interaction between
// per-step Timeout and RetryConfig: when the step timeout expires during
// retry backoff, the loop stops and reports the last step error.
func TestWorkflow_RetryRespectsStepTimeout(t *testing.T) {
	attempts := 0

	wf, err := NewWorkflow().
		Step("flaky-slow", func(ctx context.Context, input any) (any, error) {
			attempts++
			return nil, errors.New("still failing")
		}).
		WithRetry(RetryConfig{
			MaxRetries:        10,
			InitialBackoff:    30 * time.Millisecond,
			MaxBackoff:        30 * time.Millisecond,
			BackoffMultiplier: 1.0,
		}).
		WithTimeout(50 * time.Millisecond).
		Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	start := time.Now()
	_, err = wf.Run(context.Background(), nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error when step timeout expires during retries")
	}
	if !strings.Contains(err.Error(), "still failing") {
		t.Errorf("error = %v, want it to wrap the step error", err)
	}
	if attempts >= 10 {
		t.Errorf("attempts = %d, timeout should have stopped retries early", attempts)
	}
	if elapsed > 2*time.Second {
		t.Errorf("Run() took %v, should stop near the 50ms step timeout", elapsed)
	}
}

// TestWorkflow_RetryTimeoutBeforeFirstAttempt verifies that an
// already-expired step context surfaces the context error when the step
// never ran (lastErr == nil path in executeStep's select).
func TestWorkflow_RetryTimeoutBeforeFirstAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ran := false
	wf, err := NewWorkflow().
		Step("never-runs", func(ctx context.Context, input any) (any, error) {
			ran = true
			return "ok", nil
		}).
		WithRetry(RetryConfig{
			MaxRetries:        3,
			InitialBackoff:    time.Millisecond,
			MaxBackoff:        time.Millisecond,
			BackoffMultiplier: 1.0,
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Run checks ctx before the step, so use executeStep directly to hit the
	// retry loop's own cancellation check.
	_, stepErr := executeStep(ctx, wf.steps[0], nil)
	if !errors.Is(stepErr, context.Canceled) {
		t.Errorf("executeStep() error = %v, want context.Canceled", stepErr)
	}
	if ran {
		t.Error("step should not run when context is already cancelled")
	}
}

func TestWorkflowBuilder_EmptyWorkflow(t *testing.T) {
	_, err := NewWorkflow().Build()
	if err == nil {
		t.Fatal("expected error for empty workflow")
	}
}

func TestWorkflowBuilder_WithRetryBeforeStep(t *testing.T) {
	_, err := NewWorkflow().WithRetry(DefaultRetryConfig()).Build()
	if err == nil {
		t.Fatal("expected error for WithRetry before any Step")
	}
}

func TestWorkflowBuilder_WithTimeoutBeforeStep(t *testing.T) {
	_, err := NewWorkflow().WithTimeout(time.Second).Build()
	if err == nil {
		t.Fatal("expected error for WithTimeout before any Step")
	}
}

func TestWorkflow_DataPassingBetweenSteps(t *testing.T) {
	wf, err := NewWorkflow().
		Step("produce", func(ctx context.Context, input any) (any, error) {
			return map[string]int{"count": 42}, nil
		}).
		Step("consume", func(ctx context.Context, input any) (any, error) {
			m := input.(map[string]int)
			return m["count"] * 2, nil
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	result, err := wf.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result != 84 {
		t.Errorf("Run() = %v, want 84", result)
	}
}
