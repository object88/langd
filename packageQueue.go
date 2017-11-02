package langd

// import (
// 	"github.com/object88/langd/collections"
// )

// type tuple struct {
// 	a string
// 	b string
// }

// type directoryQueue struct {
// 	*collections.InfiniteQueue
// }

// func createDirectoryQueue() *directoryQueue {
// 	dq := &directoryQueue{
// 		InfiniteQueue: collections.CreateInfiniteQueue(),
// 	}

// 	return dq
// }

// func (dq *directoryQueue) Close() {
// 	dq.InfiniteQueue.Close()
// }

// func (dq *directoryQueue) In() chan<- *tuple {
// 	return dq.InfiniteQueue.In()
// }

// func (dq *directoryQueue) Len() int {
// 	return <-dq.InfiniteQueue.Len()
// }

// func (dq *directoryQueue) Out() <-chan *tuple {
// 	return dq.out
// }
