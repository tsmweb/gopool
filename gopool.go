/*
Package gopool contains tools for goroutine reuse.

*/

package gopool

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
)

// Job type executed on pool workers.
type Job interface {
	Start(ctx context.Context)
}

// Pool contains logic of goroutine reuse.
type Pool struct {
	size int
	sema chan struct{}
	wg   sync.WaitGroup

	queueSize int
	queue     chan Job

	ctx        context.Context
	cancelFunc context.CancelFunc

	close   chan struct{}
	mu      sync.Mutex // guard stopped
	stopped bool
}

// New returns an initialized goroutine pool.
//
// The size parameter determines the size of the pool.
// The queueSize parameter determines the size of the job queue.
func New(size, queueSize int) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("size must be greater than 0")
	}

	if queueSize <= 0 {
		return nil, errors.New("queue size must be greater than 0")
	}

	p := &Pool{
		size:      size,
		queueSize: queueSize,
	}
	p.initialize()

	return p, nil
}

func (p *Pool) initialize() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	p.sema = make(chan struct{}, p.size)
	p.queue = make(chan Job, p.queueSize)
	p.close = make(chan struct{})
	p.ctx = ctx
	p.cancelFunc = cancelFunc
	p.stopped = false
}

// Schedule schedules job to be executed over pool's workers.
func (p *Pool) Schedule(job Job) error {
	if p.IsStopped() {
		return errors.New("pool stopped")
	}

	select {
	case p.queue <- job:
		return nil
	case p.sema <- struct{}{}:
		p.worker(job)
		return nil
	}
}

// IsStopped returns if the pool is stopped.
func (p *Pool) IsStopped() bool {
	select {
	case <-p.close:
		return true
	default:
		return false
	}
}

// Stop stops the workers.
func (p *Pool) Stop() {
	p.mu.Lock()

	if !p.stopped {
		log.Println("[POOL] pool->Stop()")
		close(p.close) // signal to shutdown workers and close the pool.
		p.cancelFunc() // tells an operation to abandon its work.
		p.stopped = true

		go func() {
			p.wg.Wait()
			close(p.queue) // close the queue channel.
		}()

		for job := range p.queue {
			job.Start(p.ctx)
		}
	}

	p.mu.Unlock()
}

func (p *Pool) worker(job Job) {
	p.wg.Add(1)

	go func() {
		defer func() {
			fmt.Println("[POOL] worker done")
			<-p.sema
			p.wg.Done()
		}()

	loop:
		for {
			select {
			case <-p.close:
				return
			case job, ok := <-p.queue:
				if !ok {
					break loop
				}
				job.Start(p.ctx)
			}
		}
	}()

	p.queue <- job
}
