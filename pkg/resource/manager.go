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

func (s HandleState) Done() bool     { return s == Ready }
func (s HandleState) Checking() bool { return s == NotReady }

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
	logger := log.FromContext(ctx).WithValues("type", fmt.Sprintf("%T", obj))
	logger.Info("Handling state")

	// Fetch the object
	shouldCreate := false
	if err := h.FetchFunc(obj); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info("Object not found, marking for CREATE")
			shouldCreate = true // not found, mark
		} else {
			logger.Error(err, "Failed to FETCH object")
			return Fetch, fmt.Errorf("failed to fetch object %s: %w", obj.GetName(), err) // critical fetch error occurred
		}
	}

	// Create object if marked for creation
	if shouldCreate {
		if err := h.CreateFunc(obj); err != nil {
			logger.Error(err, "Failed to CREATE object")
			return Create, fmt.Errorf("failed to create object %s: %w", obj.GetName(), err) // critical create error occurred
		} else {
			logger.Info("Successfully ran CREATE operation for object")
		}
	}

	// Update object
	updated, err := h.UpdateFunc(obj)
	if err != nil {
		logger.Error(err, "Failed to UPDATE object")
		return Update, fmt.Errorf("failed to update object %s: %w", obj.GetName(), err) // critical update error occurred
	} else if updated {
		logger.Info("Successfully ran UPDATE operation for object")
	}

	// Check if object is ready
	if h.IsReady(obj) {
		logger.Info("Object is in READY state")
		return Ready, nil
	}
	logger.Info("Object is in NOT READY state")
	return NotReady, nil
}
