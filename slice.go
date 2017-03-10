package segmentedSlice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// DefaultSegmentLen is used if segLen is 0, mostly during an auto-constructed slice from JSON.
var DefaultSegmentLen = 128

// New returns a new Slice with the specified segment length.
// Length must be a power of two or 0, if it is 0 it will use the DefaultSegmentLen.
func New(segLen int) *Slice {
	return NewSortable(segLen, nil)
}

// NewSortable returns a Slice that supports the sort.Interface
// Length must be a power of two or 0, if it is 0 it will use the DefaultSegmentLen.
func NewSortable(segLen int, lessFn func(a, b interface{}) bool) *Slice {
	if !isPowerOfTwo(segLen) {
		panic("segLen is not power of two")
	}

	return &Slice{
		segLen: segLen - 1,
		shift:  findShift(segLen),
		lessFn: lessFn,
	}
}

// Slice is a special slice-of-slices, when it grows it creates a new internal slice
// rather than growing and copying data.
type Slice struct {
	len    int
	cap    int
	segLen int

	shift uint

	baseIdx int

	data   [][]interface{}
	lessFn func(a, b interface{}) bool

	typ reflect.Type
}

// Get returns the item at the specified index, if i > Cap(), it panics.
func (ss *Slice) Get(i int) interface{} {
	di, si := ss.index(ss.baseIdx + i)
	return ss.data[di][si]
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
	oss.Grow(ss.Len())
	ss.ForEach(func(i int, v interface{}) (breakNow bool) {
		oss.Append(v)
		return
	})
	return oss
}

// Pop deletes and returns the last item in the slice.
// If used on a sub-slice, it turns into an independent slice.
func (ss *Slice) Pop() (v interface{}) {
	ss.Grow(0)
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
	nss := NewSortable(ss.segLen+1, ss.lessFn)
	nss.Grow(ss.len)
	nss.typ, nss.len, nss.shift = ss.typ, ss.len, ss.shift
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

	if sz == 0 { // special case for cloning
		return 0
	}

	if ss.segLen < 1 {
		ss.segLen = DefaultSegmentLen - 1
		ss.shift = findShift(DefaultSegmentLen)
	}

	if sz = ss.len + sz; sz <= ss.cap {
		return 0
	}

	segLen := ss.segLen + 1
	newSize := 1 + (sz-ss.cap)/segLen

	for i := 0; i < newSize; i++ {
		ss.data = append(ss.data, make([]interface{}, segLen))
		ss.cap += segLen
	}
	//log.Println(sz, segLen, len(ss.data))
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
	return i >> ss.shift, i & ss.segLen
}

func (ss *Slice) ptrAt(i int) *interface{} {
	di, si := ss.index(i)
	// log.Println(i, di, si, ss.segLen)
	return &ss.data[di][si]
}

func isPowerOfTwo(n int) bool {
	return n > 0 && n&(n-1) == 0
}

func findShift(d int) (shift uint) {
	for d > 1 {
		d = d >> 1
		shift++
	}
	return
}
