package gopool

import (
	"context"
	"log"
	"testing"
	"time"
)

func TestPool_Schedule(t *testing.T) {
	workerSize := 4
	queueSize := 1

	ctx, stop := context.WithCancel(context.Background())
	pool := New(ctx, workerSize, queueSize)

	for i := 0; i < 100; i++ {
		task := &PrintTask{
			Index: i,
		}

		err := pool.Schedule(task.Run())
		if err != nil {
			t.Log(err)
			break
		}
	}

	stop()
	pool.Wait()
}

func TestPool_ScheduleTimeout(t *testing.T) {
	workerSize := 4
	queueSize := 1

	ctx, stop := context.WithCancel(context.Background())
	pool := New(ctx, workerSize, queueSize)

	for i := 0; i < 100; i++ {
		task := &PrintTask{
			Index:    i,
			Duration: time.Millisecond,
		}

		err := pool.ScheduleTimeout(time.Millisecond, task.Run())
		if err != nil {
			if err == ErrScheduleTimeout {
				goto cooldown
			}

			t.Log(err)
			break

		cooldown:
			delay := 5 * time.Millisecond
			log.Printf("error: timeout; retrying in %s", delay)
			time.Sleep(delay)
		}
	}

	stop()
	pool.Wait()
}

type PrintTask struct {
	Index    int
	Duration time.Duration
}

func (p *PrintTask) Run() func(ctx context.Context) {
	return func(ctx context.Context) {
		select {
		case <-ctx.Done():
			log.Printf("[TASK] ID %d - stop \n", p.Index)
			return
		default:
			log.Printf("[TASK] ID %d \n", p.Index)
			time.Sleep(p.Duration)
		}
	}
}
