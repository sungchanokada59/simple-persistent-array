# simple-persistent-array

An immutable array of `interface{}` values with O(log n) reads and writes, where every update returns a new version that shares structure with the previous one.

```go
package main

import (
	"fmt"

	spa "simple_persistent_array"
)

func main() {
	v1 := spa.From([]interface{}{"a", "b", "c"})
	v2 := v1.Set(1, "B")

	x, _ := v1.Get(1)
	y, _ := v2.Get(1)
	fmt.Println(x, y)          // b B
	fmt.Println(v1.Length())   // 3

	// Set at Length() appends; that is the only way to grow.
	v3 := v2.Set(v2.Length(), "d")
	z, _ := v3.Get(3)
	fmt.Println(v3.Length(), z) // 4 d
}
```

## Why this exists

Mutating a slice in place is O(1) but destroys the previous contents, which makes it awkward to keep a history of an array (undo stacks, snapshots, structural sharing between related configurations). A copy-on-write slice is simple but costs O(n) per write. This library keeps the copy cheap: a write copies only the nodes on the path from the root to the changed leaf, so both the cost and the new allocation are O(log n). The trade-off is indirection — reads walk a tree instead of indexing a flat slice — and a per-element `interface{}` box, which is the price of a single container type in a language without generics in the standard library.

## API

- `New() *Array` — empty array.
- `From(values []interface{}) *Array` — array holding a copy of `values`.
- `(*Array).Length() int` — number of elements.
- `(*Array).Get(index int) (interface{}, error)` — value at `index`, or `ErrIndexOutOfRange`.
- `(*Array).Set(index int, value interface{}) *Array` — new array with `value` at `index`. If `index == Length()` this appends. If `index` is out of range the receiver is returned unchanged.
- `(*Array).SetChecked(index int, value interface{}) (*Array, error)` — like `Set` but reports `ErrIndexOutOfRange` for bad indices.
- `ErrIndexOutOfRange` — sentinel error for the above.

The zero value of `Array` is a valid empty array, so `var a Array` works without calling `New`.

## The awkward edge

`Set` cannot return an error and a value at the same time without making the common append path clumsy, so it silently returns the receiver when the index is out of range. If you need to distinguish "wrote at index 10" from "index 10 was too large", call `SetChecked`, which returns `ErrIndexOutOfRange` instead. This is the one place where the API makes you choose between ergonomics and error reporting; it does not try to do both.
