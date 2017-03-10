package segmentedSlice_test

import (
	"fmt"

	"github.com/OneOfOne/segmentedSlice"
)

func ExampleBasicUsage() {
	ss := segmentedSlice.New(8)

	for i := 0; i < 10; i++ {
		ss.Append(i * i)
	}

	fmt.Print("data: ")
	for it := ss.Iter(); it.More(); {
		fmt.Print(it.Next())
		if it.More() {
			fmt.Print(", ")
		}
	}
	fmt.Println()
	fmt.Println("Slice:", ss)

	fmt.Printf("Before Pop: len: %d, cap: %d, segments: %d\n", ss.Len(), ss.Cap(), ss.Segments())

	v := ss.Pop()
	fmt.Printf("After Pop: val: %v, len: %d, cap: %d, segments: %d\n", v, ss.Len(), ss.Cap(), ss.Segments())

	fmt.Println("Slice:", ss.Slice(4, 7))
	// Output:
	// data: 0, 1, 4, 9, 16, 25, 36, 49, 64, 81
	// Slice: [0, 1, 4, 9, 16, 25, 36, 49, 64, 81]
	// Before Pop: len: 10, cap: 16, segments: 2
	// After Pop: val: 81, len: 9, cap: 16, segments: 2
	// Slice: [16, 25, 36]
}
