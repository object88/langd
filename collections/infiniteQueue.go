package collections

type InfiniteQueue struct {
	in, out chan interface{}
	length  chan int
	ll      *LinkedList
}

type infiniteQueueLinkedListItem struct {
	prev, next LinkedListItem
	Value      interface{}
}

func (iqlli *infiniteQueueLinkedListItem) AssignSiblings(prev, next LinkedListItem) {
	iqlli.prev, iqlli.next = prev, next
}

func (iqlli *infiniteQueueLinkedListItem) Siblings() (prev, next LinkedListItem) {
	return iqlli.prev, iqlli.next
}

func CreateInfiniteQueue() *InfiniteQueue {
	pq := &InfiniteQueue{
		in:     make(chan interface{}),
		length: make(chan int),
		out:    make(chan interface{}),
		ll:     CreateLinkedList(),
	}

	go pq.process()

	return pq
}

func (pq *InfiniteQueue) Close() {
	close(pq.in)
}

func (pq *InfiniteQueue) In() chan<- interface{} {
	return pq.in
}

func (pq *InfiniteQueue) Len() int {
	return <-pq.length
}

func (pq *InfiniteQueue) Out() <-chan interface{} {
	return pq.out
}

func (pq *InfiniteQueue) process() {
	in := pq.in
	var out chan interface{}
	var v interface{}
	for in != nil || out != nil {
		select {
		case v, ok := <-in:
			if ok {
				pq.ll.Push(&infiniteQueueLinkedListItem{Value: v})
			} else {
				in = nil
			}
		case out <- v:
			pq.ll.Pop()
		case pq.length <- pq.ll.Size():
		}

		if pq.ll.Size() > 0 {
			out = pq.out
			v = pq.ll.Peek().(*infiniteQueueLinkedListItem).Value
		} else {
			out = nil
			v = nil
		}
	}

	close(pq.out)
	close(pq.length)
}
