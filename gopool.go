/*
Package gopool contains tools for goroutine reuse.

*/
package gopool

import (
	"context"
	"errors"
	"log"
	"sync"
)

// WorkFunc is the type of function performed by workers.
type WorkFunc func(ctx context.Context)

// Pool contains logic of goroutine reuse.
type Pool struct {
	workers    int
	queue      int
	sema       chan struct{}
	work       chan WorkFunc
	wg         sync.WaitGroup
	cancel     chan struct{}
	ctx        context.Context
	cancelFunc context.CancelFunc
	mu         sync.RWMutex // guard closed
	closed     bool
}

// New returns an initialized goroutine pool.
//
// The workers parameter determines the size of the pool.
// The queue parameter determines the size of the work queue.
func New(workers, queue int) (*Pool, error) {
	if workers <= 0 {
		return nil, errors.New("size must be greater than 0")
	}

	if queue <= 0 {
		return nil, errors.New("queue must be greater than 0")
	}

	p := &Pool{
		workers: workers,
		queue:   queue,
	}

	p.initialize()

	return p, nil
}

func (p *Pool) initialize() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	p.ctx = ctx
	p.cancelFunc = cancelFunc
	p.sema = make(chan struct{}, p.workers)
	p.work = make(chan WorkFunc, p.queue)
	p.cancel = make(chan struct{})
	p.closed = false
}

// Schedule schedules task to be executed over pool's workers.
func (p *Pool) Schedule(work WorkFunc) error {
	if p.IsClosed() {
		return errors.New("pool closed")
	}

	select {
	case p.work <- work:
		return nil
	case p.sema <- struct{}{}:
		p.worker(work)
		return nil
	}
}

// IsClosed returns if the pool is closed.
func (p *Pool) IsClosed() bool {
	select {
	case <-p.cancel:
		return true
	default:
		return false
	}
}

// Close closes the pool.
func (p *Pool) Close() {
	p.mu.Lock()

	if !p.closed {
		log.Println("[POOL] pool->Close()")
		close(p.cancel)
		p.cancelFunc()
		p.closed = true

		go func(p *Pool) {
			p.wg.Wait()
			close(p.work)
		}(p)
	}

	for task := range p.work {
		task(p.ctx)
	}

	p.mu.Unlock()
}

// Reset resets the closed pool.
func (p *Pool) Reset() {
	p.mu.Lock()
	p.wg.Wait()

	if !p.closed {
		p.mu.Unlock()
		return
	}

	p.initialize()
	p.mu.Unlock()
}

func (p *Pool) worker(work WorkFunc) {
	p.wg.Add(1)

	go func(p *Pool) {
		defer func() {
			log.Println("[POOL] worker done")
			<-p.sema
			p.wg.Done()
		}()

	loop:
		for {
			select {
			case <-p.cancel:
				return
			case task, ok := <-p.work:
				if !ok {
					break loop
				}
				task(p.ctx)
			}
		}
	}(p)

	p.work <- work
}
