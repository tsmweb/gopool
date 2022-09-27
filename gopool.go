/*
Package gopool contains tools for reusing goroutine and for
limiting resource consumption when running a collection of tasks.

Executor example:

	workerSize := 10
	queueSize := 1

	ctx, stop := context.WithCancel(context.Background())
	pool := gopool.New(ctx, workerSize, queueSize)

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

	stop()
	pool.Wait()
	...
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
	ctx  context.Context
	sema chan struct{}
	work chan func(ctx context.Context)
	wg   sync.WaitGroup
}

// New creates new goroutine pool with given size. It also creates a work
// queue of given size.
func New(ctx context.Context, size, queue int) *Pool {
	if size <= 0 && queue > 0 {
		panic("dead configuration detected")
	}

	return &Pool{
		sema: make(chan struct{}, size),
		work: make(chan func(ctx context.Context), queue),
		ctx:  ctx,
	}
}

// Wait until all workers are terminated.
func (p *Pool) Wait() {
	p.wg.Wait()
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

func (p *Pool) schedule(task func(ctx context.Context), timeout <-chan time.Time) error {
	select {
	case <-p.ctx.Done():
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

	for {
		select {
		case <-p.ctx.Done():
			return
		case task := <-p.work:
			task(p.ctx)
		}
	}
}
