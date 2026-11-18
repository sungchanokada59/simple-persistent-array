package simple_persistent_array

import (
	"errors"
	"testing"
)

func TestEmptyArray(t *testing.T) {
	a := New()
	if got := a.Length(); got != 0 {
		t.Fatalf("Length() = %d, want 0", got)
	}
	if _, err := a.Get(0); !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("Get(0) err = %v, want ErrIndexOutOfRange", err)
	}
	if _, err := a.Get(-1); !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("Get(-1) err = %v, want ErrIndexOutOfRange", err)
	}
}

func TestZeroValueIsUsable(t *testing.T) {
	var a Array
	if got := a.Length(); got != 0 {
		t.Fatalf("Length() = %d, want 0", got)
	}
	a = *a.Set(0, "x")
	if got := a.Length(); got != 1 {
		t.Fatalf("Length() = %d, want 1", got)
	}
	v, err := a.Get(0)
	if err != nil {
		t.Fatalf("Get(0) err = %v", err)
	}
	if v != "x" {
		t.Fatalf("Get(0) = %v, want x", v)
	}
}

func TestSetAppendAndGet(t *testing.T) {
	a := New()
	for i := 0; i < 100; i++ {
		a = a.Set(i, i)
	}
	if got := a.Length(); got != 100 {
		t.Fatalf("Length() = %d, want 100", got)
	}
	for i := 0; i < 100; i++ {
		v, err := a.Get(i)
		if err != nil {
			t.Fatalf("Get(%d) err = %v", i, err)
		}
		if v != i {
			t.Fatalf("Get(%d) = %v, want %d", i, v, i)
		}
	}
}

func TestSetReplacesWithoutMutating(t *testing.T) {
	a := From([]interface{}{"a", "b", "c"})
	b := a.Set(1, "B")

	if got := a.Length(); got != 3 {
		t.Fatalf("a.Length() = %d, want 3", got)
	}
	if got := b.Length(); got != 3 {
		t.Fatalf("b.Length() = %d, want 3", got)
	}

	av, err := a.Get(1)
	if err != nil {
		t.Fatalf("a.Get(1) err = %v", err)
	}
	if av != "b" {
		t.Fatalf("a.Get(1) = %v, want b", av)
	}
	bv, err := b.Get(1)
	if err != nil {
		t.Fatalf("b.Get(1) err = %v", err)
	}
	if bv != "B" {
		t.Fatalf("b.Get(1) = %v, want B", bv)
	}
}

func TestOldVersionsSurviveManyWrites(t *testing.T) {
	base := From([]interface{}{0, 1, 2, 3, 4})
	versions := []*Array{base}
	cur := base
	for i := 0; i < 50; i++ {
		cur = cur.Set(i%5, i)
		versions = append(versions, cur)
	}
	// The original must still report its original contents.
	for i := 0; i < 5; i++ {
		v, err := base.Get(i)
		if err != nil {
			t.Fatalf("base.Get(%d) err = %v", i, err)
		}
		if v != i {
			t.Fatalf("base.Get(%d) = %v, want %d", i, v, i)
		}
	}
	// And the final version must hold the last write at each index.
	for i := 0; i < 5; i++ {
		v, err := cur.Get(i)
		if err != nil {
			t.Fatalf("cur.Get(%d) err = %v", i, err)
		}
		want := 45 + i
		if v != want {
			t.Fatalf("cur.Get(%d) = %v, want %d", i, v, want)
		}
	}
}

func TestGetOutOfRange(t *testing.T) {
	a := From([]interface{}{1, 2, 3})
	if _, err := a.Get(3); !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("Get(3) err = %v, want ErrIndexOutOfRange", err)
	}
	if _, err := a.Get(-1); !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("Get(-1) err = %v, want ErrIndexOutOfRange", err)
	}
}

func TestSetCheckedRejectsOutOfRange(t *testing.T) {
	a := From([]interface{}{1, 2, 3})
	if _, err := a.SetChecked(4, 9); !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("SetChecked(4) err = %v, want ErrIndexOutOfRange", err)
	}
	if _, err := a.SetChecked(-1, 9); !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("SetChecked(-1) err = %v, want ErrIndexOutOfRange", err)
	}
	b, err := a.SetChecked(3, 4)
	if err != nil {
		t.Fatalf("SetChecked(3) err = %v", err)
	}
	if got := b.Length(); got != 4 {
		t.Fatalf("b.Length() = %d, want 4", got)
	}
}

func TestSetOutOfRangeReturnsReceiver(t *testing.T) {
	a := From([]interface{}{1, 2, 3})
	b := a.Set(10, 9)
	if b != a {
		t.Fatalf("Set(10) returned a different array for an invalid index")
	}
	if got := b.Length(); got != 3 {
		t.Fatalf("b.Length() = %d, want 3", got)
	}
}

func TestFromCopiesInput(t *testing.T) {
	src := []interface{}{1, 2, 3}
	a := From(src)
	src[0] = 99
	v, err := a.Get(0)
	if err != nil {
		t.Fatalf("Get(0) err = %v", err)
	}
	if v != 1 {
		t.Fatalf("Get(0) = %v, want 1", v)
	}
}

func TestNilValuesRoundTrip(t *testing.T) {
	a := From([]interface{}{nil, "x", nil})
	v, err := a.Get(0)
	if err != nil {
		t.Fatalf("Get(0) err = %v", err)
	}
	if v != nil {
		t.Fatalf("Get(0) = %v, want nil", v)
	}
	v, err = a.Get(1)
	if err != nil {
		t.Fatalf("Get(1) err = %v", err)
	}
	if v != "x" {
		t.Fatalf("Get(1) = %v, want x", v)
	}
}

func TestLargeArrayAcrossTreeGrowth(t *testing.T) {
	const n = 5000
	a := New()
	for i := 0; i < n; i++ {
		a = a.Set(i, i*2)
	}
	if got := a.Length(); got != n {
		t.Fatalf("Length() = %d, want %d", got, n)
	}
	for i := 0; i < n; i++ {
		v, err := a.Get(i)
		if err != nil {
			t.Fatalf("Get(%d) err = %v", i, err)
		}
		if v != i*2 {
			t.Fatalf("Get(%d) = %v, want %d", i, v, i*2)
		}
	}
}

func TestSharedStructureAfterAppend(t *testing.T) {
	base := New()
	for i := 0; i < 40; i++ {
		base = base.Set(i, i)
	}
	grown := base.Set(40, 40)
	if got := base.Length(); got != 40 {
		t.Fatalf("base.Length() = %d, want 40", got)
	}
	if got := grown.Length(); got != 41 {
		t.Fatalf("grown.Length() = %d, want 41", got)
	}
	// base must be unaffected by the append.
	if _, err := base.Get(40); !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("base.Get(40) err = %v, want ErrIndexOutOfRange", err)
	}
}
