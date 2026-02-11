// Package pmmap provides a Go implementation of a persistent key-value hash
// map with an efficient merge operation.
//
// The maps are immutable, so modifying operations (inserts and removals)
// return a copy of the map with the operation applied.
//
// The backing data structure is a [patricia trie] on key hashes.
//
// [patricia trie]: https://en.wikipedia.org/wiki/Radix_tree#PATRICIA
package pmmap

import (
	"fmt"
	"iter"
	"math/bits"
)

// Construct a new persistent key-value map with the specified hasher.
func New[V, K any](hasher Hasher[K]) Tree[K, V] {
	// Order of K and V is swapped because K can be inferred from the argument.
	return Tree[K, V]{hasher, nil}
}

// Tree represents a persistent hash map.
//
// Hash collisions are resolved by putting key-value pairs into buckets that
// are scanned at lookups.
//
// Hash values of keys must not change while they are stored in the map.
type Tree[K, V any] struct {
	hasher Hasher[K]
	root   node[K, V]
}

// hash computes the big-endian 64-bit hash key.
func (tree Tree[K, V]) hash(key K) keyt {
	// The paper claims that big-endian patricia trees work better than
	// little-endian trees in practice. Instead of modifying the functions
	// operating on the tree, we can get the benefits of a big-endian tree by
	// reversing the bit representation of hashes up-front.
	return bits.Reverse64(tree.hasher.Hash(key))
}

// Lookup returns the value mapped to the provided key in the map.
// The semantics are equivalent to those of 2-valued lookup in regular Go maps.
func (tree Tree[K, V]) Lookup(key K) (ret V, found bool) {
	node := tree.root
	if node == nil {
		return
	}

	for hash := tree.hash(key); ; {
		switch n := node.(type) {
		case *leaf[K, V]:
			if n.key == hash {
				for _, pr := range n.values {
					if tree.hasher.Equal(key, pr.key) {
						return pr.value, true
					}
				}
			}

			return

		case *branch[K, V]:
			if !n.match(hash) {
				return
			} else if zeroBit(hash, n.branchBit) {
				node = n.left
			} else {
				node = n.right
			}
		default:
			panic("Impossible: unknown tree root type.")
		}
	}
}

// Insert the given key-value pair into the map.
// Replaces previous value with the same key if it exists.
func (tree Tree[K, V]) Insert(key K, value V) Tree[K, V] {
	return tree.InsertOrMerge(key, value, nil)
}

// Inserts the given key-value pair into the map. If a previous mapping
// (prevValue) exists for the key, the inserted value will be `f(value, prevValue)`.
func (tree Tree[K, V]) InsertOrMerge(key K, value V, f MergeFunc[V]) Tree[K, V] {
	tree.root, _ = insert(tree.root, tree.hash(key), key, value, tree.hasher, f)
	return tree
}

// Remove a mapping for the given key if it exists.
func (tree Tree[K, V]) Remove(key K) Tree[K, V] {
	tree.root = remove(tree.root, tree.hash(key), key, tree.hasher)
	return tree
}

// All returns an iterator over all key-value pairs in the map.
func (tree Tree[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if tree.root != nil {
			tree.root.iter(yield)
		}
	}
}

// Keys returns an iterator over all keys in the map.
func (tree Tree[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for k, _ := range tree.All() {
			if !yield(k) {
				return
			}
		}
	}
}

// Values returns an iterator over all values in the map.
func (tree Tree[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range tree.All() {
			if !yield(v) {
				return
			}
		}
	}
}

// Call the given function once for each key-value pair in the map.
func (tree Tree[K, V]) ForEach(f func(key K, value V)) {
	for k, v := range tree.All() {
		f(k, v)
	}
}

