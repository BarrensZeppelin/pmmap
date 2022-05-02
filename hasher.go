package hashtree

type Hasher[K any] interface {
	Equal(a, b K) bool
	Hash(K) uint32
}

func hashUint64(x uint64) uint32 {
	return uint32(x ^ (x >> 32))
}

type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type NumericHasher[T Numeric] struct{}

func (NumericHasher[T]) Equal(a, b T) bool { return a == b }
func (NumericHasher[T]) Hash(a T) uint32   { return hashUint64(uint64(a)) }
