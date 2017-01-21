package list

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

	data   [][]interface{}
	lessFn func(a, b interface{}) bool
}

// Get returns the item at the specified index, if i > Len(), it panics.
func (l *SegmentedSlice) Get(i int) interface{} {
	return *l.ptrAt(i)
}

// Set sets the value at the specified index, if i > Len(), it panics.
func (l *SegmentedSlice) Set(i int, v interface{}) {
	*l.ptrAt(i) = v
}

// Append appends vals to the slice.
func (l *SegmentedSlice) Append(vals ...interface{}) {
	l.grow(len(vals))
	for _, v := range vals {
		*l.ptrAt(l.len) = v
		l.len++
	}
}

// Pop deletes and returns the last item in the slice.
func (l *SegmentedSlice) Pop() (v interface{}) {
	p := l.ptrAt(l.len - 1)
	v = *p
	*p = nil
	l.len--
	return v
}

// ForEachAt loops over the slice and calls fn for each element.
// If fn returns true, it breaks early and returns true otherwise returns false.
func (l *SegmentedSlice) ForEachAt(i int, fn func(i int, v interface{}) (breakNow bool)) bool {
	di, si := l.index(i)
	for dii := di; dii < len(l.data); dii++ {
		s := l.data[dii]
		for sii := si; sii < len(s); sii++ {
			if fn(i, s[sii]) {
				return true
			}
			i++
		}
		si = 0 // only needed to be > 0 if we're starting at a specific index
	}

	return false
}

// ForEach is an alias for ForEachAt(0, fn).
func (l *SegmentedSlice) ForEach(fn func(i int, v interface{}) (breakNow bool)) bool {
	return l.ForEachAt(0, fn)
}

func (l *SegmentedSlice) IterAt(i int) <-chan interface{} {
	ch := make(chan interface{}, l.segLen)
	go l.ForEachAt(i, func(_ int, v interface{}) bool {
		ch <- v
		return false
	})
	return ch
}

func (l *SegmentedSlice) Iter() <-chan interface{} { return l.IterAt(0) }

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

// grow grows the data list returns the number of added segments
func (l *SegmentedSlice) grow(sz int) int {
	if sz = l.len + sz; sz <= l.cap {
		return 0
	}

	newSize := 1 + (sz-l.cap)/l.segLen

	//log.Println(l.len, l.cap, sz, sz <= l.cap, newSize)

	for i := 0; i < newSize; i++ {
		l.data = append(l.data, make([]interface{}, l.segLen))
		l.cap += l.segLen

	}

	return newSize
}

// index returns the internal data index and slice index for an index
func (l *SegmentedSlice) index(i int) (int, int) {
	return i / l.segLen, i % l.segLen
}

func (l *SegmentedSlice) ptrAt(i int) *interface{} {
	di, si := l.index(i)
	return &l.data[di][si]
}