// Merges two maps. If both maps contain a value for a key, the resulting map
// will map the key to the result of `f` on the two values.
//
// See the documentation for MergeFunc for conditions that `f` must satisfy.
// No guarantees are made about the order of arguments provided to `f`.
//
// This operation is made fast by skipping processing of shared subtrees.
// Merging a tree with itself after r updates takes linear time in r.
func (tree Tree[K, V]) Merge(other Tree[K, V], f MergeFunc[V]) Tree[K, V] {
	tree.root, _ = merge(tree.root, other.root, tree.hasher, f)
	return tree
}

// Equal checks whether two maps are equal. Values are compared with the provided
// function. This operation also skips processing of shared subtrees.
func (tree Tree[K, V]) Equal(other Tree[K, V], f func(V, V) bool) bool {
	return equal(tree.root, other.root, tree.hasher, f)
}

// Size returns the number of key-value pairs in the map.
func (tree Tree[K, V]) Size() int {
	return nodeSize(tree.root)
}

func (tree Tree[K, V]) String() string {
	buf := []string{}

	tree.ForEach(func(k K, v V) {
		buf = append(buf, fmt.Sprintf("%v ↦ %v", k, v))
	})

	return fmt.Sprintf("tree%s", buf)
}

// MergeFunc describes a binary operator, f, that merges two values.
// The operator must be commutative and idempotent. I.e.:
//
//	f(a, b) = f(b, a)
//	f(a, a) = a
//
// The second return value informs the caller whether a == b.
// This flag allows some optimizations in the implementation.
type MergeFunc[V any] func(a, b V) (V, bool)

// End of public interface

// The patricia tree implementation is based on:
// https://web.archive.org/web/20220515235749/http://ittc.ku.edu/~andygill/papers/IntMap98.pdf

type (
	// node is an interface defined over nodes in the Patricia tree.
	node[K, V any] interface {
		iter(yield func(K, V) bool) bool
	}

	// keyt is an alias over the key type of a Patricia tree.
	keyt = uint64

	// branch encodes a branching node in the Patricia tree.
	branch[K, V any] struct {
		prefix keyt // Common prefix of all keys in the left and right subtrees
		// A number with exactly one positive bit. The position of the bit
		// determines where the prefixes of the left and right subtrees diverge.
		branchBit   keyt
		left, right node[K, V]
		size        int
	}
	// pair encodes a key-value pair in leaves.
	pair[K, V any] struct {
		key   K
		value V
	}
	// leaf encodes a terminal node in the Patricia tree.
	leaf[K, V any] struct {
		// The (shared) hash value of all keys in the leaf.
		key keyt
		// List of values to handle hash collisions.
		// TODO: Since collisions should be rare it might be worth
		// it to have a fast implementation when no collisions occur.
		values []pair[K, V]
		size   int
	}
)

// iter yields all key-value pairs in the subtree, returning false if iteration was stopped early.
func (b *branch[K, V]) iter(yield func(K, V) bool) bool {
	return b.left.iter(yield) && b.right.iter(yield)
}

// match returns whether the key matches the prefix up until the branching bit.
// Intuitively: does the key belong in the branch's subtree?
func (b *branch[K, V]) match(key keyt) bool {
	return (key & (b.branchBit - 1)) == b.prefix
}

// nodeSize returns the number of key-value pairs stored in the subtree rooted at n.
func nodeSize[K, V any](n node[K, V]) int {
	switch n := n.(type) {
	case *leaf[K, V]:
		return n.size
	case *branch[K, V]:
		return n.size
	default:
		return 0
	}
}

// copy constructs a new leaf that inherits the values of this leaf.
func (l *leaf[K, V]) copy() *leaf[K, V] {
	return &leaf[K, V]{
		l.key,
		append([]pair[K, V](nil), l.values...),
		l.size,
	}
}

// iter yields all key-value pairs in the leaf, returning false if iteration was stopped early.
func (l *leaf[K, V]) iter(yield func(K, V) bool) bool {
	for _, pr := range l.values {
		if !yield(pr.key, pr.value) {
			return false
		}
	}
	return true
}

