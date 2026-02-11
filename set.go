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

// IntersectionSize returns the number of elements present in both sets.
//
// This operation is made fast by skipping processing of shared subtrees.
func (s Set[K]) IntersectionSize(other Set[K]) int {
	if s.m.root == nil || other.m.root == nil {
		return 0
	}
	return intersectionSize(s.m.root, other.m.root, s.m.hasher)
}

// intersectionSize returns the number of keys present in both trees.
func intersectionSize[K any](a, b node[K, struct{}], hasher Hasher[K]) int {
	// Shared subtree — all keys match.
	if a == b {
		return nodeSize(a)
	}

	// Check if either a or b is a leaf.
	lf, isLeaf := a.(*leaf[K, struct{}])
	other := b
	if !isLeaf {
		lf, isLeaf = b.(*leaf[K, struct{}])
		other = a
	}

	if isLeaf {
		count := 0
		for _, pr := range lf.values {
			if _, found := lookup(other, lf.key, pr.key, hasher); found {
				count++
			}
		}
		return count
	}

	// Both are branches.
	s, t := a.(*branch[K, struct{}]), b.(*branch[K, struct{}])
	if s.branchBit == t.branchBit && s.prefix == t.prefix {
		return intersectionSize(s.left, t.left, hasher) +
			intersectionSize(s.right, t.right, hasher)
	}

	if s.branchBit > t.branchBit {
		s, t = t, s
	}

	if s.branchBit < t.branchBit && s.match(t.prefix) {
		// s contains t.
		sub := s.right
		if zeroBit(t.prefix, s.branchBit) {
			sub = s.left
		}
		return intersectionSize(sub, t, hasher)
	}

	// Prefixes disagree — no intersection.
	return 0
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
