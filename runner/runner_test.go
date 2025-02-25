package runner

import (
	"context"
	"testing"
	"time"
)

var interval = 100 * time.Millisecond

type backgroundTask struct {
	*Runner
	count   int
	working bool
}

func newBackgroundTask() *backgroundTask {
	return &backgroundTask{
		Runner: NewRunner(),
	}
}

func newBackgroundTaskWithContext(c context.Context) *backgroundTask {
	return &backgroundTask{
		Runner: NewRunnerWithContext(c),
	}
}

func (t *backgroundTask) start() {
	t.Runner.Mark()
	go t.running()
}

func (t *backgroundTask) stop() {
	t.Runner.CloseWait()
}

func (t *backgroundTask) running() {
	interval := 100 * time.Millisecond
	timer := time.NewTimer(interval)
	t.working = true
	defer func() {
		timer.Stop()
		t.working = false
		t.Runner.Done()
	}()
	for {
		select {
		case <-t.Runner.Quit():
			return
		case <-timer.C:
			timer.Reset(interval)
		}
	}
}

func (t *backgroundTask) do() {
	t.count++
}

func TestRunner(t *testing.T) {
	task := newBackgroundTask()
	task.start()
	time.Sleep(10 * interval)
	task.stop()
	if task.working {
		t.Fatalf("runner is working after stop called, the expect is not working, count[%d]", task.count)
	}
	if task.count > 10 {
		t.Fatalf("count[%d] is more bigger than expected[%d]", task.count, 10)
	}
}

func TestRunnerWithContext(t *testing.T) {
	c, cancel := context.WithTimeout(context.Background(), 5*interval)
	defer cancel()
	task := newBackgroundTaskWithContext(c)
	task.start()
	time.Sleep(10 * interval)
	if task.working {
		t.Fatalf("runner is working after context timeout, the expect is not working, count[%d]", task.count)
	}
	task.stop()
	if task.count > 5 {
		t.Fatalf("count[%d] is more bigger than expected[%d]", task.count, 5)
	}
}
