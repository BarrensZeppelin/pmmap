package pmmap

import "math/bits"

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
