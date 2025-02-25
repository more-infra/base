package observer

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/more-infra/base/event"
	"github.com/more-infra/base/queue"
)

func TestObserver(t *testing.T) {
	mgr := NewManager(
		WithObserverOption(WithNotifyChannelCapacity(10)),
		WithQueueBufferOption(queue.WithQueueCapacity(256)),
	)
	defer mgr.Dispose()
	ob := mgr.Add()
	defer ob.Close()
	go func ()  {
		for i := 0; i < 1024; i++ {
			i := i
			mgr.Push(event.NewEvent(fmt.Sprintf("%d", i)).WithContent(i))
		}
	}()

	var received int
	for i := 0; i < 1024; i++ {
		var evt *event.Event
		select {case evt = <-ob.Notify():}
		c := evt.Category()
		n, err := strconv.Atoi(c)
		if err != nil {
			t.Fatalf("category %s, expected %d", c, received)
		}
		if n != received {
			t.Fatalf("category %s, expected %d", c, received)
		}
		received++
	}
	if received != 1024 {
		t.Fatalf("received %d, expected 1024", received)
	}
}
