package hashtree

import (
	"math/rand"
	"testing"
)

type MutableMap[K, V any] interface {
	Insert(K, V)
	Lookup(K) (V, bool)
}

type MapMap[K comparable, V any] map[K]V

func (m MapMap[K, V]) Insert(key K, value V) { m[key] = value }
func (m MapMap[K, V]) Lookup(key K) (V, bool) {
	v, ok := m[key]
	return v, ok
}

type MutableTree[K, V any] struct{ Tree[K, V] }

func (mt *MutableTree[K, V]) Insert(key K, value V) {
	mt.Tree = mt.Tree.Insert(key, value)
}

const benchmarkSize = 10000

var mutableMapImpls = []struct {
	name    string
	factory func() MutableMap[int, int]
}{
	{"map", func() MutableMap[int, int] { return make(MapMap[int, int]) }},
	{"tree", func() MutableMap[int, int] { return &MutableTree[int, int]{New[int](intHasher)} }},
}

func BenchmarkInserts(b *testing.B) {
	for _, mmap := range mutableMapImpls {
		b.Run(mmap.name, func(b *testing.B) {
			for bi := 0; bi < b.N; bi++ {
				mp := mmap.factory()
				for i := 0; i < benchmarkSize; i++ {
					mp.Insert(i, i)
				}
			}
		})
	}
}

var blackhole interface{}

func BenchmarkLookups(b *testing.B) {
	var seq [benchmarkSize]int
	for i := range seq {
		seq[i] = rand.Int()
	}

	for _, mmap := range mutableMapImpls {
		b.Run(mmap.name, func(b *testing.B) {
			var res int
			mp := mmap.factory()
			for i := 0; i < benchmarkSize/2; i++ {
				mp.Insert(seq[i], i)
			}
			b.ResetTimer()
			for bi := 0; bi < b.N; bi++ {
				for i := 0; i < benchmarkSize; i++ {
					res, _ = mp.Lookup(seq[i])
				}
			}
			blackhole = res
		})
	}
}

type Set[T any] interface {
	Contains(T) bool
}

type JoinableSet[T any, S Set[T]] interface {
	Set[T]
	Join(S)
}

type MapSet[K comparable] map[K]struct{}

func (m MapSet[K]) Contains(key K) bool {
	_, ok := m[key]
	return ok
}

func (m MapSet[K]) Join(o MapSet[K]) {
	for k, v := range o {
		m[k] = v
	}
}

var _ JoinableSet[int, MapSet[int]] = make(MapSet[int])

type TreeSet[T any] struct { tree Tree[T, struct{}] }

func (t *TreeSet[T]) Contains(key T) bool {
	_, ok := t.tree.Lookup(key)
	return ok
}

func (t *TreeSet[T]) Join(o *TreeSet[T]) {
	t.tree = t.tree.Merge(o.tree, func(a, b struct{}) (struct{}, bool) {
		return a, true
	})
}

var _ JoinableSet[int, *TreeSet[int]] = &TreeSet[int]{}
