# Shared libraries

## Generic Executor
Executor simplifies synchronization logic for Kubernetes API resources.
It exposes simple methods to control, fine-tune, and handle resource lifecycle.
Used for implementation of child resources managed by Plant Operator.
A bit more work could be invested to fine-tune and "prettify" the interfaces for more standardized usage.

Refer to `pkg/resource/executor.go` for info.

## Object comparison

### Generic derivative diff-er
This approach was used to monitor managed object changes to predefined values.
Complexities aside, it performs well, but needs to be further explored and tested to verify behaviour.


```golang
// expected defined somewhere
expectedSpecsMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&expected.Spec)
receivedSpecsMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&received.Spec)

// check if fields that expected CONTAINS are semantically equal to ones received has
if !equality.Semantic.DeepDerivative(expectedSpecsMap, receivedSpecsMap) {
    // object was updated
}
```
Refer to `pkg/utils/diff.go` for info.

### Alternative idea: Selective object comparator
For non-transformative and selective approach to comparison, objects that contain fields with non-zero default values,
or for granular comparison.
Think: _selective, exact, close, or not really_.
Example: Structure that contains a lot of fields, but we are interested in specific ones to avoid dynamic injection checks.

Refer to `pkg/utils/map.go` for PoC rundown.

Extends https://github.com/cisco-open/k8s-objectmatcher

- Let _expected_ and _actual_ be two objects which are marshall-able or reflect-able
- Recurse through the objects to create field map with names and types
- Compare to Options by depth and type, prioritize Exclusion by default, but offer options for filtering
- Expose following options for comparison:
    - `WithFields(...string)` => e.g. `WithFields(".Spec.[]Ports.*", ".Spec.Type")`
    - `WithoutFields(...string)` => e.g. `WithoutFields(".Spec.[]Ports.NodePort", ".Spec.IPFamilies")`
- Invoked as `Compare(expected, actual, ...Options)` and implements:
    - `Equal() bool` - returns if they are equal
    - `Error() error` - helps catch runtime errors, but pririotized through: base types, transforms, option misses and paradox cases.
