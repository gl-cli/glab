package filter

// Filter the slice for all elements which test positive
func Filter[T any](s []T, test func(t T) bool) []T {
	result := []T{}
	for _, v := range s {
		if test(v) {
			result = append(result, v)
		}
	}
	return result
}
