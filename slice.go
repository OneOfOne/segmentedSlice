package segmentedSlice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// DefaultSegmentLen is used if segLen is 0, mostly during an auto-constructed slice from JSON.
var DefaultSegmentLen = 100

// New returns a new SegmentedSlice with the specified segment length
func New(segLen int) *SegmentedSlice {
	return NewSortable(segLen, nil)
}

// NewSortable returns a SegmentedSlice that supports the sort.Interface
func NewSortable(segLen int, lessFn func(a, b interface{}) bool) *SegmentedSlice {
	return &SegmentedSlice{
		segLen: segLen,
		lessFn: lessFn,
	}
}

// SegmentedSlice is a special slice-of-slices, when it grows it creates a new internal slice
// rather than growing and copying data.
type SegmentedSlice struct {
	len    int
	cap    int
	segLen int

	baseIdx int

	data   [][]interface{}
	lessFn func(a, b interface{}) bool

	typ reflect.Type
}

// Get returns the item at the specified index, if i > Cap(), it panics.
func (l *SegmentedSlice) Get(i int) interface{} {
	return *l.ptrAt(l.baseIdx + i)
}

// Set sets the value at the specified index, if i > Cap(), it panics.
func (l *SegmentedSlice) Set(i int, v interface{}) {
	*l.ptrAt(l.baseIdx + i) = v
}

// Append appends vals to the slice.
// If used on a sub-slice, it turns into an independent slice.
func (l *SegmentedSlice) Append(vals ...interface{}) {
	l.Grow(len(vals))
	for _, v := range vals {
		*l.ptrAt(l.len) = v
		l.len++
	}
}

// AppendTo appends all the data in the current slice to `other` and returns `other`.
func (l *SegmentedSlice) AppendTo(oss *SegmentedSlice) *SegmentedSlice {
	// TODO optimize
	l.ForEach(func(i int, v interface{}) (breakNow bool) {
		oss.Append(v)
		return
	})
	return oss
}

// Pop deletes and returns the last item in the slice.
// If used on a sub-slice, it turns into an independent slice.
func (l *SegmentedSlice) Pop() (v interface{}) {
	if l.baseIdx != 0 {
		panic("can't pop on a sub slice")
	}
	p := l.ptrAt(l.len - 1)
	v = *p
	*p = nil
	l.len--
	return v
}

// ForEachAt loops over the slice and calls fn for each element.
// If fn returns true, it breaks early and returns true otherwise returns false.
func (l *SegmentedSlice) ForEachAt(i int, fn func(i int, v interface{}) (breakNow bool)) bool {
	di, si := l.index(l.baseIdx + i)
	for dii := di; dii < len(l.data); dii++ {
		s := l.data[dii]
		for sii := si; sii < len(s); sii++ {
			if fn(i, s[sii]) {
				return true
			}
			if i++; i == l.len {
				return false
			}
		}
		si = 0 // only needed to be > 0 if we're starting at a specific index
	}

	return false
}

// ForEach is an alias for ForEachAt(0, fn).
func (l *SegmentedSlice) ForEach(fn func(i int, v interface{}) (breakNow bool)) bool {
	return l.ForEachAt(0, fn)
}

// IterAt returns an Iterator object
// Example:
// 	for it := ss.IterAt(0, ss.Len()); it.More(); {
// 		log.Println(it.Next())
// 	}
func (l *SegmentedSlice) IterAt(start, end int) *Iterator {
	return &Iterator{
		ss:    l,
		start: start,
		end:   end,
	}
}

// Iter is an alias for IterAt(0, ss.Len()).
func (l *SegmentedSlice) Iter() *Iterator { return l.IterAt(0, l.Len()) }

// Slice returns a sub-slice, the equivalent of ss[start:end], modifying any data in the returned slice modifies the parent.
func (l *SegmentedSlice) Slice(start, end int) *SegmentedSlice {
	cp := *l
	cp.len, cp.baseIdx = end-start, start
	return &cp
}

