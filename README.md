# gopool

The gopool package contains tools to reuse goroutine and limit resource consumption when running a collection of tasks.

## Sample Use

```Go
package main

import (
	"log"
	"time"
	"github.com/tsmweb/gopool"
)

func main() {
	workerSize := 10
	queueSize := 1
	
	pool := gopool.New(workerSize, queueSize)
	defer pool.Close()

	for i := 0; i < 100; i++ {
		task := &PrintTask{
			Index:    i,
			Duration: 10*time.Millisecond,
		}

		err := pool.Schedule(task.Run())
		if err != nil {
			log.Println(err.Error())
			break
		}
	}
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
```