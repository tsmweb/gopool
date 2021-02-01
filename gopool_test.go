package gopool

import (
	"context"
	"log"
	"testing"
	"time"
)

// go test -v -run ^TestPool$
func TestPool(t *testing.T) {
	const workers = 10
	const jobSize = 1000

	pool, err := New(workers, 1)
	if err != nil {
		t.Error(err.Error())
	}

	defer pool.Stop()

	for i := 0; i < jobSize; i++ {
		job := &PrintJob{
			Index:    i,
			Duration: time.Millisecond * 100,
		}

		err = pool.Schedule(job)
		if err != nil {
			t.Log(err)
			break
		}
	}

	time.Sleep(time.Second * 1)
}

// go test -timeout 5s -v -run ^TestStop$
func TestStop(t *testing.T) {
	const workers = 2
	const jobSize = 10

	pool, err := New(workers, 1)
	if err != nil {
		t.Error(err.Error())
	}

	defer pool.Stop()

	tm := time.NewTimer(time.Second * 1)
	go func() {
		<-tm.C
		pool.Stop()
	}()

	for i := 0; i < jobSize; i++ {
		job := &PrintJob{
			Index:    i,
			Duration: time.Millisecond * 500,
		}
		err := pool.Schedule(job)
		if err != nil {
			t.Log(err)
			break
		}
	}
}

type PrintJob struct {
	Index    int
	Duration time.Duration
}

func (p *PrintJob) Start(ctx context.Context) {
	select {
	case <-ctx.Done():
		log.Printf("[JOB] ID %d - stop \n", p.Index)
		return
	default:
		log.Printf("[JOB] ID %d \n", p.Index)
		time.Sleep(p.Duration)
	}
}
