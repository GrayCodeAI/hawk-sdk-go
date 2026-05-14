package hawksdk

import (
	"context"
	"errors"
	"fmt"
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
