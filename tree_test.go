package pmmap

import (
	"fmt"
	"math/rand"
	"testing"
)

var intHasher Hasher[int] = NumericHasher[int]{}
var uint32Hasher Hasher[uint32] = NumericHasher[uint32]{}

func testLookup[K any, V comparable](
	t *testing.T, tree Tree[K, V],
	key K, expectFound bool, expectVal V,
) {
	val, found := tree.Lookup(key)
	if found != expectFound {
		if found {
			t.Error("Expected miss for", key)
		} else {
			t.Error("Expected hit for", key)
		}
	}

	if val != expectVal {
		t.Errorf("Lookup(%v) = %v, expected: %v", key, val, expectVal)
	}
}

func mkTest[K any, V comparable](t *testing.T) (
	func(tree Tree[K, V], key K, val V),
	func(Tree[K, V], K),
) {
	return func(tree Tree[K, V], key K, expectVal V) {
			if val, found := tree.Lookup(key); found {
				if val != expectVal {
					t.Errorf("Lookup(%v) = %v, expected: %v", key, val, expectVal)
				}
			} else {
				t.Error("Expected hit for", key)
			}
		}, func(tree Tree[K, V], key K) {
			if _, found := tree.Lookup(key); found {
				t.Fatal("Expected miss for", key)
			}
		}
}

func TestEmpty(t *testing.T) {
	tree := New[int](intHasher)
	testLookup(t, tree, 0, false, 0)
}

func cmpEq[T comparable](a, b T) bool { return a == b }

type memHasher struct {
	mem   map[int]keyt
	limit int
}

func (m memHasher) Hash(x int) keyt {
	if v, ok := m.mem[x]; ok {
		return v
	}
	h := keyt(rand.Intn(m.limit))
	m.mem[x] = h
	return h
}
func (m memHasher) Equal(a, b int) bool {
	return a == b
}
func mkMemHasher(limit int) Hasher[int] {
	return memHasher{make(map[int]keyt), limit}
}

func TestSameKey(t *testing.T) {
	for _, hasher := range []Hasher[int]{intHasher, badHasher[int]{}} {
		hit, miss := mkTest[int, string](t)
		tree0 := New[string](hasher)
		tree1 := tree0.Insert(0, "v1")
		tree2 := tree1.Insert(0, "v2")

		miss(tree0, 0)
		hit(tree1, 0, "v1")
		hit(tree2, 0, "v2")

		if tree1.Equal(tree2, cmpEq[string]) {
			t.Error(tree1, "should not equal", tree2)
		}
	}
}

type badHasher[T comparable] struct{}

func (badHasher[T]) Hash(T) keyt       { return 0 }
func (badHasher[T]) Equal(a, b T) bool { return a == b }

func TestHashCollision(t *testing.T) {
	hit, miss := mkTest[int, string](t)
	tree0 := New[string](Hasher[int](badHasher[int]{}))
	tree1 := tree0.Insert(1, "v1")
	tree2 := tree1.Insert(2, "v2")

	miss(tree0, 1)
	miss(tree0, 2)

	hit(tree1, 1, "v1")
	miss(tree1, 2)

	hit(tree2, 1, "v1")
	hit(tree2, 2, "v2")
}

func TestDiffKey(t *testing.T) {
	hit, _ := mkTest[int, string](t)
	tree := New[string](intHasher).Insert(0, "v1").Insert(1, "v2")
	hit(tree, 0, "v1")
	hit(tree, 1, "v2")

	tree = tree.Insert(2, "v3")
	hit(tree, 0, "v1")
	hit(tree, 1, "v2")
	hit(tree, 2, "v3")
}

func TestManyInsert(t *testing.T) {
	iterations := 100
	N := 100

	for range iterations {
		tree := New[uint32](uint32Hasher)

		var keys []uint32
		for range N {
			k := rand.Uint32()
			keys = append(keys, k)
			tree = tree.Insert(k, k)
		}

		rand.Shuffle(N, func(i, j int) {
			keys[i], keys[j] = keys[j], keys[i]
		})

		for _, k := range keys {
			testLookup(t, tree, k, true, k)
		}
	}
}

func TestHistory(t *testing.T) {
	hit, miss := mkTest[int, int](t)
	N := 100

	for _, hasher := range []Hasher[int]{intHasher, mkMemHasher(N / 5)} {
		tree := New[int](hasher)
		history := []Tree[int, int]{tree}

		for i := range N {
			tree = tree.Insert(i, i)
			history = append(history, tree)
		}

		for vidx, tree := range history {
			for i := range N {
				if vidx <= i {
					miss(tree, i)
				} else {
					hit(tree, i, i)
				}
			}
		}
	}
}

func max(x, y int) (int, bool) {
	if x == y {
		return x, true
	}
	if x > y {
		return x, false
	} else {
		return y, false
	}
}

func TestSimpleMerge(t *testing.T) {
	hit, _ := mkTest[int, int](t)
	for _, hasher := range []Hasher[int]{intHasher, Hasher[int](badHasher[int]{}), mkMemHasher(2)} {
		a := New[int](hasher).Insert(0, 1).Insert(1, 1)
		b := New[int](hasher).Insert(1, 2).Insert(2, 2)

		check := func(tree Tree[int, int]) {
			hit(tree, 0, 1)
			hit(tree, 1, 2)
			hit(tree, 2, 2)

			if sz := tree.Size(); sz != 3 {
				t.Error("Wrong size:", sz)
			}
		}

		check(a.Merge(b, max))
		check(b.Merge(a, max))
	}
}

func TestMergeWithEmpty(t *testing.T) {
	a := New[int](intHasher).Insert(0, 0)
	a.Merge(New[int](intHasher), max)
}

