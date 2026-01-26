package util

import (
	"cmp"
	"maps"
	"slices"
)

// SortedKeys returns the keys of a map in sorted order.
func SortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	return slices.Sorted(maps.Keys(m))
}
