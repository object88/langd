package sigqueue

import (
	"errors"
)

const (
	heapMinimum = 8
)

// ErrHeapEmpty is returned for an operation that cannot work on an empty
// heap.
var ErrHeapEmpty = errors.New("Heap is empty")

// Heap is a heap data structure with minimal functionality to support a
// sigqueue.
// Behavior is not defined if the same value is entered more than once.
// Heap is not concurrency-safe.
type Heap struct {
	a    []int
	size int
}

// CreateHeap returns a new instance of a Heap.
func CreateHeap() *Heap {
	a := make([]int, heapMinimum)
	return &Heap{a, 0}
}

// Insert adds a new value to the heap, and sorts it.
func (h *Heap) Insert(i int) {
	if h.size == len(h.a) {
		h.grow()
	}

	h.a[h.size] = i
	h.size++

	h.up()
}

// Minimum returns the minimum value in the heap without pulling it off,
// or an error if the heap is empty.
func (h *Heap) Minimum() (int, error) {
	if h.size == 0 {
		return 0, ErrHeapEmpty
	}
	return h.a[0], nil
}

// RemoveMinimum pulls the minimum value off the heap and returns it.
// The heap is still sorted after this operation.
func (h *Heap) RemoveMinimum() (int, error) {
	i, err := h.Minimum()
	if err != nil {
		return 0, err
	}

	h.a[0] = h.a[h.size-1]
	h.a[h.size-1] = 0
	h.size--

	h.shrink()

	h.down()

	return i, nil
}

func (h *Heap) grow() {
	newSize := len(h.a) << 1
	a := make([]int, newSize)
	copy(a, h.a)
	h.a = a
}

func (h *Heap) shrink() {
	if len(h.a)>>1 <= heapMinimum || len(h.a)>>2 < h.size {
		return
	}

	newSize := len(h.a) >> 1
	a := h.a
	h.a = make([]int, newSize)
	copy(h.a, a)
}

func (h *Heap) down() {
	index := 0
	size := h.size
	for leftIndex := index<<1 + 1; leftIndex < size; leftIndex = index<<1 + 1 {
		rightIndex := index<<1 + 2
		smallerIndex := leftIndex
		leftValue := h.a[leftIndex]
		rightValue := h.a[rightIndex]
		if rightIndex < size && leftValue > rightValue {
			smallerIndex = rightIndex
		}
		indexValue := h.a[index]
		smallerValue := h.a[smallerIndex]
		if indexValue > smallerValue {
			h.a[index], h.a[smallerIndex] = h.a[smallerIndex], h.a[index]
		} else {
			break
		}
		index = smallerIndex
	}
}

// Performs the "bubble up" operation. This is to place a newly inserted
// element (i.e. last element in the list) in its correct place so that
// the heap maintains the min/max-heap order property.
func (h *Heap) up() {
	index := h.size - 1
	for parentIndex := (index - 1) >> 1; index > 0; parentIndex = (index - 1) >> 1 {
		if h.a[parentIndex] <= h.a[index] {
			break
		}
		h.a[index], h.a[parentIndex] = h.a[parentIndex], h.a[index]
		index = parentIndex
	}
}