func TestPointerEqualityAfterMerge(t *testing.T) {
	a, b := New[int](intHasher), New[int](intHasher)
	for i := range 4 {
		a = a.Insert(i, i)
		if i < 3 {
			b = b.Insert(i, i)
		}
	}

	c := a.Merge(b, func(x, y int) (int, bool) {
		return x, x == y
	})

	if !c.Equal(a, cmpEq[int]) {
		t.Fatalf("Equality or Merge is buggy. %v should be equal to %v", c, a)
	}

	if c.root != a.root {
		// Since `a` is a superset of `b`, we should be able to retain the
		// identity of the root.
		t.Errorf("Expected %p to be %p", c.root, a.root)
		t.Log(c.root.(*branch[int, int]).left)
		t.Log(a.root.(*branch[int, int]).left)
	}
}

func TestManyMerge(t *testing.T) {
	hit, _ := mkTest[int, int](t)
	iterations := 100
	N := 100

	for range iterations {
		for _, hasher := range []Hasher[int]{intHasher, mkMemHasher(N / 5)} {
			a, b := New[int](hasher), New[int](hasher)

			mp := make([]int, 2*N)
			for i := 0; i < 2*N; i++ {
				v1, v2 := rand.Int(), rand.Int()
				if i < N {
					mp[i], _ = max(v1, v2)
					a = a.Insert(i, v1)
					b = b.Insert(i, v2)
				} else if i < 3*N/2 {
					mp[i] = v1
					a = a.Insert(i, v1)
				} else {
					mp[i] = v2
					b = b.Insert(i, v2)
				}
			}

			merged := a.Merge(b, max)
			for k, v := range mp {
				hit(merged, k, v)
			}

			reconstructed := New[int](hasher)
			for k, v := range mp {
				reconstructed = reconstructed.Insert(k, v)
			}

			if !reconstructed.Equal(merged, cmpEq[int]) {
				t.Fatal("Expected", reconstructed, "to equal", merged)
			}
		}
	}
}

func TestRemove(t *testing.T) {
	hit, miss := mkTest[uint32, uint32](t)
	iterations := 100
	N := 100
	N_remove := 20

	for range iterations {
		tree := New[uint32](uint32Hasher)

		var keys []uint32
		for range N {
			k := rand.Uint32()
			keys = append(keys, k)
			tree = tree.Insert(k, k)
		}

		rand.Shuffle(N, func(i, j int) {
			keys[i], keys[j] = keys[j], keys[i]
		})

		removed := keys[:N_remove]
		for _, k := range removed {
			tree = tree.Remove(k)
		}

		if sz := tree.Size(); sz != N-N_remove {
			t.Error("Expected sz to be", N-N_remove, "was", sz)
		}

		for _, k := range removed {
			miss(tree, k)
		}

		for _, k := range keys[N_remove:] {
			hit(tree, k, k)
		}
	}
}

func TestIter(t *testing.T) {
	const N = 50
	tree := New[int](intHasher)
	for i := range N {
		tree = tree.Insert(i, i*10)
	}

	t.Run("All", func(t *testing.T) {
		count := 0
		for k, v := range tree.All() {
			got, found := tree.Lookup(k)
			if !found {
				t.Fatalf("All yielded key %v not found in tree", k)
			}
			if got != v {
				t.Fatalf("All yielded (%v, %v) but Lookup returned %v", k, v, got)
			}
			count++
		}
		if count != N {
			t.Fatalf("All yielded %d pairs, expected %d", count, N)
		}
	})

	t.Run("Keys", func(t *testing.T) {
		seen := map[int]bool{}
		for k := range tree.Keys() {
			seen[k] = true
		}
		if len(seen) != N {
			t.Fatalf("Keys yielded %d keys, expected %d", len(seen), N)
		}
		for i := range N {
			if !seen[i] {
				t.Fatalf("Keys missing key %d", i)
			}
		}
	})

	t.Run("Values", func(t *testing.T) {
		var vals []int
		for v := range tree.Values() {
			vals = append(vals, v)
		}
		if len(vals) != N {
			t.Fatalf("Values yielded %d values, expected %d", len(vals), N)
		}
	})

	t.Run("EarlyTermination", func(t *testing.T) {
		count := 0
		for range tree.All() {
			count++
			if count == 3 {
				break
			}
		}
		if count != 3 {
			t.Fatalf("expected 3 iterations, got %d", count)
		}

		count = 0
		for range tree.Keys() {
			count++
			if count == 5 {
				break
			}
		}
		if count != 5 {
			t.Fatalf("expected 5 iterations, got %d", count)
		}

		count = 0
		for range tree.Values() {
			count++
			if count == 7 {
				break
			}
		}
		if count != 7 {
			t.Fatalf("expected 7 iterations, got %d", count)
		}
	})

	t.Run("Empty", func(t *testing.T) {
		empty := New[int](intHasher)
		for range empty.All() {
			t.Fatal("All should not yield on empty tree")
		}
		for range empty.Keys() {
			t.Fatal("Keys should not yield on empty tree")
		}
		for range empty.Values() {
			t.Fatal("Values should not yield on empty tree")
		}
	})
}

func Example() {
	hasher := NumericHasher[int]{}
	tree0 := New[int](hasher)
	tree1 := tree0.Insert(5, 6)
	fmt.Println(tree0)
	fmt.Println(tree1)
	tree2 := tree0.Insert(5, 10)
	fmt.Println(tree1.Equal(tree2, hasher.Equal))
	fmt.Println(tree1.Merge(tree2, func(a, b int) (int, bool) {
		// Return the max of a and b
		if a < b {
			a, b = b, a
		}
		return a, a == b
	}))

	// Output:
	// tree[]
	// tree[5 ↦ 6]
	// false
	// tree[5 ↦ 10]
}
