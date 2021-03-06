# SegmentedSlice [![GoDoc](http://godoc.org/github.com/OneOfOne/segmentedSlice?status.svg)](http://godoc.org/github.com/OneOfOne/segmentedSlice) [![Build Status](https://travis-ci.org/OneOfOne/segmentedSlice.svg?branch=master)](https://travis-ci.org/OneOfOne/segmentedSlice)
A fast, index-able, sort-able, grow-only Slice.

## FAQ

### Why?
* Appending to a normal slice can get slow and very memory heavy as the slice grows,
	and for a lot of work loads it's usually append-only with some sorting.


## Benchmarks

```go
➤ go18 test -bench=. -benchmem -benchtime=5s
BenchmarkAppendSegmentedSlice-8         200000000               42.4 ns/op            25 B/op          1 allocs/op
BenchmarkAppendNormalSlice-8            20000000               252 ns/op              88 B/op          1 allocs/op
PASS
ok      github.com/OneOfOne/segmentedSlice      20.752s
```
