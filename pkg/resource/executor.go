package resource

import (
	"context"
	"errors"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Operation int

const (
	Skip Operation = 1 << iota
	Fetch
	Create
	Update
	Check
)

var opsMap = map[Operation]string{
	Skip:   "Skip",
	Fetch:  "Fetch",
	Create: "Create",
	Update: "Update",
	Check:  "Check",
}

func (o Operation) String() string {
	name, ok := opsMap[o]
	if !ok {
		return "Unknown"
	}
	return name
}

var (
	MissingHandlerResourcesErr = errors.New("missing handler resources, cannot perform handling")
	OperationNotReadyErr       = errors.New("operation not ready yet")
)

// Executor simplifies synchronization logic for a requested resource.
// It exposes a simple Execute method which processes resource lifecycle.
type Executor[T client.Object] struct {
	Name       string
	FetchFunc  func(ctx context.Context, obj T) error
	CreateFunc func(ctx context.Context, obj T) error
	UpdateFunc func(ctx context.Context, obj T) (bool, error)
	IsReady    func(ctx context.Context, obj T) bool

	// nop indicates that no operation will be performed during Execute.
	// Specify when Executor should do nothing.
	// Private field and can only be used with NopExecutor.
	nop bool
}

// NopExecutor is noop executor for workflows. It can be used to indicate
// that requested operation is valid, but nothing to execute.
func NopExecutor[T client.Object](name string) Executor[T] {
	return Executor[T]{
		Name: name,
		nop:  true,
	}
}

// Execute performs the resource execution by invoking Executor functions in ordered manner.
// Returns an error if data is missing or for runtime operations.
// Returns all the operations performed during execution.
func (h *Executor[T]) Execute(ctx context.Context, obj T) ExecuteResult {
	results := ExecuteResult{
		name:   h.Name,
		object: obj,
	}

	// Validate
	if h.nop {
		return results.Add(Skip)
	}
	if op, err := h.validate(); err != nil {
		return results.AddWithErr(op, err)
	}

	// Fetch the object
	shouldCreate := false
	if err := h.FetchFunc(ctx, obj); err != nil {
		if client.IgnoreNotFound(err) == nil {
			shouldCreate = true // not found, mark
			results.Add(Fetch)
		} else {
			return results.AddWithErr(Fetch, fmt.Errorf("failed to fetch object %T: %w", obj, err)) // critical fetch error occurred
		}
	}

	// Create object if marked for creation
	if shouldCreate {
		if err := h.CreateFunc(ctx, obj); err != nil {
			return results.AddWithErr(Create, fmt.Errorf("failed to create object %T: %w", obj, err)) // critical create error occurred
		} else {
			results.Add(Create)
		}
	}

	// Update object
	updated, err := h.UpdateFunc(ctx, obj)
	if err != nil {
		return results.AddWithErr(Update, fmt.Errorf("failed to update object %T: %w", obj, err)) // critical update error occurred
	} else if updated {
		results.Add(Update)
	}

	// Check if object is ready
	if h.IsReady(ctx, obj) {
		return results.Add(Check)
	}
	return results.AddWithErr(Check, OperationNotReadyErr)
}

func (h *Executor[T]) validate() (Operation, error) {
	switch {
	case h.FetchFunc == nil:
		return Fetch, MissingHandlerResourcesErr
	case h.CreateFunc == nil:
		return Create, MissingHandlerResourcesErr
	case h.UpdateFunc == nil:
		return Update, MissingHandlerResourcesErr
	case h.IsReady == nil:
		return Check, MissingHandlerResourcesErr
	}
	return Skip, nil
}

// ExecuteResult defines a wrapper for multiple Operations performed by Executor.Execute.
type ExecuteResult struct {
	name   string
	object client.Object
	op     Operation
	err    error
}

func (r ExecuteResult) Name() string { return r.name }

func (r ExecuteResult) Object() client.Object { return r.object }

// Add adds a successful Operation to ExecuteResult
func (r ExecuteResult) Add(op Operation) ExecuteResult {
	return r.AddWithErr(op, nil)
}

// AddWithErr adds a failed Operation with its error to ExecuteResult
func (r ExecuteResult) AddWithErr(op Operation, err error) ExecuteResult {
	r.op = r.op | op
	r.err = err
	return r
}

// Error returns the errored operation
func (r ExecuteResult) Error() error {
	if r.err != nil && r.err != OperationNotReadyErr {
		return r.err
	}
	return nil
}

func (r ExecuteResult) Skipped() bool {
	return r.op&Skip != 0
}

func (r ExecuteResult) Ready() bool {
	return r.op&Check != 0 && r.err == nil
}

func (r ExecuteResult) NotReady() bool {
	return r.op&Check != 0 && r.err == OperationNotReadyErr
}

// ProcessingOps returns processing operations performed.
func (r ExecuteResult) ProcessingOps() []string {
	results := make([]string, 0, 2)
	if r.op&Create != 0 {
		results = append(results, opsMap[Create])
	}
	if r.op&Update != 0 {
		results = append(results, opsMap[Update])
	}
	return results
}
