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

type numericHasher[T Numeric] struct{}

func (numericHasher[T]) Equal(a, b T) bool { return a == b }
func (numericHasher[T]) Hash(a T) uint64   { return uint64(a) }

/* Once this works (in Go 1.19 maybe) the hasher implementations can be
   used directly instead of going through the "constructor" functions
func f[T any](h Hasher[T]) {}
var _ = func() struct{} {
	f(NumericHasher[int]{})
	return struct{}{}
}()
*/

func NumericHasher[T Numeric]() Hasher[T] { return numericHasher[T]{} }

type stringHasher[T ~string] struct{}

func (stringHasher[T]) Equal(a, b T) bool { return a == b }
func (stringHasher[T]) Hash(a T) (res uint64) {
	// TODO: This is bad because the two halves of the output are
	//  formed independently by the even and odd characters...
	for _, c := range a {
		res = bits.RotateLeft64(res, 32) ^ uint64(c)
	}
	return res
}

func StringHasher[T ~string]() Hasher[T] { return stringHasher[T]{} }
