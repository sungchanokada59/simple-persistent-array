// Package simple_persistent_array provides an immutable array with O(log n)
// reads and writes. Updates return a new version that shares structure with
// the previous one, so old versions remain valid and cheap to keep alive.
package simple_persistent_array

import "errors"

// ErrIndexOutOfRange is returned when an index falls outside [0, Length()).
var ErrIndexOutOfRange = errors.New("simple_persistent_array: index out of range")

// branchingFactor is the number of children per internal node. 32 keeps the
// tree shallow (depth ~ log32 n) while leaving room for a bit-mask trie-style
// index computation. It is a power of two so we can shift instead of divide.
const branchingFactor = 32

// shift is log2(branchingFactor).
const shift = 5

// mask selects the low `shift` bits of an index.
const mask = branchingFactor - 1

// Array is a persistent (immutable) array of interface{} values. The zero
// value is a valid empty array. Every mutating-looking method returns a new
// *Array; the receiver is never modified.
type Array struct {
	// length is the number of elements visible to callers. It may be smaller
	// than the capacity implied by the tree when nodes are shared.
	length int
	// root is the top internal node. It is nil exactly when length == 0.
	root *node
}

// node is an internal or leaf node in the tree. Leaf nodes have depth 0 and
// store values directly; internal nodes have depth > 0 and store children.
type node struct {
	// children holds either sub-nodes (internal) or values (leaf).
	children []interface{}
}

// New returns an empty array.
func New() *Array {
	return &Array{}
}

// From returns an array containing the given values in order. The slice is
// copied, so later mutation of the caller's slice does not affect the array.
func From(values []interface{}) *Array {
	a := &Array{}
	for _, v := range values {
		a = a.Set(a.length, v)
	}
	return a
}

// Length reports the number of elements in the array.
func (a *Array) Length() int {
	if a == nil {
		return 0
	}
	return a.length
}

// Get returns the value at index. It returns ErrIndexOutOfRange if index is
// negative or >= Length().
func (a *Array) Get(index int) (interface{}, error) {
	if a == nil || index < 0 || index >= a.length {
		return nil, ErrIndexOutOfRange
	}
	n := a.root
	// Walk from the root down to the leaf holding index. The number of
	// levels is determined by the current depth of the tree.
	for level := a.depth(); level > 0; level-- {
		n = n.children[(index>>(level*shift))&mask].(*node)
	}
	return n.children[index&mask], nil
}

// Set returns a new array identical to a except that index holds value.
//
// If index == Length() the value is appended, which is the only way to grow
// the array. Any larger index returns ErrIndexOutOfRange. Nodes along the
// path to index are copied; everything else is shared with a.
func (a *Array) Set(index int, value interface{}) *Array {
	if a == nil {
		a = &Array{}
	}
	if index < 0 || index > a.length {
		// The caller cannot observe this error, so we return the receiver
		// unchanged. Use SetChecked when the error matters.
		return a
	}
	if index == a.length {
		return a.append(value)
	}
	return &Array{
		length: a.length,
		root:   a.root.update(a.depth(), index, value),
	}
}

// SetChecked is Set but reports ErrIndexOutOfRange instead of silently
// returning the receiver.
func (a *Array) SetChecked(index int, value interface{}) (*Array, error) {
	if a == nil {
		a = &Array{}
	}
	if index < 0 || index > a.length {
		return nil, ErrIndexOutOfRange
	}
	return a.Set(index, value), nil
}

// append grows the array by one, extending the tree if needed.
func (a *Array) append(value interface{}) *Array {
	capacity := 1 << (uint(a.depth()+1) * shift)
	if a.length < capacity {
		// There is spare room in the existing tree; copy the path and add.
		return &Array{
			length: a.length + 1,
			root:   a.root.append(a.depth(), a.length, value),
		}
	}
	// The tree is full, so we grow a new root one level taller and keep the
	// old root as its first child.
	newRoot := &node{children: make([]interface{}, 1, branchingFactor)}
	newRoot.children[0] = a.root
	newRoot = newRoot.append(a.depth()+1, a.length, value)
	return &Array{length: a.length + 1, root: newRoot}
}

// depth returns the number of internal levels above the leaves. An empty
// array has depth 0.
func (a *Array) depth() int {
	if a == nil || a.root == nil {
		return 0
	}
	d := 0
	for n := a.root; ; {
		if len(n.children) == 0 {
			return d
		}
		child, ok := n.children[0].(*node)
		if !ok {
			return d
		}
		n = child
		d++
	}
}

// update returns a copy of n with the value at index replaced. It copies
// only the nodes on the path from n to the leaf containing index.
func (n *node) update(level, index int, value interface{}) *node {
	if n == nil {
		n = &node{children: make([]interface{}, 0, branchingFactor)}
	}
	out := &node{children: make([]interface{}, len(n.children))}
	copy(out.children, n.children)
	if level == 0 {
		out.children[index&mask] = value
		return out
	}
	slot := (index >> (uint(level) * shift)) & mask
	child, _ := out.children[slot].(*node)
	out.children[slot] = child.update(level-1, index, value)
	return out
}

// append returns a copy of n with value inserted at index. The index is
// always exactly one past the current end of the array at this level.
func (n *node) append(level, index int, value interface{}) *node {
	if n == nil {
		n = &node{children: make([]interface{}, 0, branchingFactor)}
	}
	out := &node{children: make([]interface{}, len(n.children))}
	copy(out.children, n.children)
	if level == 0 {
		out.children = append(out.children, value)
		return out
	}
	slot := (index >> (uint(level) * shift)) & mask
	if slot < len(out.children) {
		child, _ := out.children[slot].(*node)
		out.children[slot] = child.append(level-1, index, value)
	} else {
		// A fresh subtree is needed on this branch.
		fresh := &node{children: make([]interface{}, 0, branchingFactor)}
		out.children = append(out.children, fresh.append(level-1, index, value))
	}
	return out
}
