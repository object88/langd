package sigqueue

import (
	"testing"
)

func Test_Heap_Create(t *testing.T) {
	h := CreateHeap()
	if h == nil {
		t.Errorf("Got nil")
	}
	if len(h.a) != heapMinimum {
		t.Errorf("Did not create minimum internal array")
	}
	if h.size != 0 {
		t.Errorf("Size incorrect; expected 0, got %d", h.size)
	}
	for k, v := range h.a {
		if v != 0 {
			t.Errorf("Incorrect value %d at %d", v, k)
		}
	}
	_, err := h.Minimum()
	if err == nil {
		t.Error("Minimum did not return error")
	}
	if err != ErrHeapEmpty {
		t.Error("Did not get mepty heap error")
	}
}

func Test_Heap_Insert(t *testing.T) {
	h := CreateHeap()
	h.Insert(1)

	if len(h.a) != heapMinimum {
		t.Errorf("Internal array has unexpected size %d", len(h.a))
	}
	if h.size != 1 {
		t.Errorf("Size incorrect; expected 1, got %d", h.size)
	}
	if h.a[0] != 1 {
		t.Errorf("Unexpected internal state")
	}
	for k, v := range h.a[h.size:] {
		if v != 0 {
			t.Errorf("Incorrect value %d at %d", v, k)
		}
	}
	m, err := h.Minimum()
	if err != nil {
		t.Errorf("Unexpected error getting minimum: %s", err.Error())
	}
	if m != 1 {
		t.Errorf("Incorrect minimum %d", m)
	}
}

func Test_Heap_Inserts(t *testing.T) {
	h := CreateHeap()
	h.Insert(3) // [3,0,0,0,0,0,0,0]
	h.Insert(5) // [3,5,0,0,0,0,0,0]
	h.Insert(1) // [1,5,3,0,0,0,0,0]

	expected := []int{1, 5, 3}

	if len(h.a) != heapMinimum {
		t.Errorf("Internal array has unexpected size %d", len(h.a))
	}
	if h.size != 3 {
		t.Errorf("Size incorrect; expected 1, got %d", h.size)
	}
	for k, v := range expected {
		if h.a[k] != v {
			t.Errorf("Unexpected internal state %d at %d", h.a[k], k)
		}
	}
	for k, v := range h.a[h.size:] {
		if v != 0 {
			t.Errorf("Incorrect value %d at %d", v, k)
		}
	}
}

func Test_Heap_Remove_Empty(t *testing.T) {
	h := CreateHeap()

	_, err := h.RemoveMinimum()
	if err == nil {
		t.Error("Minimum did not return error")
	}
	if err != ErrHeapEmpty {
		t.Error("Did not get mepty heap error")
	}
}

func Test_Heap_Remove(t *testing.T) {
	h := CreateHeap()
	h.Insert(1) // [1,0,0,0,0,0,0,0]

	m, err := h.RemoveMinimum() // [0,0,0,0,0,0,0,0]
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if m != 1 {
		t.Errorf("Unexpected value %d", m)
	}
	if h.size != 0 {
		t.Errorf("Unexpected size %d", h.size)
	}
	for k, v := range h.a[h.size:] {
		if v != 0 {
			t.Errorf("Incorrect value %d at %d", v, k)
		}
	}
}

func Test_Heap_Removes(t *testing.T) {
	h := CreateHeap()
	h.Insert(3) // [3,0,0,0,0,0,0,0]
	h.Insert(5) // [3,5,0,0,0,0,0,0]
	h.Insert(1) // [1,5,3,0,0,0,0,0]

	m, err := h.RemoveMinimum() // [3,5,0,0,0,0,0,0]
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if m != 1 {
		t.Errorf("Unexpected value %d", m)
	}

	expected := []int{3, 5}

	if h.size != 2 {
		t.Errorf("Unexpected size %d", h.size)
	}
	for k, v := range expected {
		if h.a[k] != v {
			t.Errorf("Unexpected internal state %d at %d", h.a[k], k)
		}
	}
	for k, v := range h.a[h.size:] {
		if v != 0 {
			t.Errorf("Incorrect value %d at %d", v, k)
		}
	}
}

func Test_Heap_Grow(t *testing.T) {
	h := CreateHeap()
	inserts := []int{4, 6, 2, 8, 1, 3, 9, 5, 7}
	// [4, 0, 0, 0, 0, 0, 0, 0]
	// [4, 6, 0, 0, 0, 0, 0, 0]
	// [2, 6, 4, 0, 0, 0, 0, 0]
	// [2, 6, 4, 8, 0, 0, 0, 0]
	// [1, 2, 4, 8, 6, 0, 0, 0]
	// [1, 2, 3, 8, 6, 4, 0, 0]
	// [1, 2, 3, 8, 6, 4, 9, 0]
	// [1, 2, 3, 5, 6, 4, 9, 8]
	// [1, 2, 3, 5, 6, 4, 9, 8, 7, 0, 0, 0, 0, 0, 0, 0]
	for _, v := range inserts {
		h.Insert(v)
	}

	expected := []int{1, 2, 3, 5, 6, 4, 9, 8, 7}

	if len(h.a) != 16 {
		t.Errorf("Internal array has unexpected size %d", len(h.a))
	}
	if h.size != 9 {
		t.Errorf("Size incorrect; expected 1, got %d", h.size)
	}
	for k, v := range expected {
		if h.a[k] != v {
			t.Errorf("Unexpected internal state %d at %d", h.a[k], k)
		}
	}
	for k, v := range h.a[h.size:] {
		if v != 0 {
			t.Errorf("Incorrect value %d at %d", v, k)
		}
	}
}

func Test_Heap_Grow_No_Shrink(t *testing.T) {
	h := CreateHeap()

	for i := 0; i < 9; i++ {
		h.Insert(i + 1)
	}

	h.RemoveMinimum()
	h.RemoveMinimum()

	if h.size != 7 {
		t.Errorf("Unexpected shrink to %d", h.size)
	}
	if len(h.a) != 16 {
		t.Errorf("Unexpected shrink to %d", h.size)
	}
}

func Test_Heap_Grow_Shrink(t *testing.T) {
	h := CreateHeap()

	for i := 0; i < 33; i++ {
		h.Insert(i + 1)
	}

	for i := 0; i < 25; i++ {
		h.RemoveMinimum()
	}

	if h.size != 8 {
		t.Errorf("Unexpected shrink to %d", h.size)
	}
	if len(h.a) != 16 {
		t.Errorf("Unexpected shrink to %d", h.size)
	}
}
