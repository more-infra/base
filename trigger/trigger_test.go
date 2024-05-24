package trigger

import (
	"context"
	"github.com/more-infra/base/queue"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	conditionKey = "total_size"
)

type conditionContext struct {
	count     int
	totalSize int64
}

type entry struct {
	size int64
}

func TestCondition(t *testing.T) {
	var (
		triggerSize   int64
		triggerCount  int64
		expectedSize  int64
		expectedCount int64
	)

	receiver := queue.NewBuffer()

	c := context.WithValue(context.Background(), conditionKey, &conditionContext{})
	tr := NewTrigger(receiver,
		WithMaxCount(10),
		WithMaxTime(1*time.Second),
		WithCondition(c, func(ctx context.Context, event string, ee ...interface{}) int {
			cc := ctx.Value(conditionKey).(*conditionContext)
			switch event {
			case EventConditionScheme:
				entry := ee[0].(*entry)
				cc.count++
				cc.totalSize += entry.size
				if cc.totalSize > 300 {
					count := cc.count
					cc.count = 0
					cc.totalSize = 0
					return count
				}
			case EventCountReached:
				fallthrough
			case EventTimeReached:
				for _, e := range ee {
					entry := e.(*entry)
					cc.totalSize -= entry.size
					cc.count--
				}
			}
			return 0
		}))

	tr.Start()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case v := <-receiver.Channel():
				ee := v.([]interface{})
				if len(ee) == 0 {
					return
				}
				var (
					count     int
					totalSize int64
				)
				for _, e := range ee {
					entry := e.(*entry)
					count++
					totalSize += entry.size
				}
				t.Logf("count:[%d]\tsize:[%d]\n", count, totalSize)
				atomic.AddInt64(&triggerSize, totalSize)
				atomic.AddInt64(&triggerCount, int64(count))
			}
		}
	}()

	for n := 0; n != 100; n++ {
		size := rand.Int63n(int64(n + 1))
		tr.Add(&entry{
			size: size,
		})
		expectedCount++
		expectedSize += size
		time.Sleep(100 * time.Millisecond)
	}
	tr.Stop()
	wg.Wait()
	if expectedCount != triggerCount {
		t.Fatal("count is not expected")
	}
	if expectedSize != triggerSize {
		t.Fatal("size is not expected")
	}
}
