package segmentedSlice

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"sort"
	"testing"
)

func TestSegmentedSlice(t *testing.T) {
	rand.Seed(0)
	const sliceLen = 100
	l := NewSortable(5, func(a, b interface{}) bool { return a.(int) < b.(int) })

	for i := 0; i < sliceLen; i++ {
		l.Append((sliceLen - 1) - i)
	}

	if l.Len() != 100 {
		t.Fatalf("expected length %d, got %d", sliceLen, l.Len())
	}

	t.Logf("len: %d, cap: %d, segments: %d", l.Len(), l.Cap(), l.Segments())

	t.Run("Sort", func(t *testing.T) {
		sort.Sort(l)

		l.ForEach(func(i int, v interface{}) (breakNow bool) {
			if i != v.(int) {
				t.Errorf("expected %v, got %v", i, v)
				return true
			}
			return
		})
	})

	t.Run("Slice and Copy", func(t *testing.T) {
		for it := l.Slice(5, 10).Copy().Iter(); it.More(); {
			if idx, v := it.NextIndex(); v.(int) != idx+5 {
				t.Errorf("expected %v, got %v", idx+5, v)
			}
		}
	})
}

func TestJSON(t *testing.T) {
	testData := intJSONData(100)

	l := New(5)
	for i := 0; i < 100; i++ {
		l.Append(i)
	}
	j, _ := json.Marshal(l)

	t.Run("Marshal", func(t *testing.T) {
		if !bytes.Equal(j, testData) {
			t.Fatalf("expected:\n\t%s\ngot:\n\t:%s", testData, j)
		}
	})

	t.Run("Untyped", func(t *testing.T) {
		var ss SegmentedSlice

		if err := json.Unmarshal(j, &ss); err != nil {
			t.Fatal(err)
		}

		if ss.segLen != DefaultSegmentLen {
			t.Fatalf("expected %d segLen, got %d", DefaultSegmentLen, ss.segLen)
		}

		for it := l.Iter(); it.More(); {
			idx, v := it.NextIndex()
			// untyped numbers in json are automatically converted to float64
			if ss.Get(idx).(float64) != float64(v.(int)) {
				t.Fatalf("something is really wrong")
			}
		}
	})

	t.Run("Typed", func(t *testing.T) {
		var ss SegmentedSlice

		ss.SetUnmarshalType(0) // set type to untyped int

		if err := json.Unmarshal(j, &ss); err != nil {
			t.Fatal(err)
		}

		if ss.segLen != DefaultSegmentLen {
			t.Fatalf("expected %d segLen, got %d", DefaultSegmentLen, ss.segLen)
		}

		for it := l.Iter(); it.More(); {
			idx, v := it.NextIndex()
			if ss.Get(idx).(int) != v.(int) {
				t.Fatalf("something is really wrong")
			}
		}
	})
}

func BenchmarkAppendSegmentedSlice(b *testing.B) {
	l := New(99) // odd number to make sure we will have an extra segment at the end.
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

func intJSONData(ln int) []byte {
	s := make([]interface{}, ln)
	for i := range s {
		s[i] = i
	}
	j, _ := json.Marshal(s)
	return j
}

func printJSON(tb testing.TB, v interface{}) {
	j, _ := json.MarshalIndent(v, "", "  ")
	tb.Logf("%s", j)
}
