package utils

// MergeMapsSrcDst adds data from source to dest
func MergeMapsSrcDst(from, to map[string]string) {
	for key, value := range from {
		to[key] = value
	}
}
