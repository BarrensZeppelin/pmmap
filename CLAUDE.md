# CLAUDE.md

## Build & Test

```bash
go test ./...              # run all tests
go test -run TestName      # run a specific test
go test -bench .           # run benchmarks
go vet ./...               # static analysis
```

## Architecture

Single-package Go library implementing a persistent (immutable) mergeable hash map backed by a patricia trie on bit-reversed 64-bit key hashes.

**Source files:**
- `tree.go` — `Tree[K,V]` (public API: Lookup, Insert, Remove, Merge, Equal, iterators) and internal trie implementation (`node` interface, `branch`, `leaf`, and node-level operations such as `merge`, `equal`, `intersectionSize`)
- `set.go` — `Set[K]`, a thin wrapper around `Tree[K, struct{}]`; it reads `Tree` internals (`root`, `hasher`) and calls node-level operations directly
- `hasher.go` — `Hasher[K]` interface and built-in hashers (`NumericHasher[T]`, `PointerHasher[T]`, `ComparableHasher[T]`)
- `lib.go` — bit-manipulation helpers (`zeroBit`, `branchingBit`)

**Key types:**
- `Tree[K,V]` — the persistent map; all mutating methods return a new tree
- `Set[K]` — the persistent set, backed by `Tree[K, struct{}]`
- `Hasher[K]` — interface with `Equal(a, b K) bool` and `Hash(K) uint64`
- `MergeFunc[V]` — `func(a, b V) (V, bool)`; must be commutative and idempotent
