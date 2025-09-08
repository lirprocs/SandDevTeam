package pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const size = 1024

var hook = func() {}

func TestNewWorkerPoolValidation(t *testing.T) {
	tests := []struct {
		name            string
		queueSize       int
		numberOfWorkers int
		shouldPanic     bool
		panicMessage    string
	}{
		{
			name:            "Valid parameters - normal case",
			queueSize:       10,
			numberOfWorkers: 5,
			shouldPanic:     false,
		},
		{
			name:            "Invalid - zero workers",
			queueSize:       10,
			numberOfWorkers: 0,
			shouldPanic:     true,
			panicMessage:    "numberOfWorkers must be > 0",
		},
		{
			name:            "Invalid - zero queue size",
			queueSize:       0,
			numberOfWorkers: 3,
			shouldPanic:     true,
		},
		{
			name:            "Invalid - negative workers large",
			queueSize:       10,
			numberOfWorkers: -1,
			shouldPanic:     true,
			panicMessage:    "numberOfWorkers must be > 0",
		},
		{
			name:            "Invalid - negative queue size large",
			queueSize:       -1,
			numberOfWorkers: 5,
			shouldPanic:     true,
			panicMessage:    "queueSize must be > 0",
		},
		{
			name:            "Invalid - both parameters negative",
			queueSize:       -5,
			numberOfWorkers: -3,
			shouldPanic:     true,
			panicMessage:    "numberOfWorkers must be > 0",
		},
		{
			name:            "Valid - large parameters",
			queueSize:       1000,
			numberOfWorkers: 100,
			shouldPanic:     false,
		},
		{
			name:            "Valid - minimal valid parameters",
			queueSize:       1,
			numberOfWorkers: 1,
			shouldPanic:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.shouldPanic {
					if r == nil {
						t.Errorf("Expected panic, but got none")
						return
					}
					if tt.panicMessage != "" && r != tt.panicMessage {
						t.Errorf("Expected panic message %q, got %q", tt.panicMessage, r)
					}
				} else {
					if r != nil {
						t.Errorf("Unexpected panic: %v", r)
					}
				}
			}()

			pool := NewWorkerPool(tt.queueSize, tt.numberOfWorkers, hook)

			if !tt.shouldPanic {
				if pool == nil {
					t.Error("Expected pool to be created, got nil")
				}

				pool.Stop()
			}
		})
	}
}

func TestSubmit(t *testing.T) {
	var counter int32
	numTask := 10

	wp := NewWorkerPool(size, 5, hook)

	for i := 0; i < numTask; i++ {
		wp.Submit(func() {
			atomic.AddInt32(&counter, 1)
		})
	}

	err := wp.Stop()
	if counter != int32(numTask) {
		t.Errorf("TestSubmit: tasks did not run")
	}

	if err != nil {
		t.Errorf("TestSubmit: %v", err)
	}
}

func TestParallelExecution(t *testing.T) {
	var counter int32
	numTask := 10

	wp := NewWorkerPool(size, 10, hook)
	start := time.Now()
	for i := 0; i < numTask; i++ {
		wp.Submit(func() {
			atomic.AddInt32(&counter, 1)
			time.Sleep(1000 * time.Millisecond)
		})
	}
	err := wp.Stop()
	stop := time.Since(start)

	if stop > 2000*time.Millisecond {
		t.Errorf("TestParallelExecution: tasks did not run in parallel, took %v", stop)
	}
	if err != nil {
		t.Errorf("TestParallelExecution: %v", err)
	}
}

func TestSubmitAfterStop(t *testing.T) {
	wp := NewWorkerPool(size, 2, hook)
	err := wp.Stop()
	if err != nil {
		t.Errorf("TestSubmitAfterStop: %v", err)
	}
	err = wp.Submit(func() {})
	if err != ErrPoolStopped {
		t.Errorf("TestSubmitAfterStop: %v", err)
	}
}

func TestStop(t *testing.T) {
	var counter int32
	numTask := 5

	wp := NewWorkerPool(size, 2, hook)

	for i := 0; i < numTask; i++ {
		wp.Submit(func() {
			time.Sleep(20 * time.Millisecond)
			atomic.AddInt32(&counter, 1)
		})
	}

	err := wp.Stop()
	if err != nil {
		t.Errorf("TestStop: %v", err)
	}

	if counter != 5 {
		t.Errorf("TestStop: expected %d, got %d", numTask, counter)
	}
}

func TestStopAfterStop(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestStopAfterStop: expected no panic on second Stop, but got: %v", r)
		}
	}()

	wp := NewWorkerPool(size, 2, hook)

	wp.Submit(func() {})

	err := wp.Stop()
	if err != nil {
		t.Errorf("TestStopAfterStop: %v", err)
	}

	err = wp.Stop()
	if err != nil {
		t.Errorf("TestStopAfterStop: %v", err)
	}
}

func TestConcurrentSubmitWait(t *testing.T) {
	var counter int32
	numTasks := 20
	wp := NewWorkerPool(size, 5, hook)

	var wg sync.WaitGroup
	wg.Add(numTasks)

	for i := 0; i < numTasks; i++ {
		go func() {
			defer wg.Done()
			wp.Submit(func() {
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt32(&counter, 1)
			})
		}()
	}

	wg.Wait()
	err := wp.Stop()
	if err != nil {
		t.Errorf("TestConcurrentSubmitWait: %v", err)
	}

	if counter != int32(numTasks) {
		t.Errorf("TestConcurrentSubmitWait: expected %d, got %d", numTasks, counter)
	}
}

func TestStopWithRemainingTasks(t *testing.T) {
	var counter int32
	numTasks := 10

	wp := NewWorkerPool(size, 2, hook)

	for i := 0; i < numTasks; i++ {
		wp.Submit(func() {
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(&counter, 1)
		})
	}

	time.Sleep(150 * time.Millisecond)

	err := wp.Stop()
	if err != nil {
		t.Errorf("TestStopWithRemainingTasks: %v", err)
	}

	if counter != int32(numTasks) {
		t.Errorf("TestStopWithRemainingTasks: expected some tasks to remain unexecuted, got %d/%d", counter, numTasks)
	}
}

func TestQueueOverflow(t *testing.T) {
	wp := NewWorkerPool(1, 1, hook)
	defer wp.Stop()

	if err := wp.Submit(func() { time.Sleep(50 * time.Millisecond) }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := wp.Submit(func() {}); err != ErrQueueOverflow {
		t.Errorf("expected ErrQueueOverflow, got %v", err)
	}
}
