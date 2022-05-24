# Persistent Mergeable Hash Map (pmmap)

[![Go Reference](https://pkg.go.dev/badge/github.com/BarrensZeppelin/pmmap@v0.1.0.svg)](https://pkg.go.dev/github.com/BarrensZeppelin/pmmap@v0.1.0)

This package provides a Go implementation of a persistent key-value hash map with an efficient _merge_ operation.

The maps are immutable, so modifying operations (inserts and removals) return a copy of the map with the operation applied.

The backing data structure is a [patricia trie](https://en.wikipedia.org/wiki/Radix_tree#PATRICIA) on key hashes.

## Usage

The map uses generics to make the API ergonomic. Therefore Go 1.18+ is required.

Install: `go get github.com/BarrensZeppelin/pmmap`

```go
hasher := pmmap.NumericHasher[int]()
map0 := pmmap.New[string](hasher)
map1 := map0.Insert(42, "Hello World")
fmt.Println(map0.Lookup(42)) // "", false
fmt.Println(map1.Lookup(42)) // "Hello World", true
```

To create a map with key type `K` you must supply an implementation of `Hasher[K]`:

```go
type Hasher[K any] interface {
	Equal(a, b K) bool
	Hash(K) uint64
}
```

Default hasher implementations are included for numeric types and strings.

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
