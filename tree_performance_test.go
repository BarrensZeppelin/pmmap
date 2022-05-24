package pmmap

import (
	"fmt"
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
	sequential := make([]int, benchmarkSize)
	for i := range sequential {
		sequential[i] = i
	}

	random := make([]int, benchmarkSize)
	for i := range random {
		random[i] = rand.Int()
	}

	keyDistributions := [...]struct {
		name string
		keys []int
	}{
		{"sequential", sequential},
		{"random", random},
	}

	for _, mmap := range mutableMapImpls {
		for _, dist := range keyDistributions {
			b.Run(fmt.Sprintf("%s-%s", mmap.name, dist.name), func(b *testing.B) {
				for bi := 0; bi < b.N; bi++ {
					mp := mmap.factory()
					for _, i := range dist.keys {
						mp.Insert(i, i)
					}
				}
			})
		}
	}
}

var blackhole interface{}

func BenchmarkLookups(b *testing.B) {
	var seq [benchmarkSize]int
	for i := range seq {
		seq[i] = rand.Int()
	}

	for _, mmap := range mutableMapImpls {
		for _, missProbability := range [...]int{50, 10, 1} {
			b.Run(fmt.Sprintf("%s-miss-%d", mmap.name, missProbability), func(b *testing.B) {
				var res int
				mp := mmap.factory()
				for i := 0; i < benchmarkSize*(100-missProbability)/100; i++ {
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
}

type Sets interface {
	Copy(src, dest int)
}

type MapSets struct{ maps []map[int]struct{} }

func (m MapSets) Copy(src, dest int) {
	a, b := m.maps[src], m.maps[dest]
	for k, v := range a {
		b[k] = v
	}
}

type TreeSets struct{ trees []Tree[int, struct{}] }

func (t TreeSets) Copy(src, dest int) {
	t.trees[dest] = t.trees[dest].Merge(t.trees[src], func(a, b struct{}) (struct{}, bool) {
		return a, true
	})
}

var setsImpls = []struct {
	name    string
	factory func(N int) Sets
}{
	{"map", func(N int) Sets {
		maps := make([]map[int]struct{}, N)
		for i := range maps {
			maps[i] = map[int]struct{}{i: {}}
		}
		return MapSets{maps}
	}},
	{"tree", func(N int) Sets {
		trees := make([]Tree[int, struct{}], N)
		for i := range trees {
			trees[i] = New[struct{}](intHasher).Insert(i, struct{}{})
		}
		return TreeSets{trees}
	}},
}

const dagSize = 1000

func BenchmarkDAGReachability(b *testing.B) {
	rnd := rand.New(rand.NewSource(0))

	edges := [dagSize][]int{}
	for i := 0; i < dagSize*20; i++ {
		a := rnd.Intn(dagSize - 1)
		b := a + 1 + rnd.Intn(dagSize-a-1)
		edges[b] = append(edges[b], a)
	}

	for _, simpl := range setsImpls {
		b.Run(simpl.name, func(b *testing.B) {
			for bi := 0; bi < b.N; bi++ {
				sets := simpl.factory(dagSize)
				for i := 0; i < dagSize; i++ {
					for _, j := range edges[i] {
						sets.Copy(j, i)
					}
				}
			}
		})
	}
}
