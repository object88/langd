package sigqueue

import (
	"testing"
)

func Test_LinkedList_Create(t *testing.T) {
	ll := CreateLinkedList()
	if ll == nil {
		t.Error("Create returned nil")
	}

	if ll.head != nil || ll.tail != nil {
		t.Error("Newly created linked list has non-nil head or tail")
	}

	if ll.size != 0 {
		t.Errorf("Initial size is non-zero: %d", ll.size)
	}
}

func Test_LinkedList_Push_Nil(t *testing.T) {
	var ll *LinkedList

	f := 1
	ll0 := ll.Push(f)
	if ll0 == nil {
		t.Error("Pushing to a nil receiver did not create the linked list")
	}
	if ll0.head == nil || ll0.tail == nil {
		t.Error("Head or tail is nil")
	}
	if ll0.head != ll0.tail {
		t.Error("Head and tail point to different linked list items")
	}
	if ll0.head.n != f {
		t.Error("Head does not point to item")
	}
	if ll0.head.next != nil || ll0.head.prev != nil {
		t.Error("Head next or prev is not nil")
	}
	if ll0.size != 1 {
		t.Errorf("Incorrect size %d", ll.size)
	}
}

func Test_LinkedList_Push_Empty(t *testing.T) {
	ll := CreateLinkedList()

	f := 1
	ll0 := ll.Push(f)
	if ll0 == nil {
		t.Error("Push returned different pointer")
	}
	if ll.head == nil || ll.tail == nil {
		t.Error("Head or tail is nil")
	}
	if ll.head != ll.tail {
		t.Error("Head and tail point to different linked list items")
	}
	if ll.head.n != f {
		t.Error("Head does not point to item")
	}
	if ll.head.next != nil || ll.head.prev != nil {
		t.Error("Head next or prev is not nil")
	}
	if ll.size != 1 {
		t.Errorf("Incorrect size %d", ll.size)
	}
}

func Test_LinkedList_Push_One(t *testing.T) {
	f0 := 0
	f1 := 1
	ll := CreateLinkedList()
	ll.Push(f0)

	ll0 := ll.Push(f1)
	if ll0 == nil {
		t.Error("Push returned different pointer")
	}
	if ll.tail.n != f1 {
		t.Error("Tail item is not f1")
	}
	if ll.head == nil || ll.tail == nil {
		t.Error("Head or tail is nil")
	}
	if ll.head == ll.tail {
		t.Error("Head and tail point to same linked list items")
	}
	if ll.head.next != ll.tail {
		t.Error("Head.next is not tail")
	}
	if ll.tail.prev != ll.head {
		t.Error("Tail.prev is not head")
	}
	if ll.size != 2 {
		t.Errorf("Incorrect size %d", ll.size)
	}
}

func Test_LinkedList_Push_Two(t *testing.T) {
	f0 := 0
	f1 := 1
	f2 := 2
	ll := CreateLinkedList()
	ll.Push(f0)
	ll.Push(f1)

	ll0 := ll.Push(f2)
	if ll0 == nil {
		t.Error("Push returned different pointer")
	}
	if ll.tail.n != f2 {
		t.Error("Tail has wrong item")
	}
	if ll.tail.next != nil {
		t.Error("Tail has non-nil pointer")
	}
	if ll.tail.prev.next != ll.tail {
		t.Error("Tail.prev.next is not tail")
	}
}

func Test_LinkedList_Peek_Nil(t *testing.T) {
	var ll *LinkedList

	x0 := ll.Peek()
	if x0 != -1 {
		t.Error("Did not get -1 from nil pointer receiver")
	}
}

func Test_LinkedList_Peek_Empty(t *testing.T) {
	ll := CreateLinkedList()

	x0 := ll.Peek()
	if x0 != -1 {
		t.Error("Did not get -1 from nil pointer receiver")
	}
}

func Test_LinkedList_Peek_One(t *testing.T) {
	f := 1
	ll := CreateLinkedList()
	ll.Push(f)

	x0 := ll.Peek()
	if x0 == -1 {
		t.Error("Got -1")
	}
	if x0 != f {
		t.Error("Got wrong item from peek")
	}
}

func Test_LinkedList_Peek_Two(t *testing.T) {
	f0 := 0
	f1 := 1
	ll := CreateLinkedList()
	ll.Push(f0)
	ll.Push(f1)

	x0 := ll.Peek()
	if x0 == -1 {
		t.Error("Got -1")
	}
	if x0 != f0 {
		t.Error("Got wrong item from peek")
	}
}

func Test_LinkedList_Peer_Empty(t *testing.T) {
	ll := CreateLinkedList()

	x0 := ll.Peer()
	if x0 != -1 {
		t.Error("Did not get -1 from nil pointer receiver")
	}
}