// Smart branch constructor
func br[K, V any](prefix, branchBit keyt, left, right node[K, V]) node[K, V] {
	if left == nil {
		return right
	} else if right == nil {
		return left
	}

	return &branch[K, V]{prefix, branchBit, left, right, nodeSize(left) + nodeSize(right)}
}

// join merges two trees t0 and t1 which have prefixes p0 and p1 respectively.
// The prefixes must not be equal!
func join[K, V any](p0, p1 keyt, t0, t1 node[K, V]) node[K, V] {
	bbit := branchingBit(p0, p1)
	prefix := p0 & (bbit - 1)
	sz := nodeSize(t0) + nodeSize(t1)
	if zeroBit(p0, bbit) {
		return &branch[K, V]{prefix, bbit, t0, t1, sz}
	} else {
		return &branch[K, V]{prefix, bbit, t1, t0, sz}
	}
}

// If `f` is nil the old value is always replaced with the argument value, otherwise
// the old value is replaced with `f(value, prevValue)`.
// If the returned flag is false, the returned node is (reference-)equal to the input node.
func insert[K, V any](tree node[K, V], hash keyt, key K, value V, hasher Hasher[K], f MergeFunc[V]) (node[K, V], bool) {
	if tree == nil {
		return &leaf[K, V]{key: hash, values: []pair[K, V]{{key, value}}, size: 1}, true
	}

	var prefix keyt
	switch tree := tree.(type) {
	case *leaf[K, V]:
		if tree.key == hash {
			for i, pr := range tree.values {
				// If key matches previous key, replace value
				if hasher.Equal(key, pr.key) {
					newValue := value
					if f != nil {
						var equal bool
						newValue, equal = f(value, pr.value)

						if equal {
							return tree, false
						}
					}

					lf := tree.copy()
					lf.values[i].value = newValue
					return lf, true
				}
			}

			// Hash collision - append to list of values in leaf
			lf := tree.copy()
			lf.values = append(lf.values, pair[K, V]{key, value})
			lf.size++
			return lf, true
		}

		prefix = tree.key

	case *branch[K, V]:
		if tree.match(hash) {
			l, r := tree.left, tree.right
			var changed bool
			if zeroBit(hash, tree.branchBit) {
				l, changed = insert(l, hash, key, value, hasher, f)
			} else {
				r, changed = insert(r, hash, key, value, hasher, f)
			}
			if !changed {
				return tree, false
			}
			return &branch[K, V]{tree.prefix, tree.branchBit, l, r, nodeSize(l) + nodeSize(r)}, true
		}

		prefix = tree.prefix

	default:
		panic("Impossible: unknown tree root type.")
	}

	newLeaf, _ := insert(nil, hash, key, value, nil, nil)
	return join(hash, prefix, newLeaf, tree), true
}

// remove returns a tree with the key-value pair matching the provided key if it exists.
// If such a pair does not exist the input tree is returned.
func remove[K, V any](tree node[K, V], hash keyt, key K, hasher Hasher[K]) node[K, V] {
	if tree == nil {
		return tree
	}

	switch tree := tree.(type) {
	case *leaf[K, V]:
		if tree.key == hash {
			for i, pr := range tree.values {
				if hasher.Equal(key, pr.key) {
					if len(tree.values) == 1 { // Common case
						return nil
					}

					return &leaf[K, V]{
						tree.key,
						// Remove the i'th entry
						append(tree.values[:i:i], tree.values[i+1:]...),
						tree.size - 1,
					}
				}
			}
		}
	case *branch[K, V]:
		if tree.match(hash) {
			left, right := tree.left, tree.right
			if zeroBit(hash, tree.branchBit) {
				left = remove(left, hash, key, hasher)
				if left == tree.left {
					return tree
				}
			} else {
				right = remove(right, hash, key, hasher)
				if right == tree.right {
					return tree
				}
			}

			return br(tree.prefix, tree.branchBit, left, right)
		}
	default:
		panic("Impossible: unknown tree root type.")
	}

	return tree
}

