package queue

import (
	"sync"
	"testing"
	"time"
)

func TestElementOrder(t *testing.T) {
	const (
		num        = 100
		num2       = 100000
		chCapacity = 10
	)
	var (
		chCount  int
		bufCount int
		wg       sync.WaitGroup
	)
	q := NewBuffer(WithChannelCapacity(chCapacity))
	for i := 0; i != num; i++ {
		switch q.Push(i) {
		case PushToChan:
			chCount++
		case PushToQueue:
			bufCount++
		case PushDropped:
			t.Fatalf("unexcepted PushDropped return")
		}
	}

	if chCount != chCapacity {
		t.Fatalf("push to chan num[%d] is not expected[%d]", chCount, chCapacity)
	}

	if bufCount != num-chCapacity {
		t.Fatalf("push to queue num[%d] is not expected[%d]", bufCount, num-chCapacity)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		last := -1
		for i := 0; i != num+num2; i++ {
			select {
			case v := <-q.Channel():
				n := v.(int)
				if last != -1 && n != last+1 {
					t.Errorf("element is not in order, element:%d, expected:%d", n, last+1)
				}
				last = n
			}
		}
	}()
	for i := 0; i != num2; i++ {
		q.Push(num + i)
	}

	wg.Wait()

	q.Dispose()

	if q.Push(1) != PushDropped {
		t.Errorf("pushing to disposed Buffer does not return PushDropped as expected")
	}
}

func TestPolicy(t *testing.T) {
	const (
		chanCapacity  = 10
		queueCapacity = 100
		num           = 200
	)
	q := NewBuffer(
		WithChannelCapacity(chanCapacity),
		WithBufferingIdleTime(1*time.Second),
		WithQueueCapacity(queueCapacity),
		WithQueuePolicy(PolicyRemove))
	defer q.Dispose()

	for i := 0; i != num; i++ {
		ret := q.Push(i)
		if i < chanCapacity {
			if ret != PushToChan {
				t.Fatalf("pushing return value[%s] is not expected[%s]", ret, PushToChan)
			}
		} else if i >= chanCapacity && i < chanCapacity+queueCapacity {
			if ret != PushToQueue {
				t.Fatalf("pushing return value[%s] is not expected[%s]", ret, PushToQueue)
			}
		} else {
			if ret != PushToQueueReplace {
				t.Fatalf("pushing return value[%s] is not expected[%s]", ret, PushToQueueReplace)
			}
		}
	}
}
