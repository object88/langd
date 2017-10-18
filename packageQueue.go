package langd

import (
	"github.com/object88/langd/collections"
)

type packageQueue struct {
	in, out chan *string
	length  chan int
	ll      *collections.LinkedList
}

type packageQueueLinkedListItem struct {
	prev, next collections.LinkedListItem
	Value      string
}

func (pqlli *packageQueueLinkedListItem) AssignSiblings(prev, next collections.LinkedListItem) {
	pqlli.prev, pqlli.next = prev, next
}

func (pqlli *packageQueueLinkedListItem) Siblings() (prev, next collections.LinkedListItem) {
	return pqlli.prev, pqlli.next
}

func createPackageQueue() *packageQueue {
	pq := &packageQueue{
		in:     make(chan *string),
		length: make(chan int),
		out:    make(chan *string),
		ll:     collections.CreateLinkedList(),
	}

	go pq.process()

	return pq
}

func (pq *packageQueue) Close() {
	close(pq.in)
}

func (pq *packageQueue) In() chan<- *string {
	return pq.in
}

func (pq *packageQueue) Len() int {
	return <-pq.length
}

func (pq *packageQueue) Out() <-chan *string {
	return pq.out
}

func (pq *packageQueue) process() {
	in := pq.in
	var out chan *string
	var s *string
	for in != nil || out != nil {
		select {
		case s, ok := <-in:
			if ok {
				pq.ll.Push(&packageQueueLinkedListItem{Value: *s})
			} else {
				in = nil
			}
		case out <- s:
			pq.ll.Pop()
		case pq.length <- pq.ll.Size():
		}

		if pq.ll.Size() > 0 {
			out = pq.out
			s = &(pq.ll.Peek().(*packageQueueLinkedListItem).Value)
		} else {
			out = nil
			s = nil
		}
	}

	close(pq.out)
	close(pq.length)
}
