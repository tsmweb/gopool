/*
Package gopool contains tools for reusing goroutine and for
limiting resource consumption when running a collection of tasks.

Executor example:

	workerSize := 10
	queueSize := 1

	pool := gopool.New(workerSize, queueSize)
	defer pool.Shutdown()

	for i := 0; i < 100; i++ {
		idx := i
		err := pool.Schedule(func(ctx context.Context) {
			select {
			case <-ctx.Done():
				log.Printf("[TASK] ID %d - stop \n", idx)
				return
			default:
				log.Printf("[TASK] ID %d \n", idx)
				time.Sleep(time.Millisecond * 100)
		})
		if err != nil {
			t.Log(err)
			break
		}
	}
*/
package gopool

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var (
	ErrClosedPool      = fmt.Errorf("closed pool")
	ErrScheduleTimeout = fmt.Errorf("schedule error: timed out")
)

// Pool contains logic for goroutine reuse.
type Pool struct {
	sema chan struct{}
	work chan func(ctx context.Context)
	wg   sync.WaitGroup

	shutdown   chan struct{}
	mu         sync.Mutex // guard terminated
	terminated bool

	ctx        context.Context
	cancelFunc context.CancelFunc
}

// New creates new goroutine pool with given size. It also creates a work
// queue of given size.
func New(size, queue int) *Pool {
	if size <= 0 && queue > 0 {
		panic("dead configuration detected")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	return &Pool{
		sema:       make(chan struct{}, size),
		work:       make(chan func(ctx context.Context), queue),
		shutdown:   make(chan struct{}),
		terminated: false,
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}
}

// Schedule schedules task to be executed over pool's workers.
// It returns ErrClosedPool when pool is closed.
func (p *Pool) Schedule(task func(ctx context.Context)) error {
	return p.schedule(task, nil)
}

// ScheduleTimeout schedules task to be executed over pool's workers.
// It returns ErrClosedPool when pool is closed.
// It returns ErrScheduleTimeout when no free workers met during given timeout.
func (p *Pool) ScheduleTimeout(timeout time.Duration, task func(ctx context.Context)) error {
	return p.schedule(task, time.After(timeout))
}

// Shutdown close the pool.
func (p *Pool) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.terminated {
		close(p.shutdown) // signal to shutdown workers and close the pool.
		p.cancelFunc()    // tells an operation to abandon its work.
		p.terminated = true
		close(p.work)
		p.wg.Wait()
	}
}

func (p *Pool) schedule(task func(ctx context.Context), timeout <-chan time.Time) error {
	select {
	case <-p.shutdown:
		return ErrClosedPool
	case <-timeout:
		return ErrScheduleTimeout
	case p.work <- task:
		return nil
	case p.sema <- struct{}{}:
		p.wg.Add(1)
		go p.worker(task)
		return nil
	}
}

func (p *Pool) worker(task func(ctx context.Context)) {
	defer func() {
		<-p.sema
		p.wg.Done()
	}()

	task(p.ctx)

	for task := range p.work {
		task(p.ctx)
	}
}
