package pmmap

import (
	"hash/maphash"
	"math/bits"
	"unsafe"
)

// The Hasher type provides a way to hash values as well as compare them for
// equality.
//
// The implementation guarantees that the hash function is called exactly once
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

type StringHasher[T ~string] struct{}

func (StringHasher[T]) Equal(a, b T) bool { return a == b }
func (StringHasher[T]) Hash(a T) (res uint64) {
	// TODO: This is bad because the two halves of the output are
	//  formed independently by the even and odd characters...
	for _, c := range a {
		res = bits.RotateLeft64(res, 32) ^ uint64(c)
	}
	return res
}

// PointerHasher hashes pointers by their memory address.
// Go's GC is non-moving, so addresses are stable for the lifetime of an object.
type PointerHasher[T any] struct{}

func (PointerHasher[T]) Equal(a, b *T) bool { return a == b }
func (PointerHasher[T]) Hash(p *T) uint64   { return uint64(uintptr(unsafe.Pointer(p))) }

// ComparableHasher hashes any comparable type using maphash.Comparable.
// Each instance carries its own seed; create instances with NewComparableHasher.
// This hasher is slower but the hash function is much better than the others.
type ComparableHasher[T comparable] struct{ seed maphash.Seed }

func NewComparableHasher[T comparable]() ComparableHasher[T] {
	return ComparableHasher[T]{seed: maphash.MakeSeed()}
}

func (h ComparableHasher[T]) Equal(a, b T) bool { return a == b }
func (h ComparableHasher[T]) Hash(a T) uint64   { return maphash.Comparable(h.seed, a) }
