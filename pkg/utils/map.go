package utils

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
)

// MergeMapsSrcDst adds data from source to dest
func MergeMapsSrcDst(from, to map[string]string) {
	for key, value := range from {
		to[key] = value
	}
}

// UnsafeMapDiff just returns a diff between two objects. Must pass a reference.
// TODO: Resolve this for better usage
func UnsafeMapDiff(objA, objB interface{}) (diffValues, error) {
	a, err := runtime.DefaultUnstructuredConverter.ToUnstructured(objA)
	if err != nil {
		return nil, err
	}
	b, err := runtime.DefaultUnstructuredConverter.ToUnstructured(objB)
	if err != nil {
		return nil, err
	}

	var lMap = a
	var sMap = b
	if len(b) > len(a) {
		sMap = a
		lMap = b
	}

	var result []diffValue
	for aKey, aVal := range lMap {
		isMissingKey := true
		var diffVals []string
		for bKey, bVal := range sMap {
			if aKey == bKey {
				isMissingKey = false
				if !reflect.DeepEqual(aVal, bVal) {
					diffVals = append(diffVals, fmt.Sprintf("%v: %v => %v", aKey, aVal, bVal))
				}
				continue
			}
		}
		if isMissingKey {
			result = append(result, diffValue{
				Missing: true,
				Fields: []string{
					fmt.Sprintf("%v: %v", aKey, aVal),
				},
			})
		}
		if len(diffVals) > 0 {
			result = append(result, diffValue{
				Different: true,
				Fields:    diffVals,
			})
		}
	}
	return result, nil
}

type diffValue struct {
	Fields    []string
	Missing   bool
	Different bool
}

type diffValues []diffValue

func (d diffValues) Values(different, missing bool) (res []string) {
	for _, diffVal := range d {
		if diffVal.Missing && missing {
			res = append(res, diffVal.Fields...)
		}
		if diffVal.Different && different {
			res = append(res, diffVal.Fields...)
		}
	}
	return
}
