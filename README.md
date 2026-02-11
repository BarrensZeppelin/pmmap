# Persistent Mergeable Hash Map (pmmap)

[![Go Reference](https://pkg.go.dev/badge/github.com/BarrensZeppelin/pmmap.svg)](https://pkg.go.dev/github.com/BarrensZeppelin/pmmap)


This package provides a Go implementation of persistent (immutable) hash-based collections with efficient _merge_ and _equality_ operations.

The backing data structure is a [patricia trie](https://en.wikipedia.org/wiki/Radix_tree#PATRICIA) on key hashes.

## Usage

Go 1.24+ is required.

Install: `go get github.com/BarrensZeppelin/pmmap`

### Map

```go
hasher := pmmap.NumericHasher[int]{}
map0 := pmmap.New[string](hasher)
map1 := map0.Insert(42, "Hello World")
fmt.Println(map0.Lookup(42)) // "", false
fmt.Println(map1.Lookup(42)) // "Hello World", true
```

### Set

`Set[K]` is a thin wrapper around the map with a simplified API:

```go
hasher := pmmap.NumericHasher[int]{}
s := pmmap.NewSet(hasher).Insert(1).Insert(2).Insert(3)
fmt.Println(s.Contains(2)) // true
fmt.Println(s.Size())      // 3
```

### Hashers

To create a collection with key type `K` you must supply an implementation of `Hasher[K]`:

```go
type Hasher[K any] interface {
	Equal(a, b K) bool
	Hash(K) uint64
}
```

Built-in hashers:

| Hasher | Key constraint | Notes |
|---|---|---|
| `NumericHasher[T]{}` | `~int`, `~uint`, etc. | Identity hash |
| `StringHasher[T]{}` | `~string` | |
| `PointerHasher[T]{}` | `*T` | Hashes by memory address |
| `NewComparableHasher[T]()` | `comparable` | Uses `maphash.Comparable`; slower but high-quality hash |

## Merges

The hash maps support a merge operation that will join the key-value pairs in two maps into a single map.
This operation is made efficient by:

* Re-using substructures from the two merged maps when possible.

	For instance, merging a map with an empty map returns the first map directly without traversing any of its elements.

* Skipping processing of shared substructures.

	For instance, merging a map with itself always takes constant time.

Re-using shared substructures in merged maps drastically reduces memory usage and execution time of the merge operation.
Generally, merging a map with a version of the map with $r$ new insertions will take linear time in $r$ (indepedent of the size of the map, compared to looping over one of the maps).

When the merged maps both contain a mapping for a key, the mapped values are merged with a user-provided value merging binary operator $f$.
This operator must be commutative and idempotent:

$$
\forall a, b. f(a, b) = f(b, a) \textrm{ and } f(a, a) = a
$$

```go
map2 := map0.Insert(42, "Hello Wzrld").Insert(43, "Goodbye")
fmt.Println(
	map1.Merge(map2, func(a, b string) (string, bool) {
		// Return the lexicographically smallest string
		if a > b {
			a, b = b, a
		}
		return a, a == b
	})
) // [42 ↦ Hello World 43 ↦ Goodbye]
```

The returned flag should be `true` iff. the two values are equal.
This allows the implementation to re-use more substructures.

## Benchmarks

The project includes some performance benchmarks that compare the speed of insert and lookup operations to that of Go's builtin `map` implementation.
Inserts are roughly 8-10 times slower than the builtin map and lookups are roughly 6 times slower.
This map implementation is not a good general-purpose replacement for hash maps.
It is most useful when the merge operation is used to speed up merges of large maps and when large maps must be copied (which is essentially free due to immutability).
