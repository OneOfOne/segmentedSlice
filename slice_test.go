package list

import (
	"math/rand"
	"sort"
	"testing"
)

func TestSort(t *testing.T) {
	rand.Seed(0)
	const sliceLen = 100
	l := NewSortable(5, func(a, b interface{}) bool { return a.(int) < b.(int) })

	for i := 0; i < sliceLen; i++ {
		l.Append((sliceLen - 1) - i)
	}

	if l.Len() != 100 {
		t.Fatalf("expected length %d, got %d", sliceLen, l.Len())
	}
	sort.Sort(l)

	l.ForEach(func(i int, v interface{}) (breakNow bool) {
		if i != v.(int) {
			t.Errorf("expected %v, got %v", i, v)
			return true
		}
		return
	})
}

func BenchmarkAppendSegmentedSlice(b *testing.B) {
	l := New(99)
	for i := 0; i < b.N; i++ {
		l.Append(i)
	}

	if testing.Verbose() {
		b.Logf("len: %d, cap: %d, segments: %d", l.Len(), l.Cap(), l.Segments())
	}
}

func BenchmarkAppendNormalSlice(b *testing.B) {
	l := make([]interface{}, 0, 99)
	for i := 0; i < b.N; i++ {
		l = append(l, i)
	}

	if testing.Verbose() {
		b.Logf("len: %d, cap: %d", len(l), cap(l))
	}
}
