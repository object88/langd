package langd

import "sync/atomic"

type loadState int32

const (
	queued loadState = iota
	unloaded
	loadedGo
	loadedTest
	done
)

func (ls *loadState) increment() int32 {
	return atomic.AddInt32((*int32)(ls), 1)
}

func (ls *loadState) get() loadState {
	return loadState(atomic.LoadInt32((*int32)(ls)))
}
