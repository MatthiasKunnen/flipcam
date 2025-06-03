package chanwg

import "sync"

// WaitGroup is a single-use synchronization primitive similar to [sync.WaitGroup].
// Instead of a blocking Wait method, it exposes a channel that closes when all tracked
// operations have completed.
//
// If a normal [sync.WaitGroup] never completes, waiting on it blocks the goroutine it is waiting
// in indefinitely. This allows abandoning a wait.
//
// WaitGroup requires at least one Add and corresponding Done before completing.
//
// WaitGroup must be created with New.
type WaitGroup struct {
	counter int32
	closed  bool
	mu      sync.Mutex
	done    chan struct{}
}

func New() *WaitGroup {
	return &WaitGroup{
		done: make(chan struct{}),
	}
}

// Add increments the counter by the given positive delta.
// Add must be called before the corresponding operations begin execution.
//
// Add may not be called after the wait channel has completed.
func (cwg *WaitGroup) Add(delta int32) {
	if delta == 0 {
		return
	}

	cwg.mu.Lock()
	defer cwg.mu.Unlock()
	if cwg.closed {
		panic("chanwg: WaitGroup already closed")
	}

	cwg.counter += delta

	switch {
	case cwg.counter < 0:
		panic("chanwg: negative WaitGroup counter, too many Done calls")
	case cwg.counter == 0:
		cwg.closed = true
		close(cwg.done)
	}
}

// Done decrements the counter by one.
// When the counter reaches zero, the internal done channel is closed.
//
// Panics if:
//   - Done is called more times than Add
//   - the group has already been closed
func (cwg *WaitGroup) Done() {
	cwg.Add(-1)
}

// WaitChan returns a channel that will be closed when all tracked operation are complete.
func (cwg *WaitGroup) WaitChan() <-chan struct{} {
	return cwg.done
}