func Test_LinkedList_Peer_One(t *testing.T) {
	f := 1
	ll := CreateLinkedList()
	ll.Push(f)

	x0 := ll.Peer()
	if x0 == -1 {
		t.Error("Got -1")
	}
	if x0 != f {
		t.Error("Got wrong item from peer")
	}
}

func Test_LinkedList_Peer_Two(t *testing.T) {
	f0 := 0
	f1 := 1
	ll := CreateLinkedList()
	ll.Push(f0)
	ll.Push(f1)

	x0 := ll.Peer()
	if x0 == -1 {
		t.Error("Got -1")
	}
	if x0 != f1 {
		t.Error("Got wrong item from peer")
	}
}

func Test_LinkedList_Pop_Nil(t *testing.T) {
	var ll *LinkedList

	x0 := ll.Pop()
	if x0 != -1 {
		t.Error("Got non-nil from nil pointer receiver")
	}
}

func Test_LinkedList_Pop_Empty(t *testing.T) {
	ll := CreateLinkedList()

	x0 := ll.Pop()
	if x0 != -1 {
		t.Error("Got non-nil from empty linked list")
	}
	if ll.size != 0 {
		t.Errorf("Incorrect size %d", ll.size)
	}
}

func Test_LinkedList_Pop_One(t *testing.T) {
	f := 1
	ll := CreateLinkedList()
	ll.Push(f)

	x0 := ll.Pop()
	if x0 == -1 {
		t.Error("Got -1")
	}
	if x0 != f {
		t.Error("Got wrong item")
	}
	if ll.head != nil || ll.tail != nil {
		t.Error("Head or tail is not nil")
	}
	if ll.size != 0 {
		t.Errorf("Incorrect size %d", ll.size)
	}
}

func Test_LinkedList_Pop_Two(t *testing.T) {
	f0 := 0
	f1 := 1
	ll := CreateLinkedList()
	ll.Push(f0)
	ll.Push(f1)

	x0 := ll.Pop()
	if x0 == -1 {
		t.Error("Got -1")
	}
	if x0 != f0 {
		t.Error("Got wrong item")
	}
	if ll.head == nil || ll.tail == nil {
		t.Error("Head or tail is nil")
	}
	if ll.head != ll.tail {
		t.Error("Head and tail point to different items")
	}
	if ll.head.n != f1 {
		t.Error("Head points to wrong item")
	}
	if ll.head.prev != nil {
		t.Error("Head.prev does not point to nil")
	}
	if ll.head.next != nil {
		t.Error("Head.next does not point to nil")
	}
	if ll.size != 1 {
		t.Errorf("Incorrect size %d", ll.size)
	}
}

func Test_LinkedList_Pop_Three(t *testing.T) {
	f0 := 0
	f1 := 1
	f2 := 2
	ll := CreateLinkedList()
	ll.Push(f0)
	ll.Push(f1)
	ll.Push(f2)

	x0 := ll.Pop()
	if x0 == -1 {
		t.Error("Got -1")
	}
	if x0 != f0 {
		t.Error("Got wrong item")
	}
	if ll.head == nil || ll.tail == nil {
		t.Error("Head or tail is nil")
	}
	if ll.head == ll.tail {
		t.Error("Head and tail point to same item")
	}
	if ll.head.n != f1 {
		t.Error("Head points to wrong item item")
	}
	if ll.tail.n != f2 {
		t.Error("Tail points to wrong item item")
	}
	if ll.head.prev != nil {
		t.Error("Head.prev is not nil")
	}
	if ll.head.next != ll.tail {
		t.Error("Head.next is not tail")
	}
	if ll.tail.prev != ll.head {
		t.Error("Tail.prev is not head")
	}
	if ll.tail.next != nil {
		t.Error("Tail.next is not nil")
	}
}

func Test_LinkedList_Sequence_Push_And_Pop(t *testing.T) {
	a := []int{0, 1, 2, 3, 4}

	ll := CreateLinkedList()
	for _, v := range a {
		f := v
		ll.Push(f)
	}

	for _, v := range a {
		x := ll.Pop()
		if x != v {
			t.Errorf("Bad value; expected %d, got %d", v, x)
		}
	}
}

func Test_LinkedList_Sequence(t *testing.T) {
	a := []int{0, 1, 2, 3, 4}
	b := []int{5, 6, 7, 8, 9}

	ll := CreateLinkedList()
	for _, v := range a {
		f := v
		ll.Push(f)
	}

	for _, v := range b {
		f := v
		ll.Pop()
		ll.Push(f)
	}

	for _, v := range b {
		x := ll.Pop()
		if x != v {
			t.Errorf("Bad value; expected %d, got %d", v, x)
		}
	}
}
