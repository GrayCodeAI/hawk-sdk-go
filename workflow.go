package hawksdk

import (
	"context"
	"fmt"
	"time"
)

// StepFunc is the function signature for a workflow step.
// It receives context and an input value, returning an output value or error.
type StepFunc func(ctx context.Context, input any) (any, error)

// Step represents a single unit of work in a workflow.
type Step struct {
	// Name identifies this step for logging and debugging.
	Name string

	// Fn is the function to execute for this step.
	Fn StepFunc

	// RetryConfig optionally configures per-step retry behavior.
	RetryConfig *RetryConfig

	// Timeout optionally sets a deadline for this step.
	Timeout time.Duration
}

// Workflow represents a sequence of steps to execute in order,
// passing the output of each step as input to the next.
type Workflow struct {
	steps []Step
}

// Run executes the workflow steps sequentially, threading each step's output
// as the next step's input. It begins with initialInput and returns the
// final step's output.
func (w *Workflow) Run(ctx context.Context, initialInput any) (any, error) {
	current := initialInput

	for i, step := range w.steps {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("hawk-sdk: workflow cancelled before step %q: %w", step.Name, ctx.Err())
		default:
		}

		result, err := executeStep(ctx, step, current)
		if err != nil {
			return nil, fmt.Errorf("hawk-sdk: workflow failed at step %d (%q): %w", i+1, step.Name, err)
		}
		current = result
	}

	return current, nil
}

// executeStep runs a single step with its configured timeout and retry logic.
func executeStep(ctx context.Context, step Step, input any) (any, error) {
	// Apply step-level timeout if configured.
	stepCtx := ctx
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	// No retry config — just run once.
	if step.RetryConfig == nil {
		return step.Fn(stepCtx, input)
	}

	cfg := step.RetryConfig
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		select {
		case <-stepCtx.Done():
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, stepCtx.Err()
		default:
		}

		result, err := step.Fn(stepCtx, input)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if attempt < cfg.MaxRetries {
			backoff := cfg.backoffDuration(attempt)
			if sleepErr := sleepWithContext(stepCtx, backoff); sleepErr != nil {
				return nil, lastErr
			}
		}
	}

	return nil, lastErr
}

// WorkflowBuilder provides a fluent API for constructing workflows.
type WorkflowBuilder struct {
	steps []Step
	err   error
}

// NewWorkflow creates a new WorkflowBuilder.
func NewWorkflow() *WorkflowBuilder {
	return &WorkflowBuilder{}
}

// Step adds a named step to the workflow.
func (wb *WorkflowBuilder) Step(name string, fn StepFunc) *WorkflowBuilder {
	wb.steps = append(wb.steps, Step{
		Name: name,
		Fn:   fn,
	})
	return wb
}

// WithRetry sets the retry configuration for the most recently added step.
// Must be called immediately after Step().
func (wb *WorkflowBuilder) WithRetry(cfg RetryConfig) *WorkflowBuilder {
	if len(wb.steps) == 0 {
		wb.err = fmt.Errorf("hawk-sdk: WithRetry called before any Step")
		return wb
	}
	wb.steps[len(wb.steps)-1].RetryConfig = &cfg
	return wb
}

// WithTimeout sets the timeout for the most recently added step.
// Must be called immediately after Step() or WithRetry().
func (wb *WorkflowBuilder) WithTimeout(d time.Duration) *WorkflowBuilder {
	if len(wb.steps) == 0 {
		wb.err = fmt.Errorf("hawk-sdk: WithTimeout called before any Step")
		return wb
	}
	wb.steps[len(wb.steps)-1].Timeout = d
	return wb
}

// Build finalizes the workflow and returns it. Returns an error if the builder
// was configured incorrectly.
func (wb *WorkflowBuilder) Build() (*Workflow, error) {
	if wb.err != nil {
		return nil, wb.err
	}
	if len(wb.steps) == 0 {
		return nil, fmt.Errorf("hawk-sdk: workflow must have at least one step")
	}
	return &Workflow{steps: wb.steps}, nil
}
