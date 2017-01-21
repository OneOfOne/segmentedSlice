package segmentedSlice

import (
	"bytes"
	"encoding/json"
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

func TestJSON(t *testing.T) {
	testData := intJsonData(100)

	l := New(5)
	for i := 0; i < 100; i++ {
		l.Append(i)
	}
	j, _ := json.Marshal(l)

	if bytes.Compare(j, testData) != 0 {
		t.Fatalf("expected:\n\t%s\ngot:\n\t:%s", testData, j)
	}

	var nl SegmentedSlice

	if err := json.Unmarshal(j, &nl); err != nil {
		t.Fatal(err)
	}

	if nl.segLen != DefaultSegmentLen {
		t.Fatalf("expected %d segLen, got %d", DefaultSegmentLen, nl.segLen)
	}

	for it := l.IteratorAt(0); it.More(); {
		idx, v := it.NextIndex()
		if nl.Get(idx).(float64) != float64(v.(int)) {
			t.Fatalf("something is really wrong")
		}
	}

	var nl2 SegmentedSlice

	nl2.SetUnmarshalType(0) // set type to untyped int

	if err := json.Unmarshal(j, &nl2); err != nil {
		t.Fatal(err)
	}

	if nl2.segLen != DefaultSegmentLen {
		t.Fatalf("expected %d segLen, got %d", DefaultSegmentLen, nl2.segLen)
	}

	for it := l.IteratorAt(0); it.More(); {
		idx, v := it.NextIndex()
		if nl2.Get(idx).(int) != v.(int) {
			t.Fatalf("something is really wrong")
		}
	}
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

func intJsonData(ln int) []byte {
	s := make([]interface{}, ln)
	for i := range s {
		s[i] = i
	}
	j, _ := json.Marshal(s)
	return j
}