// merge two nodes. If the returned flag is true, a and b represent equal trees.
func merge[K, V any](a, b node[K, V], hasher Hasher[K], f MergeFunc[V]) (node[K, V], bool) {
	// Cheap pointer-equality
	if a == b {
		return a, true
	} else if a == nil {
		return b, false
	} else if b == nil {
		return a, false
	}

	// Check if either a or b is a leaf
	lf, isLeaf := a.(*leaf[K, V])
	other := b
	if !isLeaf {
		lf, isLeaf = b.(*leaf[K, V])
		other = a
	}

	if isLeaf {
		originalOther := other
		for _, pr := range lf.values {
			other, _ = insert(other, lf.key, pr.key, pr.value, hasher, f)
		}

		if oLf, oIsLeaf := other.(*leaf[K, V]); oIsLeaf &&
			other == originalOther &&
			len(lf.values) == len(oLf.values) {
			// Since the other tree is also a leaf, and it did not change as a
			// result of inserting our values, and we did not start out with a
			// fewer number of key-value pairs than the other leaf, the two
			// leaves were (and are still) equal.
			return a, true
		}

		return other, false
	}

	// Both a and b are branches
	s, t := a.(*branch[K, V]), b.(*branch[K, V])
	if s.branchBit == t.branchBit && s.prefix == t.prefix {
		l, leq := merge(s.left, t.left, hasher, f)
		r, req := merge(s.right, t.right, hasher, f)
		if leq && req {
			return s, true
		} else if (leq || l == s.left) && (req || r == s.right) {
			return s, false
		} else if (leq || l == t.left) && (req || r == t.right) {
			return t, false
		}

		return &branch[K, V]{s.prefix, s.branchBit, l, r, nodeSize(l) + nodeSize(r)}, false
	}

	if s.branchBit > t.branchBit {
		s, t = t, s
	}

	if s.branchBit < t.branchBit && s.match(t.prefix) {
		// s contains t
		l, r := s.left, s.right
		if zeroBit(t.prefix, s.branchBit) {
			l, _ = merge(l, node[K, V](t), hasher, f)
			if l == s.left {
				return s, false
			}
		} else {
			r, _ = merge(r, node[K, V](t), hasher, f)
			if r == s.right {
				return s, false
			}
		}
		return &branch[K, V]{s.prefix, s.branchBit, l, r, nodeSize(l) + nodeSize(r)}, false
	} else {
		// prefixes disagree
		return join(s.prefix, t.prefix, node[K, V](s), node[K, V](t)), false
	}
	// NOTE: The implementation of this function is complex because it is
	// performance critical, and since the performance does not rely only on
	// the implementation within this function. Using shared subtrees speeds
	// up future merge/equal operations on the result, which is important.
}

func equal[K, V any](a, b node[K, V], hasher Hasher[K], f func(V, V) bool) bool {
	if a == b {
		return true
	} else if a == nil || b == nil {
		return false
	}

	switch a := a.(type) {
	case *leaf[K, V]:
		b, ok := b.(*leaf[K, V])
		if !ok || len(a.values) != len(b.values) {
			return false
		}

	FOUND:
		for _, apr := range a.values {
			for _, bpr := range b.values {
				if hasher.Equal(apr.key, bpr.key) {
					if !f(apr.value, bpr.value) {
						return false
					}

					continue FOUND
				}
			}

			// a contained a key that b did not
			return false
		}

		return true

	case *branch[K, V]:
		b, ok := b.(*branch[K, V])
		if !ok {
			return false
		}

		return a.prefix == b.prefix && a.branchBit == b.branchBit &&
			equal(a.left, b.left, hasher, f) && equal(a.right, b.right, hasher, f)

	default:
		panic("Impossible: unknown tree root type.")
	}
}
