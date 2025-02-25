package trigger

import (
	"context"
	workerqueue "github.com/eapache/queue"
	"github.com/more-infra/base/queue"
	"github.com/more-infra/base/runner"
	"github.com/more-infra/base/status"
	"time"
)

// Trigger accepts element put, and work as a counter. When max_count, timer or condition reaches, it will do notify by the callback function.
// It's used in log or message collector scenes as usual.
// Logs and messages are put into the Trigger's queue, given a timer or max_count for trigger
// to pack the elements to a batch, and then send them in one request to the backend server.
type Trigger struct {
	statusController *status.Controller
	runner           *runner.Runner
	queue            *workerqueue.Queue
	conf             config
	addCh            chan interface{}
	flush            chan struct{}
	receiver         *queue.Buffer
}

type Option func(*Trigger)

// NewTrigger create the Trigger, the receiver param is the queue for receiving the batch elements which is return by trigger reached.
// Elements in receiver queue is type '[]interface{}'.
// Option provides trigger setting, such as max_time, max_count or condition defined by yourself.
// One option would be setting at least, all options could be set together yet.
// All methods of Trigger are thread-safe.
func NewTrigger(receiver *queue.Buffer, ops ...Option) *Trigger {
	c := &Trigger{
		statusController: status.NewController(),
		runner:           runner.NewRunner(),
		queue:            workerqueue.New(),
		receiver:         receiver,
		addCh:            make(chan interface{}),
		flush:            make(chan struct{}),
	}
	for _, op := range ops {
		op(c)
	}
	if c.conf.maxCount == 0 && c.conf.maxTime == 0 && c.conf.condition == nil {
		panic("trigger max_count, max_time and condition are all not be set")
	}
	return c
}

// WithMaxTime sets a timer, when it's expired, elements in Trigger will be packed as batch and send elements batch to receiver queue.
func WithMaxTime(t time.Duration) Option {
	return func(tr *Trigger) {
		tr.conf.maxTime = t
	}
}

// WithMaxCount sets the count, when elements in Trigger reach it, the Trigger will pack all elements in it to batch and send elements batch to receiver queue.
func WithMaxCount(n int) Option {
	return func(tr *Trigger) {
		tr.conf.maxCount = n
	}
}

const (
	EventTimeReached     = "event_time_reached"
	EventCountReached    = "event_count_reached"
	EventConditionScheme = "event_condition_scheme"
)

// WithCondition is more flexible than WithMaxTime and WithMaxCount,
// it provides a function to decide the batch count of elements by receiving the event push by Trigger.
// The param f is a function has three param
//
// ctx is the same as WithCondition first param context.Context
//
// event has three value, see more details from the comment above.
//
// elements are batch that packed when the event happened.
//
// The return is an int value, when event is EventTimeReached or EventCountReached, it's ignored,
// only when the event is EventConditionScheme, it means the count of elements need to pack for batch.
func WithCondition(c context.Context, f func(ctx context.Context, event string, elements ...interface{}) int) Option {
	return func(tr *Trigger) {
		tr.conf.condition = &condition{
			c: c,
			f: f,
		}
	}
}

type config struct {
	maxCount  int
	maxTime   time.Duration
	condition *condition
}

type condition struct {
	c context.Context
	f func(ctx context.Context, event string, elements ...interface{}) int
}

// Start is required to called before call Add or Flush.
func (tr *Trigger) Start() {
	if !tr.statusController.Starting() {
		return
	}
	defer tr.statusController.Started()
	tr.runner.Mark()
	go tr.running()
}

// Stop is required to called in pair with Start for shutdown the Trigger.
func (tr *Trigger) Stop() {
	if !tr.statusController.Stopping() {
		return
	}
	defer tr.statusController.Stopped()
	tr.runner.CloseWait()
	close(tr.addCh)
	close(tr.flush)
}

// Add put an element into Trigger.
func (tr *Trigger) Add(e interface{}) {
	if !tr.statusController.KeepRunning() {
		return
	}
	defer tr.statusController.ReleaseRunning()
	tr.addCh <- e
}

// Flush will do pack all elements in Trigger to batch manually.
func (tr *Trigger) Flush() {
	if !tr.statusController.KeepRunning() {
		return
	}
	defer tr.statusController.ReleaseRunning()
	tr.flush <- struct{}{}
}

func (tr *Trigger) running() {
	var timer *time.Timer
	defer func() {
		timer.Stop()
		var ee []interface{}
		for tr.queue.Length() != 0 {
			ee = append(ee, tr.queue.Remove())
		}
		if len(ee) != 0 {
			tr.receiver.Push(ee)
		}
		tr.receiver.Push([]interface{}{})
		tr.runner.Done()
	}()
	dur := tr.conf.maxTime
	timer = time.NewTimer(dur)
	if dur == 0 {
		<-timer.C
	}
	for {
		select {
		case <-tr.runner.Quit():
			return
		case e := <-tr.addCh:
			tr.queue.Add(e)
			if tr.schemeCondition(e) != 0 && dur != 0 {
				timer.Reset(dur)
			}
			if tr.schemeCount() != 0 && dur != 0 {
				timer.Reset(dur)
			}
		case <-tr.flush:
			tr.doFlush()
		case <-timer.C:
			tr.schemeExpired()
			timer.Reset(dur)
		}
	}
}

func (tr *Trigger) schemeExpired() int {
	ee := tr.popCount(tr.queue.Length())
	if len(ee) != 0 {
		tr.receiver.Push(ee)
		tr.notifyCondition(EventTimeReached, ee)
	}
	return len(ee)
}

func (tr *Trigger) doFlush() int {
	ee := tr.popCount(tr.queue.Length())
	if len(ee) != 0 {
		tr.receiver.Push(ee)
	}
	return len(ee)
}

func (tr *Trigger) schemeCount() int {
	if tr.conf.maxCount == 0 {
		return 0
	}
	var count int
	for tr.queue.Length() >= tr.conf.maxCount {
		ee := tr.popCount(tr.conf.maxCount)
		tr.receiver.Push(ee)
		count += len(ee)
		tr.notifyCondition(EventCountReached, ee)
	}
	return count
}

func (tr *Trigger) schemeCondition(e interface{}) int {
	condition := tr.conf.condition
	if condition == nil {
		return 0
	}
	var count int
	n := condition.f(condition.c, EventConditionScheme, e)
	if n != 0 {
		ee := tr.popCount(n)
		tr.receiver.Push(ee)
		count = len(ee)
	}
	return count
}

func (tr *Trigger) notifyCondition(evt string, ee []interface{}) {
	if tr.conf.condition != nil {
		tr.conf.condition.f(tr.conf.condition.c, evt, ee...)
	}
}

func (tr *Trigger) popCount(count int) []interface{} {
	ee := make([]interface{}, count, count)
	var n int
	for tr.queue.Length() != 0 {
		ee[n] = tr.queue.Remove()
		n++
	}
	return ee
}
