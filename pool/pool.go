package pool

import (
	"errors"
	"sync"
)

var (
	ErrPoolStopped   = errors.New("pool is stopped")
	ErrQueueOverflow = errors.New("task queue overflow")
)

type Pool interface {
	Submit(task func()) error
	Stop() error
}

type workerPool struct {
	tasks    chan func()
	wg       sync.WaitGroup
	stopOnce sync.Once
	stopCh   chan struct{}
	stopped  bool
	mu       sync.Mutex
	hook     func()
}

func NewWorkerPool(queueSize, numberOfWorkers int, hook func()) Pool {
	wp := &workerPool{
		tasks:  make(chan func(), queueSize),
		stopCh: make(chan struct{}),
		hook:   hook,
	}

	for i := 0; i < numberOfWorkers; i++ {
		go wp.worker()
	}

	return wp
}

func (wp *workerPool) worker() {
	for {
		select {
		case <-wp.stopCh:
			return
		case task, ok := <-wp.tasks:
			if !ok {
				return
			}
			task()
			if wp.hook != nil {
				wp.hook()
			}
			wp.wg.Done()
		}
	}
}

func (wp *workerPool) Submit(task func()) error {
	wp.mu.Lock()
	if wp.stopped {
		wp.mu.Unlock()
		return ErrPoolStopped
	}
	wp.mu.Unlock()

	select {
	case wp.tasks <- task:
		wp.wg.Add(1)
		return nil
	case <-wp.stopCh:
		return ErrPoolStopped
	default:
		return ErrQueueOverflow
	}
}

func (wp *workerPool) Stop() error {
	wp.stopOnce.Do(func() {
		wp.mu.Lock()
		wp.stopped = true
		wp.mu.Unlock()

		close(wp.tasks)
		wp.wg.Wait()
		close(wp.stopCh)
	})
	return nil
}
