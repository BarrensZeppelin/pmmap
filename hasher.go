package hashtree

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

