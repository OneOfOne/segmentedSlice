# segmentedSlice
A fast, index-able, sort-able, grow-only Slice.

```go
➤ go test -benchmem -bench=. -benchtime=5s
BenchmarkAppendSegmentedSlice-8         200000000               47.3 ns/op            27 B/op          1 allocs/op
BenchmarkAppendNormalSlice-8            20000000                 342 ns/op            88 B/op          1 allocs/op
PASS
ok      github.com/OneOfOne/segmentedSlice      21.906s
```
