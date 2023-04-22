package resource

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type HandleState string

const (
	Fetch    HandleState = "Fetch"
	Create   HandleState = "Create"
	Update   HandleState = "Update"
	NotReady HandleState = "Not ready"
	Ready    HandleState = "Ready"
)

func (s HandleState) Done() bool { return s == Ready }
func (s HandleState) OperationName() string {
	switch s {
	case NotReady, Ready:
		return "Check"
	}
	return string(s)
}

// Handler simplifies synchronization logic for a requested resource.
// It exposes a simple Handle method which processes resource lifecycle.
type Handler[T client.Object] struct {
	Name       string
	FetchFunc  func(obj T) error
	CreateFunc func(obj T) error
	UpdateFunc func(obj T) (bool, error)
	IsReady    func(obj T) bool
}

func (h *Handler[T]) Handle(ctx context.Context, obj T) (HandleState, error) {
	logger := log.FromContext(ctx)

	// Fetch the object
	shouldCreate := false
	if err := h.FetchFunc(obj); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info(fmt.Sprintf("Marked object %T for CREATE", obj))
			shouldCreate = true // not found, mark
		} else {
			return Fetch, fmt.Errorf("failed to fetch object %T: %w", obj, err) // critical fetch error occurred
		}
	}

	// Create object if marked for creation
	if shouldCreate {
		if err := h.CreateFunc(obj); err != nil {
			return Create, fmt.Errorf("failed to create object %T: %w", obj, err) // critical create error occurred
		} else {
			logger.Info(fmt.Sprintf("Successfully ran CREATE for object %T", obj))
		}
	}

	// Update object
	updated, err := h.UpdateFunc(obj)
	if err != nil {
		return Update, fmt.Errorf("failed to update object %T: %w", obj, err) // critical update error occurred
	} else if updated {
		logger.Info(fmt.Sprintf("Successfully ran UPDATE for object %T", obj))
	}

	// Check if object is ready
	if h.IsReady(obj) {
		return Ready, nil
	}
	return NotReady, nil
}
