package sigqueue

// LinkedList is an implimentation of a doubly-linked list with minimal
// functionality to support a Sigqueue.
type LinkedList struct {
	head *linkedListItem
	tail *linkedListItem
	size int
}

type linkedListItem struct {
	n    int
	prev *linkedListItem
	next *linkedListItem
}

// CreateLinkedList returns a new LinkedList structure.
func CreateLinkedList() *LinkedList {
	return &LinkedList{nil, nil, 0}
}

// Peek returns the item at the head of the linked list without popping it off,
// so the linked list is not changed.  If the linked list is empty or nil, it
// returns nil.
func (ll *LinkedList) Peek() int {
	if ll == nil || ll.head == nil {
		return -1
	}

	return ll.head.n
}

// Peer returns the item at the tail of the linked list without popping it off,
// so the linked list is not changed.  If the linked list is empty or nil, it
// returns nil.
func (ll *LinkedList) Peer() int {
	if ll == nil || ll.tail == nil {
		return -1
	}

	return ll.tail.n
}

// Push adds the item to the end of the list.  If the pointer receiver is nil,
// a new linked list is returned.  Otherwise, it returns the same linked list.
func (ll *LinkedList) Push(n int) *LinkedList {
	if ll == nil {
		ll = CreateLinkedList()
	}

	lli := &linkedListItem{n, nil, nil}

	if ll.tail == nil {
		ll.head = lli
		ll.tail = lli
	} else if ll.head == ll.tail {
		ll.tail = lli
		ll.head.next = lli
		ll.tail.prev = ll.head
	} else {
		lli.prev = ll.tail
		ll.tail.next = lli
		ll.tail = lli
	}

	ll.size++

	return ll
}

// Pop returns the first item in the linked list, or nil.
func (ll *LinkedList) Pop() int {
	if ll == nil || ll.head == nil {
		return -1
	}

	lli := ll.head

	if ll.head == ll.tail {
		ll.head = nil
		ll.tail = nil
	} else {
		ll.head = ll.head.next
		if ll.head != nil {
			ll.head.prev = nil
		}
	}

	ll.size--

	return lli.n
}

// Size returns the number of elements in the linked list.
func (ll *LinkedList) Size() int {
	if ll == nil {
		return 0
	}
	return ll.size
}
