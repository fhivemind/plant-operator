package utils

import (
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
)

// IsSubsetOf checks if expected is a subset of received with equal values.
// Empty or default values are ignored. Passed values must be pointers, otherwise it will error.
// Uses equality.Semantic for comparison, and runtime.UnstructuredConverter for map extraction.
// Only returns true when no error occured and expected is a subset of received.
func IsSubsetOf(expected, received interface{}) (bool, error) {
	expectedMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(expected)
	if err != nil {
		return false, err
	}
	receivedMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(received)
	if err != nil {
		return false, err
	}
	return equality.Semantic.DeepDerivative(expectedMap, receivedMap), nil
}
