package utils

import (
	"fmt"
	"strings"
)

func ObjectType(obj interface{}) string {
	return strings.Replace(fmt.Sprintf("%T", obj), "*", "", -1)
}
