package pmmap

import (
	"hash/maphash"
	"unsafe"
)

// The Hasher type provides a way to hash values as well as compare them for
// equality.
//
// The implementation guarantees that the hash function is called at most once
// for lookup, insertion and removal.
type Hasher[K any] interface {
	Equal(a, b K) bool
	Hash(K) uint64
}

type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type NumericHasher[T Numeric] struct{}

func (NumericHasher[T]) Equal(a, b T) bool { return a == b }
func (NumericHasher[T]) Hash(a T) uint64   { return uint64(a) }

// PointerHasher hashes pointers by their memory address.
// Go's GC is non-moving, so addresses are stable for the lifetime of an object.
type PointerHasher[T any] struct{}

func (PointerHasher[T]) Equal(a, b *T) bool { return a == b }
func (PointerHasher[T]) Hash(p *T) uint64   { return uint64(uintptr(unsafe.Pointer(p))) }

// Shared by all ComparableHasher instances so that maps built with separate
// instances can be merged and compared.
var comparableSeed = maphash.MakeSeed()

// ComparableHasher hashes any comparable type using maphash.Comparable.
// Hashes are stable within a process but differ between runs.
// This hasher is slower but the hash function is much better than the others.
type ComparableHasher[T comparable] struct{}

func (ComparableHasher[T]) Equal(a, b T) bool { return a == b }
func (ComparableHasher[T]) Hash(a T) uint64   { return maphash.Comparable(comparableSeed, a) }
