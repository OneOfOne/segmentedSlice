package segmentedSlice

// ConsumeIter consumes an iter channel to prevent leaking memory.
func ConsumeIter(ch <-chan interface{}) {
	for range ch {
		/* awkward emptyness */
	}
}

type Iterator struct {
	ss *SegmentedSlice
	i  int
}

func (it *Iterator) More() bool {
	return it.i < it.ss.Len()
}

func (it *Iterator) Next() (val interface{}) {
	val = it.ss.Get(it.i)
	it.i++
	return
}

func (it *Iterator) NextIndex() (idx int, val interface{}) {
	idx, val = it.i, it.ss.Get(it.i)
	it.i++
	return
}
