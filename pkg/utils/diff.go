package utils

import (
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
)

type diff struct {
	equal bool
	err   error
}

// Error returns non-nil error iff error occurred during diff
func (d *diff) Error() error {
	return d.err
}

// Equal returns true iff Error is nil and difference is not present
func (d *diff) Equal() bool {
	return d.err == nil && d.equal
}

// NotEqual returns true iff Error is nil and difference is present
func (d *diff) NotEqual() bool {
	return d.err == nil && !d.equal
}

// Diff checks if expected is a subset of received with equal values.
// Empty or default values are ignored. Passed values must be pointers, otherwise it will error.
// Uses equality.Semantic for comparison, and runtime.UnstructuredConverter for map extraction.
func Diff(expected, received interface{}) *diff {
	expectedMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(expected)
	if err != nil {
		return &diff{
			equal: false,
			err:   err,
		}
	}
	receivedMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(received)
	if err != nil {
		return &diff{
			equal: false,
			err:   err,
		}
	}

	return &diff{
		equal: equality.Semantic.DeepDerivative(expectedMap, receivedMap),
		err:   nil,
	}
}
