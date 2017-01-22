package segmentedSlice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// DefaultSegmentLen is used if segLen is 0, mostly during an auto-constructed slice from JSON.
var DefaultSegmentLen = 100

// New returns a new Slice with the specified segment length
func New(segLen int) *Slice {
	return NewSortable(segLen, nil)
}

// NewSortable returns a Slice that supports the sort.Interface
func NewSortable(segLen int, lessFn func(a, b interface{}) bool) *Slice {
	return &Slice{
		segLen: segLen,
		lessFn: lessFn,
	}
}

// Slice is a special slice-of-slices, when it grows it creates a new internal slice
// rather than growing and copying data.
type Slice struct {
	len    int
	cap    int
	segLen int

	baseIdx int

	data   [][]interface{}
	lessFn func(a, b interface{}) bool

	typ reflect.Type
}

// Get returns the item at the specified index, if i > Cap(), it panics.
func (ss *Slice) Get(i int) interface{} {
	return *ss.ptrAt(ss.baseIdx + i)
}

// Set sets the value at the specified index, if i > Cap(), it panics.
func (ss *Slice) Set(i int, v interface{}) {
	*ss.ptrAt(ss.baseIdx + i) = v
}

// Append appends vals to the slice.
// If used on a sub-slice, it turns into an independent slice.
func (ss *Slice) Append(vals ...interface{}) {
	ss.Grow(len(vals))
	for _, v := range vals {
		*ss.ptrAt(ss.len) = v
		ss.len++
	}
}

// AppendTo appends all the data in the current slice to `other` and returns `other`.
func (ss *Slice) AppendTo(oss *Slice) *Slice {
	// TODO optimize
	ss.ForEach(func(i int, v interface{}) (breakNow bool) {
		oss.Append(v)
		return
	})
	return oss
}

// Pop deletes and returns the last item in the slice.
// If used on a sub-slice, it turns into an independent slice.
func (ss *Slice) Pop() (v interface{}) {
	if ss.baseIdx != 0 {
		panic("can't pop on a sub slice")
	}
	p := ss.ptrAt(ss.len - 1)
	v = *p
	*p = nil
	ss.len--
	return v
}

// ForEachAt loops over the slice and calls fn for each element.
// If fn returns true, it breaks early and returns true otherwise returns false.
func (ss *Slice) ForEachAt(i int, fn func(i int, v interface{}) (breakNow bool)) bool {
	di, si := ss.index(ss.baseIdx + i)
	for dii := di; dii < len(ss.data); dii++ {
		s := ss.data[dii]
		for sii := si; sii < len(s); sii++ {
			if fn(i, s[sii]) {
				return true
			}
			if i++; i == ss.len {
				return false
			}
		}
		si = 0 // only needed to be > 0 if we're starting at a specific index
	}

	return false
}

// ForEach is an alias for ForEachAt(0, fn).
func (ss *Slice) ForEach(fn func(i int, v interface{}) (breakNow bool)) bool {
	return ss.ForEachAt(0, fn)
}

// IterAt returns an Iterator object
// Example:
// 	for it := ss.IterAt(0, ss.Len()); it.More(); {
// 		log.Println(it.Next())
// 	}
func (ss *Slice) IterAt(start, end int) *Iterator {
	return &Iterator{
		ss:    ss,
		start: start,
		end:   end,
	}
}

// Iter is an alias for IterAt(0, ss.Len()).
func (ss *Slice) Iter() *Iterator { return ss.IterAt(0, ss.Len()) }

// Slice returns a sub-slice, the equivalent of ss[start:end], modifying any data in the returned slice modifies the parent.
func (ss *Slice) Slice(start, end int) *Slice {
	cp := *ss
	cp.len, cp.baseIdx = end-start, start
	return &cp
}

// Copy returns an exact copy of the slice that could be used independently.
// Copy is internally used if you call Append, Pop or Grow on a sub-slice.
func (ss *Slice) Copy() *Slice {
	nss := NewSortable(ss.segLen, ss.lessFn)
	nss.Grow(ss.len)
	nss.typ, nss.len = ss.typ, ss.len
	ss.ForEach(func(i int, v interface{}) (_ bool) {
		nss.Set(i, v)
		return
	})
	return nss
}

