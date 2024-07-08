package scheduler

import (
	"context"
	"errors"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"
)

var (
	count = 1000
	pool  = 10
)

func TestScheduler(t *testing.T) {
	sc := NewScheduler(WithPoolSize(pool))
	sc.Start()
	defer sc.Stop()

	var ee []*Entity

	actual := &testSchedulerStatistic{}
	for i := 0; i != count; i++ {
		e := sc.NewEntity(&testSchedulerExecutor{
			statistics: actual,
			t:          t,
		})
		err := sc.Push(e)
		if err != nil {
			t.Error(err)
			return
		}
		ee = append(ee, e)
	}

	go func() {
		for {
			n := rand.Intn(count)
			ee[n].Cancel()
			time.Sleep(10 * time.Millisecond)
			if n == count-1 {
				sc.Stop()
			}
		}
	}()

	expect := &testSchedulerStatistic{}
	for _, e := range ee {
		<-e.Done()
		switch e.Result().Status {
		case StatusAborted:
			if e.Result().Err != nil {
				expect.aborted++
			} else {
				expect.done++
			}
		case StatusDone:
			expect.done++
		case StatusCanceled:
			expect.canceled++
		default:
			panic("invalid status")
		}
	}

	if actual.done != expect.done {
		t.Fatal("done is unexpected")
	}
	if actual.aborted != expect.aborted {
		t.Fatal("aborted is unexpected")
	}
	if actual.canceled != expect.canceled {
		t.Fatal("canceled is unexpected")
	}
}

type testSchedulerStatistic struct {
	aborted  int32
	done     int32
	canceled int32
}

type testSchedulerExecutor struct {
	t          *testing.T
	statistics *testSchedulerStatistic
}

func (se *testSchedulerExecutor) Do(c context.Context) error {
	timer := time.NewTimer(10 * time.Millisecond)
	select {
	case <-c.Done():
		atomic.AddInt32(&se.statistics.aborted, 1)
		return errors.New("canceled")
	case <-timer.C:
		atomic.AddInt32(&se.statistics.done, 1)
	}
	return nil
}

func (se *testSchedulerExecutor) Abandon() {
	atomic.AddInt32(&se.statistics.canceled, 1)
}

func TestWithContext(t *testing.T) {
	sc := NewScheduler(WithPoolSize(pool))
	sc.Start()
	defer sc.Stop()

	entityContext, entityCancel := context.WithCancel(context.Background())
	finalContext, finalCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer finalCancel()
	var (
		entityContextDone bool
		errEntityContext  = errors.New("entity context canceled")
		errFinalContext   = errors.New("final context canceled")
	)
	e := sc.NewEntity(&ExecutorWrapper{
		DoFunc: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				entityContextDone = true
				return errEntityContext
			case <-finalContext.Done():
				return errFinalContext
			}
		},
		AbandonFunc: func() {},
	}, WithEntityContext(entityContext))
	_ = e.Dispatch()

	entityCancel()

	select {
	case <-e.Done():
	}
	if !entityContextDone {
		t.Fatal("entity context not effect")
	}
	if e.Result().Err != errEntityContext {
		t.Fatal("entity result Err != entity context canceled")
	}
}

func TestDelay(t *testing.T) {
	sc := NewScheduler(WithPoolSize(pool))
	sc.Start()
	defer sc.Stop()

	delay := 5 * time.Second
	var tmRun time.Time
	e := sc.NewEntity(&ExecutorWrapper{
		DoFunc: func(c context.Context) error {
			tmRun = time.Now()
			return nil
		},
		AbandonFunc: func() {},
	}, WithEntityDelay(delay))

	tm := time.Now()
	_ = e.Dispatch()

	select {
	case <-e.Done():
	}

	if tmRun.Sub(tm) < delay {
		t.Fatal("delay not effect")
	}
}

func TestBenchmark(t *testing.T) {
	sc := NewScheduler()
	sc.Start()
	count := 1000000
	var ee []*Entity
	tm := time.Now()
	for i := 0; i != count; i++ {
		e := sc.NewEntity(&ExecutorWrapper{
			DoFunc: func(c context.Context) error {
				return nil
			},
			AbandonFunc: func() {},
		})
		ee = append(ee, e)
		_ = sc.Push(e)
	}
	for _, e := range ee {
		select {
		case <-e.Done():
		}
	}
	t.Logf("benchmark test %d Entity, time consume: %s", count, time.Since(tm).String())
}

func TestQueue(t *testing.T) {
	sc := NewScheduler(WithPoolSize(1))
	sc.Start()
	count := 100
	interval := 10 * time.Millisecond
	var ee []*Entity
	for i := 0; i != 100; i++ {
		e := sc.NewEntity(&ExecutorWrapper{
			DoFunc: func(context.Context) error {
				time.Sleep(interval)
				return nil
			},
		})
		_ = sc.Push(e)
		ee = append(ee, e)
	}
	tm := time.Now()
	for _, e := range ee {
		select {
		case <-e.Done():
		}
	}
	consume := time.Since(tm)
	expected := time.Duration(count) * interval
	if consume < expected {
		t.Fatalf("single pool execute Entity one by one time consume[%s] is not expected[%s]",
			consume.String(), expected.String())
	}
}

func TestGraceShutdown(t *testing.T) {
	sc := NewScheduler(WithPoolSize(1))
	sc.Start()
	count := 1000
	var (
		abandon int32
		do      int32
	)
	for i := 0; i != count; i++ {
		_ = sc.Push(sc.NewEntity(&ExecutorWrapper{
			DoFunc: func(c context.Context) error {
				select {
				case <-c.Done():
				}
				atomic.AddInt32(&do, 1)
				return c.Err()
			},
			AbandonFunc: func() {
				atomic.AddInt32(&abandon, 1)
			},
		}))
	}
	time.Sleep(100 * time.Millisecond)
	sc.Stop()
	if int(do+abandon) != count {
		t.Fatalf("do[%d] + abandon[%d] count is not expected[%d]", do, abandon, count)
	}
}

func TestGrowAndReduce(t *testing.T) {
	sc := NewScheduler(WithPoolSize(1000), WithPoolReduceDuration(5*time.Second))
	sc.Start()
	for i := 0; i != 1000; i++ {
		_ = sc.Push(sc.NewEntity(&ExecutorWrapper{
			DoFunc: func(c context.Context) error {
				return nil
			},
		}))
	}
	for i := 0; i != 10; i++ {
		time.Sleep(1 * time.Second)
		t.Logf("entity count: %d", sc.workerMgr.queue.Size())
		t.Logf("goroutine pool size: %d", sc.workerMgr.workers.Count())
	}
}
