package pmmap

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestSetIntersectionSize(t *testing.T) {
	for _, hasher := range []Hasher[int]{intHasher, Hasher[int](badHasher[int]{}), mkMemHasher(5)} {
		t.Run(fmt.Sprintf("%T", hasher), func(t *testing.T) {
			empty := NewSet[int](hasher)

			t.Run("BothEmpty", func(t *testing.T) {
				if n := empty.IntersectionSize(empty); n != 0 {
					t.Errorf("expected 0, got %d", n)
				}
			})

			t.Run("OneEmpty", func(t *testing.T) {
				s := empty.Insert(1).Insert(2).Insert(3)
				if n := s.IntersectionSize(empty); n != 0 {
					t.Errorf("expected 0, got %d", n)
				}
				if n := empty.IntersectionSize(s); n != 0 {
					t.Errorf("expected 0, got %d", n)
				}
			})

			t.Run("SameSet", func(t *testing.T) {
				s := empty.Insert(1).Insert(2).Insert(3)
				if n := s.IntersectionSize(s); n != 3 {
					t.Errorf("expected 3, got %d", n)
				}
			})

			t.Run("Disjoint", func(t *testing.T) {
				a := empty.Insert(1).Insert(2).Insert(3)
				b := empty.Insert(4).Insert(5).Insert(6)
				if n := a.IntersectionSize(b); n != 0 {
					t.Errorf("expected 0, got %d", n)
				}
			})

			t.Run("PartialOverlap", func(t *testing.T) {
				a := empty.Insert(1).Insert(2).Insert(3).Insert(4)
				b := empty.Insert(3).Insert(4).Insert(5).Insert(6)
				if n := a.IntersectionSize(b); n != 2 {
					t.Errorf("expected 2, got %d", n)
				}
				if n := b.IntersectionSize(a); n != 2 {
					t.Errorf("expected 2, got %d", n)
				}
			})

			t.Run("Subset", func(t *testing.T) {
				a := empty.Insert(1).Insert(2).Insert(3).Insert(4).Insert(5)
				b := empty.Insert(2).Insert(4)
				if n := a.IntersectionSize(b); n != 2 {
					t.Errorf("expected 2, got %d", n)
				}
				if n := b.IntersectionSize(a); n != 2 {
					t.Errorf("expected 2, got %d", n)
				}
			})

			t.Run("SharedSubtrees", func(t *testing.T) {
				a := empty.Insert(1).Insert(2).Insert(3)
				b := a.Insert(4).Insert(5) // b shares structure with a
				// intersection should be {1,2,3}
				if n := a.IntersectionSize(b); n != 3 {
					t.Errorf("expected 3, got %d", n)
				}
			})
		})
	}
}

func TestSetIntersectionSizeMany(t *testing.T) {
	const N = 100
	for range 50 {
		hasher := mkMemHasher(N / 5)
		empty := NewSet[int](hasher)

		a, b := empty, empty
		expect := 0
		inA := make(map[int]bool)
		inB := make(map[int]bool)
		for i := range 2 * N {
			if rand.Intn(2) == 0 {
				a = a.Insert(i)
				inA[i] = true
			}
			if rand.Intn(2) == 0 {
				b = b.Insert(i)
				inB[i] = true
			}
			if inA[i] && inB[i] {
				expect++
			}
		}

		if n := a.IntersectionSize(b); n != expect {
			t.Errorf("expected %d, got %d", expect, n)
		}
		if n := b.IntersectionSize(a); n != expect {
			t.Errorf("expected %d (reversed), got %d", expect, n)
		}
	}
}