// Grow grows internal data structure to fit `sz` amount of new items.
// If used on a sub-slice, it turns into an independent slice.
func (ss *Slice) Grow(sz int) int {
	if ss.baseIdx != 0 {
		cp := ss.Copy()
		*ss = *cp
	}

	if ss.segLen == 0 {
		ss.segLen = DefaultSegmentLen
	}

	if sz = ss.len + sz; sz <= ss.cap {
		return 0
	}

	newSize := 1 + (sz-ss.cap)/ss.segLen

	for i := 0; i < newSize; i++ {
		ss.data = append(ss.data, make([]interface{}, ss.segLen))
		ss.cap += ss.segLen
	}

	return newSize
}

// Len returns the number of elements in the slice.
func (ss *Slice) Len() int { return ss.len }

// Cap returns the max number of elements the slice can hold before growinging
func (ss *Slice) Cap() int { return ss.cap }

// Segments returns the number of segments.
func (ss *Slice) Segments() int { return len(ss.data) }

// Less adds support for sort.Interface
func (ss *Slice) Less(i, j int) bool { return ss.lessFn(ss.Get(i), ss.Get(j)) }

// Swap adds support for sort.Interface
func (ss *Slice) Swap(i, j int) {
	a, b := ss.ptrAt(i), ss.ptrAt(j)
	*a, *b = *b, *a
}

// MarshalJSON implements json.Marshaler
func (ss *Slice) MarshalJSON() ([]byte, error) {
	if ss.Len() == 0 {
		return []byte("[]"), nil
	}

	var (
		b   = bytes.NewBuffer(make([]byte, 0, 2+(6*ss.Len())))
		enc = json.NewEncoder(b)
		it  = ss.Iter()
	)

	b.WriteByte('[')
	for {
		enc.Encode(it.Next())
		if !it.More() {
			break
		}
		b.WriteString(",")
	}

	b.WriteByte(']')

	return b.Bytes(), nil
}

// SetUnmarshalType sets the internal type used for UnmarshalJSON.
// Example:
// 	ss.SetUnmarshalType(&DataStruct{})
// 	ss.SetUnmarshalType(reflect.TypeOf(&DataStruct{}))
func (ss *Slice) SetUnmarshalType(val interface{}) {
	switch val := val.(type) {
	case nil:
		ss.typ = nil
	case reflect.Type:
		ss.typ = val
	case reflect.Value:
		ss.typ = val.Type()
	default:
		ss.typ = reflect.TypeOf(val)
	}
}

// UnmarshalJSON implements json.Unmarshaler
func (ss *Slice) UnmarshalJSON(b []byte) (err error) {
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

	if ss.typ != nil {
		for dec.More() {
			v := reflect.New(ss.typ)
			if err = dec.Decode(v.Interface()); err != nil {
				return
			}
			ss.Append(v.Elem().Interface())
		}
	} else {
		for dec.More() {
			var v interface{}
			if err = dec.Decode(&v); err != nil {
				return
			}
			ss.Append(v)
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

// String implements fmt.Stringer
func (ss *Slice) String() string {
	var (
		b  = bytes.NewBuffer(make([]byte, 0, 2+(5*ss.Len())))
		it = ss.Iter()
	)

	b.WriteByte('[')

	for {
		fmt.Fprintf(b, "%v", it.Next())
		if !it.More() {
			break
		}
		b.WriteString(", ")
	}

	b.WriteByte(']')

	return b.String()
}

// GoString implements fmt.GoStringer
func (ss *Slice) GoString() string {
	return fmt.Sprintf("&Slice{Len: %d, Cap: %d, Segments: %d, Data: %s }", ss.Len(), ss.Cap(), ss.Segments(), ss.String())
}

// index returns the internal data index and slice index for an index
func (ss *Slice) index(i int) (int, int) {
	return i / ss.segLen, i % ss.segLen
}

func (ss *Slice) ptrAt(i int) *interface{} {
	di, si := ss.index(i)
	return &ss.data[di][si]
}
