package health

import (
	"math"
	"os"
	"sync/atomic"
	"time"

	sigar "github.com/cloudfoundry/gosigar"
)

// Load represents the system resources that the server process is consuming,
// in terms of CPU% and megabytes of memory.
type Load struct {
	currentCPULoad    uint32
	currentMemoryLoad uint32
	done              chan bool
	pid               int
	pm                *sigar.ProcMem
	pc                *sigar.ProcCpu
}

// StartLoadMonitoring begins the load monitoring.  The monitor can be stopped
// by invoking load.Close()
func StartLoadMonitoring() *Load {
	l := &Load{
		done: make(chan bool),
		pid:  os.Getpid(),
		pm:   &sigar.ProcMem{},
		pc:   &sigar.ProcCpu{},
	}

	go func() {
		for {
			select {
			case <-l.done:
				break
			case <-time.After(1 * time.Second):
				l.update()
			}
		}
	}()

	return l
}

// Close will end the load monitoring.
func (l *Load) Close() error {
	l.done <- true
	return nil
}

// CPU reports the instantaneous CPU load.
func (l *Load) CPU() float32 {
	x := atomic.LoadUint32(&l.currentCPULoad)

	x1 := math.Float32frombits(x)
	return x1
}

// Memory reports the instantaneous memory load in megabytes
func (l *Load) Memory() uint32 {
	x := atomic.LoadUint32(&l.currentMemoryLoad)

	return x
}

func (l *Load) update() {
	l.pc.Get(l.pid)
	l.pm.Get(l.pid)

	cpu0 := float32(l.pc.Percent)
	cpu := math.Float32bits(cpu0)

	// Bit shift 20 to the right to divide by 1024*1024.
	res0 := l.pm.Resident
	res := uint32((res0 >> 20) & math.MaxUint32)

	atomic.StoreUint32(&l.currentCPULoad, cpu)
	atomic.StoreUint32(&l.currentMemoryLoad, res)
}