// Copy returns an exact copy of the slice that could be used independently.
// Copy is internally used if you call Append, Pop or Grow on a sub-slice.
func (l *SegmentedSlice) Copy() *SegmentedSlice {
	nss := NewSortable(l.segLen, l.lessFn)
	nss.Grow(l.len)
	nss.typ, nss.len = l.typ, l.len
	l.ForEach(func(i int, v interface{}) (_ bool) {
		nss.Set(i, v)
		return
	})
	return nss
}

// Grow grows internal data structure to fit `sz` amount of new items.
// If used on a sub-slice, it turns into an independent slice.
func (l *SegmentedSlice) Grow(sz int) int {
	if l.baseIdx != 0 {
		cp := l.Copy()
		*l = *cp
	}

	if l.segLen == 0 {
		l.segLen = DefaultSegmentLen
	}

	if sz = l.len + sz; sz <= l.cap {
		return 0
	}

	newSize := 1 + (sz-l.cap)/l.segLen

	for i := 0; i < newSize; i++ {
		l.data = append(l.data, make([]interface{}, l.segLen))
		l.cap += l.segLen
	}

	return newSize
}

// Len returns the number of elements in the slice.
func (l *SegmentedSlice) Len() int { return l.len }

// Cap returns the max number of elements the slice can hold before growinging
func (l *SegmentedSlice) Cap() int { return l.cap }

// Segments returns the number of segments.
func (l *SegmentedSlice) Segments() int { return len(l.data) }

// Less adds support for sort.Interface
func (l *SegmentedSlice) Less(i, j int) bool { return l.lessFn(l.Get(i), l.Get(j)) }

// Swap adds support for sort.Interface
func (l *SegmentedSlice) Swap(i, j int) {
	a, b := l.ptrAt(i), l.ptrAt(j)
	*a, *b = *b, *a
}

// MarshalJSON implements json.Marshaler
func (l *SegmentedSlice) MarshalJSON() ([]byte, error) {
	if l.Len() == 0 {
		return []byte("[]"), nil
	}

	b := bytes.NewBuffer(make([]byte, 0, 2+(5*l.Len())))

	b.WriteByte('[')

	l.ForEach(func(i int, v interface{}) (_ bool) {
		j, _ := json.Marshal(v)
		b.Write(j)
		b.WriteByte(',')
		return
	})

	b.Bytes()[b.Len()-1] = ']'

	return b.Bytes(), nil
}

// SetUnmarshalType sets the internal type used for UnmarshalJSON.
// Example:
// 	ss.SetUnmarshalType(&DataStruct{})
// 	ss.SetUnmarshalType(reflect.TypeOf(&DataStruct{}))
func (l *SegmentedSlice) SetUnmarshalType(val interface{}) {
	switch val := val.(type) {
	case nil:
		l.typ = nil
	case reflect.Type:
		l.typ = val
	case reflect.Value:
		l.typ = val.Type()
	default:
		l.typ = reflect.TypeOf(val)
	}
}

// UnmarshalJSON implements json.Unmarshaler
func (l *SegmentedSlice) UnmarshalJSON(b []byte) (err error) {
	var (
		dec = json.NewDecoder(bytes.NewReader(b))
		t   json.Token
	)

	if t, err = dec.Token(); err != nil {
		return
	}

	if d, ok := t.(json.Delim); !ok || d != '[' {
		return fmt.Errorf("expected '[', got: %v (%T)", t, t)
	}

	if l.typ != nil {
		for dec.More() {
			v := reflect.New(l.typ)
			if err = dec.Decode(v.Interface()); err != nil {
				return
			}
			l.Append(v.Elem().Interface())
		}
	} else {
		for dec.More() {
			var v interface{}
			if err = dec.Decode(&v); err != nil {
				return
			}
			l.Append(v)
		}
	}

	if t, err = dec.Token(); err != nil {
		return
	}

	if d, ok := t.(json.Delim); !ok || d != ']' {
		return fmt.Errorf("expected ']', got: %v (%T)", t, t)
	}

	return nil
}

// index returns the internal data index and slice index for an index
func (l *SegmentedSlice) index(i int) (int, int) {
	return i / l.segLen, i % l.segLen
}

func (l *SegmentedSlice) ptrAt(i int) *interface{} {
	di, si := l.index(i)
	return &l.data[di][si]
}
