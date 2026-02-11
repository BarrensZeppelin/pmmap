# CLAUDE.md

## Build & Test

```bash
go test ./...              # run all tests
go test -run TestName      # run a specific test
go test -bench .           # run benchmarks
go vet ./...               # static analysis
```

No Makefile, linter config, or CI. No external dependencies.

## Architecture

Single-package Go library (`github.com/BarrensZeppelin/pmmap`) implementing a persistent (immutable) mergeable hash map backed by a patricia trie on bit-reversed 64-bit key hashes.

**Source files:**
- `tree.go` — `Tree[K,V]` (public API: Lookup, Insert, Remove, Merge, Equal, iterators) and internal trie implementation (`node` interface, `branch`, `leaf`)
- `hasher.go` — `Hasher[K]` interface and built-in hashers (`NumericHasher[T]`, `StringHasher[T]`)
- `lib.go` — bit-manipulation helpers (`zeroBit`, `branchingBit`)

**Key types:**
- `Tree[K,V]` — the persistent map; all mutating methods return a new tree
- `Hasher[K]` — interface with `Equal(a, b K) bool` and `Hash(K) uint64`
- `MergeFunc[V]` — `func(a, b V) (V, bool)`; must be commutative and idempotent

## Conventions

- Go 1.24+ required (range-over-func iterators)
- Hashers are exported struct types (e.g., `NumericHasher[int]{}`) not constructor functions
- `MergeFunc` must satisfy `f(a,b) = f(b,a)` and `f(a,a) = a`; its second return value signals equality
- Hash collisions are handled by bucket lists in leaves
