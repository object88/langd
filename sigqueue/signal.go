package sigqueue

// Signal sends on the provided notify channel when a signal has been raised.
// This code is largely based on work done for "infinite channels" by eapache:
// https://github.com/eapache/channels/blob/master/infinite_channel.go
type Signal struct {
	in  chan interface{}
	out chan interface{}
	c   int
}

// CreateSignal creates a new instance of the Signal struct
func CreateSignal() *Signal {
	s := &Signal{
		in:  make(chan interface{}),
		out: make(chan interface{}),
		c:   0,
	}

	go s.process()

	return s
}

// Close will stop the signal processing
func (s *Signal) Close() {
	close(s.in)
}

// Ready provides a channel to listen on for when a signal has been raised
func (s *Signal) Ready() <-chan interface{} {
	return s.out
}

// Signal will raise the signal
func (s *Signal) Signal() {
	var i interface{}
	s.in <- i
}

func (s *Signal) process() {
	// We have a write-only channel here to make sure that we don't write out to
	// any listening consumer when we haven't been signaled.
	var out chan<- interface{}
	var next interface{}
	run := true

	for run {
		select {
		case _, open := <-s.in:
			if !open {
				run = false
			} else {
				s.c++
			}
		case out <- next:
			s.c--
		}

		if s.c > 0 {
			next = true
			out = s.out
		} else {
			next = nil
			out = nil
		}
	}

	close(s.out)
}
