package xslices

// Truncate returns a slice of the first n elements of the input slice.
// If n is greater than the length of the input slice, the entire slice is returned.
func Truncate[T any](items []T, n int) []T {
	if len(items) <= n {
		return items
	}
	return items[:n]
}
