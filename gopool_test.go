package gopool

import (
	"context"
	"testing"
	"time"
)

// go test -timeout 30s -v -run ^TestPool$
func TestPool(t *testing.T) {
	const workers = 10
	const work = 1000

	pool, err := New(workers, 1)
	if err != nil {
		t.Error(err.Error())
	}

	defer pool.Close()

	newFunc := func(id int, d time.Duration) WorkFunc {
		return func(context.Context) {
			t.Log("[WORK] ID: ", id)
			time.Sleep(d)
		}
	}

	for i := 0; i < work; i++ {
		id := i
		err = pool.Schedule(newFunc(id, time.Millisecond*100))
		if err != nil {
			t.Log(err)
			break
		}
	}
}

// go test -timeout 30s -v -run ^TestCancel$
func TestCancel(t *testing.T) {
	const workers = 2
	const work = 10

	pool, err := New(workers, 1)
	if err != nil {
		t.Error(err.Error())
	}

	defer pool.Close()

	newFunc := func(id int, d time.Duration) WorkFunc {
		return func(ctx context.Context) {
			select {
			case <-ctx.Done():
				t.Logf("[WORK] ID: %d canceled", id)
				return
			default:
				t.Log("[WORK] ID: ", id)
				time.Sleep(d)
			}
		}
	}

	tm := time.NewTimer(time.Second * 1)
	go func() {
		<-tm.C
		pool.Close()
	}()

	for i := 0; i < work; i++ {
		id := i
		err := pool.Schedule(newFunc(id, time.Second*1))
		if err != nil {
			t.Log(err)
			break
		}
	}
}

// go test -timeout 30s -v -run ^TestReset$
func TestReset(t *testing.T) {
	//TODO
}
