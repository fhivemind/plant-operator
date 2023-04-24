package resource

import (
	"context"
	"errors"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type HandleState string

const (
	Fetch    HandleState = "Fetch"
	Create   HandleState = "Create"
	Update   HandleState = "Update"
	NotReady HandleState = "NotReady"
	Ready    HandleState = "Ready"
)

func (s HandleState) Done() bool { return s == Ready }

func (s HandleState) OperationName() string {
	switch s {
	case NotReady, Ready:
		return "Runtime Check"
	}
	return string(s)
}

var MissingHandlerResourcesErr = errors.New("missing handler resources, cannot perform handling")

// Handler simplifies synchronization logic for a requested resource.
// It exposes a simple Handle method which processes resource lifecycle.
type Handler[T client.Object] struct {
	Name       string
	FetchFunc  func(ctx context.Context, obj T) error
	CreateFunc func(ctx context.Context, obj T) error
	UpdateFunc func(ctx context.Context, obj T) (bool, error)
	IsReady    func(ctx context.Context, obj T) bool

	// Nop indicates that no operation will be performed during Handle.
	// Specify when Handler should do nothing.
	Nop bool
}

// NopHandler is noop executor for workflows. It can be used to indicate
// that requested operation is valid, but nothing to execute.
func NopHandler[T client.Object](name string) Handler[T] {
	return Handler[T]{
		Name: name,
		Nop:  true,
	}
}

// Handle performs the workflow handling by invoking Handler functions in ordered manner.
// Returns an error if data is missing or during operation failures.
// TODO: Use eventing instead of logging to better inform about changes.
func (h *Handler[T]) Handle(ctx context.Context, obj T) (HandleState, error) {
	logger := log.FromContext(ctx)

	// Validate
	if h.Nop {
		return Ready, nil
	}
	if op, err := h.validate(); err != nil {
		return op, err
	}

	// Fetch the object
	shouldCreate := false
	if err := h.FetchFunc(ctx, obj); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info(fmt.Sprintf("Marked object %T for CREATE", obj))
			shouldCreate = true // not found, mark
		} else {
			return Fetch, fmt.Errorf("failed to fetch object %T: %w", obj, err) // critical fetch error occurred
		}
	}

	// Create object if marked for creation
	if shouldCreate {
		if err := h.CreateFunc(ctx, obj); err != nil {
			return Create, fmt.Errorf("failed to create object %T: %w", obj, err) // critical create error occurred
		} else {
			logger.Info(fmt.Sprintf("Successfully ran CREATE for object %T", obj))
		}
	}

	// Update object
	updated, err := h.UpdateFunc(ctx, obj)
	if err != nil {
		return Update, fmt.Errorf("failed to update object %T: %w", obj, err) // critical update error occurred
	} else if updated {
		logger.Info(fmt.Sprintf("Successfully ran UPDATE for object %T", obj))
	}

	// Check if object is ready
	if h.IsReady(ctx, obj) {
		return Ready, nil
	}
	return NotReady, nil
}

func (h *Handler[T]) validate() (HandleState, error) {
	switch {
	case h.FetchFunc == nil:
		return Fetch, MissingHandlerResourcesErr
	case h.CreateFunc == nil:
		return Create, MissingHandlerResourcesErr
	case h.UpdateFunc == nil:
		return Update, MissingHandlerResourcesErr
	case h.IsReady == nil:
		return Ready, MissingHandlerResourcesErr
	}
	return "", nil
}
