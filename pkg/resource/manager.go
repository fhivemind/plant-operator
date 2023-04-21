package resource

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (h *Handler[T]) Handle(obj T) (HandleState, error) {
	shouldCreate := false

	// Fetch the object
	if err := h.FetchFunc(obj); err != nil {
		if client.IgnoreNotFound(err) == nil {
			shouldCreate = true // not found, mark
		} else {
			return Fetch, err // critical fetch error occurred
		}
	}

	// Create object if marked for creation
	if shouldCreate {
		if err := h.CreateFunc(obj); err != nil {
			return Create, err // critical create error occurred
		}
	}

	// Update object
	_, err := h.UpdateFunc(obj)
	if err != nil {
		return Update, err // critical update error occurred
	}

	// Check if object is ready
	if h.IsReady(obj) {
		return Ready, nil
	}
	return NotReady, nil
}
