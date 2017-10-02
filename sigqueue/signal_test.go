package sigqueue

import (
	"testing"
	"time"
)

func Test_Signal_Create(t *testing.T) {
	s := CreateSignal()
	if s == nil {
		t.Error("Got nil")
	}
	defer s.Close()

	if s.c != 0 {
		t.Errorf("Got non-zero count %d", s.c)
	}
}

func Test_Signal_Ready(t *testing.T) {
	s := CreateSignal()
	defer s.Close()

	c := s.Ready()
	if c == nil {
		t.Error("Got nil ready channel")
	}

	signaled := false
	var x interface{}
	select {
	case x = <-c:
		signaled = true
	case <-time.After(5 * time.Millisecond):
	}

	if signaled {
		t.Errorf("Got ready notification despite no signal, %#v", x)
	}
}

func Test_Signal_Signal(t *testing.T) {
	c := 0
	done := make(chan bool)
	s := CreateSignal()
	defer s.Close()

	go func() {
		for {
			select {
			case <-s.Ready():
				c++
				done <- true
			}
		}
	}()

	time.AfterFunc(5*time.Millisecond, func() {
		s.Signal()
	})

	<-done

	if c != 1 {
		t.Errorf("Incorrect ready notifications: %d", c)
	}
}

func Test_Signal_SignalMany(t *testing.T) {
	c := 0
	done := make(chan bool)
	max := 10
	s := CreateSignal()
	defer s.Close()

	go func() {
		for {
			select {
			case <-s.Ready():
				c++
			}

			if c == max {
				done <- true
			}
		}
	}()

	for i := 0; i < max; i++ {
		time.AfterFunc(2*time.Millisecond, func() {
			s.Signal()
		})
	}

	<-done

	if c != max {
		t.Errorf("Incorrect ready notifications: %d", c)
	}
}
