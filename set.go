package pmmap

import (
	"fmt"
	"iter"
)

// Set represents a persistent hash set backed by a [Tree].
//
// All mutating operations return a new set, leaving the original unchanged.
type Set[K any] struct{ m Tree[K, struct{}] }

// NewSet constructs a new persistent set with the specified hasher.
func NewSet[K any](hasher Hasher[K]) Set[K] {
	return Set[K]{New[struct{}](hasher)}
}

// Contains reports whether the set contains the given key.
func (s Set[K]) Contains(key K) bool {
	_, found := s.m.Lookup(key)
	return found
}

// Insert adds the given key to the set.
func (s Set[K]) Insert(key K) Set[K] {
	s.m = s.m.Insert(key, struct{}{})
	return s
}

// Remove removes the given key from the set.
func (s Set[K]) Remove(key K) Set[K] {
	s.m = s.m.Remove(key)
	return s
}

// Union returns the union of two sets.
//
// This operation is made fast by skipping processing of shared subtrees.
func (s Set[K]) Union(other Set[K]) Set[K] {
	s.m = s.m.Merge(other.m, func(a, _ struct{}) (struct{}, bool) {
		return a, true
	})
	return s
}

// Equal reports whether two sets contain the same keys.
//
// This operation is made fast by skipping processing of shared subtrees.
func (s Set[K]) Equal(other Set[K]) bool {
	return s.m.Equal(other.m, func(_, _ struct{}) bool { return true })
}

// All returns an iterator over all keys in the set.
func (s Set[K]) All() iter.Seq[K] {
	return s.m.Keys()
}

// Size returns the number of elements in the set.
func (s Set[K]) Size() int {
	return s.m.Size()
}

func (s Set[K]) String() string {
	buf := make([]string, 0, s.Size())
	for k := range s.All() {
		buf = append(buf, fmt.Sprintf("%v", k))
	}
	return fmt.Sprintf("set%s", buf)
}
