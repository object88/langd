package collections

// LinkedList is a doubly-linked list of structs which implement LinkedListItem
type LinkedList struct {
	head LinkedListItem
	tail LinkedListItem
	size int
}

// LinkedListItem is an interface which allows a struct to be collected in a
// LinkedList
type LinkedListItem interface {
	Siblings() (LinkedListItem, LinkedListItem)
	AssignSiblings(prev, next LinkedListItem)
}

// CreateLinkedList returns a new linked list struct
func CreateLinkedList() *LinkedList {
	return &LinkedList{
		head: nil,
		tail: nil,
		size: 0,
	}
}

// Peek returns the item at the head of the linked list without popping it off,
// so the linked list is not changed.  If the linked list is empty or nil, it
// returns nil.
func (ll *LinkedList) Peek() LinkedListItem {
	if ll == nil || ll.head == nil {
		return nil
	}

	return ll.head
}

// Peer returns the item at the tail of the linked list without popping it off,
// so the linked list is not changed.  If the linked list is empty or nil, it
// returns nil.
func (ll *LinkedList) Peer() LinkedListItem {
	if ll == nil || ll.tail == nil {
		return nil
	}

	return ll.tail
}

// Push adds the item to the end of the list.  If the pointer receiver is nil,
// a new linked list is returned.  Otherwise, it returns the same linked list.
func (ll *LinkedList) Push(lli LinkedListItem) {
	if ll.tail == nil {
		lli.AssignSiblings(nil, nil)
		ll.head = lli
		ll.tail = lli
	} else if ll.head == ll.tail {
		ll.tail = lli
		ll.head.AssignSiblings(nil, lli)
		ll.tail.AssignSiblings(ll.head, nil)
	} else {
		lliTailPrev, nil := ll.tail.Siblings()
		lli.AssignSiblings(ll.tail, nil)
		ll.tail.AssignSiblings(lliTailPrev, lli)
		ll.tail = lli
	}

	ll.size++
}

// Pop returns the first item in the linked list, or nil.
func (ll *LinkedList) Pop() LinkedListItem {
	if ll.head == nil {
		return nil
	}

	lli := ll.head

	if ll.head == ll.tail {
		ll.head = nil
		ll.tail = nil
	} else {
		_, headNext := ll.head.Siblings()
		ll.head = headNext
		if ll.head != nil {
			_, headNextNext := ll.head.Siblings()
			ll.head.AssignSiblings(nil, headNextNext)
		}
	}

	ll.size--

	return lli
}

// Size returns the number of elements in the linked list
func (ll *LinkedList) Size() int {
	return ll.size
}
