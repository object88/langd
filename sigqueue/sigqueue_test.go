package sigqueue

import (
	"math/rand"
	"testing"
	"time"
)

func Test_Sigqueue_Create(t *testing.T) {
	output := make(chan int)
	sq := CreateSigqueue(output)
	if sq == nil {
		t.Error("Got nil")
	}
	if sq.readied == nil {
		t.Error("Failed to create internal heap")
	}
	if sq.waiters == nil {
		t.Error("Failed to create internal linked list")
	}
	if sq.notify == nil {
		t.Error("Failed to assign notify channel")
	}
}

func Test_Sigqueue_OutOfOrder(t *testing.T) {
	output := make(chan int)
	sq := CreateSigqueue(output)

	q0 := 100
	q1 := 101

	err0 := sq.WaitOn(q1)
	if err0 != nil {
		t.Errorf("Unexpected error from first WaitOn: %s", err0.Error())
	}
	err1 := sq.WaitOn(q0)
	if err1 == nil {
		t.Errorf("Did not get error from out-of-order wait")
	}
	if _, ok := err1.(*ErrOutOfOrderWait); !ok {
		t.Errorf("Got unexpected error %s", err1.Error())
	}
}

func Test_Sigqueue_Wait(t *testing.T) {
	var ider int
	done := make(chan bool)
	output := make(chan int)
	sq := CreateSigqueue(output)

	q0 := 100

	go func() {
		ider = <-output
		done <- true
	}()

	sq.WaitOn(q0)
	sq.Ready(q0)

	<-done

	// Validate that the correct ID has been recieved.
	if ider != 100 {
		t.Error("Incorrect ID")
	}
}

func Test_Sigqueue_WaitBlocked(t *testing.T) {
	done := make(chan bool)
	output := make(chan int)
	sq := CreateSigqueue(output)

	q0 := 100
	q1 := 101

	go func() {
		<-output
		done <- true
	}()

	sq.WaitOn(q0)
	sq.WaitOn(q1)
	sq.Ready(q1)

	time.AfterFunc(50*time.Millisecond, func() { done <- false })

	completed := <-done

	if completed {
		t.Error("Unexpected completion in 50ms")
	}
}

func Test_Sigqueue_WaitUnblocked(t *testing.T) {
	completedIDs := []int{}
	done := make(chan bool)
	output := make(chan int)
	sq := CreateSigqueue(output)

	q0 := 100
	q1 := 101

	go func() {
		for {
			select {
			case item := <-output:
				completedIDs = append(completedIDs, item)
			}

			if len(completedIDs) == 2 {
				done <- true
				return
			}
		}
	}()

	sq.WaitOn(q0)
	sq.WaitOn(q1)
	sq.Ready(q1)
	sq.Ready(q0)

	<-done

	if completedIDs[0] != 100 {
		t.Error("First completed ID is not 100")
	}
	if completedIDs[1] != 101 {
		t.Error("Second completed ID is not 101")
	}
}

func Test_Sigqueue_Large(t *testing.T) {
	completed := 0
	done := make(chan bool)
	output := make(chan int)
	sq := CreateSigqueue(output)

	current := 100

	go func() {
		for {
			select {
			case item := <-output:
				if item != current {
					t.Errorf("Got out of order item: %d", item)
				}
				current++
				completed++
			}

			if completed == 100 {
				done <- true
			}
		}
	}()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 100; i++ {
		q := i + 100
		sq.WaitOn(q)
		t := time.Duration(r.Intn(5))
		time.AfterFunc(t*time.Millisecond, func() { sq.Ready(q) })
	}

	<-done

	if completed != 100 {
		t.Errorf("Did not complete 100; got %d\n", completed)
	}
}
