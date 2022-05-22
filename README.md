# Persistent Mergeable Hash Map (pmmap)

This package provides a Go implementation of a persistent key-value hash map with an efficient _merge_ operation.

The backing data structure is a [patricia trie](https://en.wikipedia.org/wiki/Radix_tree#PATRICIA).

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
