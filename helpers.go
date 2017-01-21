package segmentedSlice

// Iterator is a SegmentedSlice iterator.
type Iterator struct {
	ss         *SegmentedSlice
	start, end int
}

// More returns true if the iterator have more items/
func (it *Iterator) More() bool {
	return it.start < it.end
}

// Next returns the next item.
func (it *Iterator) Next() (val interface{}) {
	val = it.ss.Get(it.start)
	it.start++
	return
}

// NextIndex returns the next item and index.
func (it *Iterator) NextIndex() (idx int, val interface{}) {
	idx, val = it.start, it.ss.Get(it.start)
	it.start++
	return
}
