package trigger

import (
	"context"
	"github.com/eapache/queue"
	"github.com/more-infra/base/runner"
	"github.com/more-infra/base/status"
	"sync"
	"time"
)

// Trigger accepts element put, and work as a counter. When max_count, timer or condition reaches, it will do notify by the callback function.
// It's used in log or message collector scenes as usual.
// Logs and messages are put into the Trigger's queue, given a timer or max_count for trigger
// to pack the elements to a batch, and then send them in one request to the backend server.
type Trigger struct {
	statusController *status.Controller
	runner           *runner.Runner
	queue            *queue.Queue
	conf             config
	handler          func([]interface{})
	addCh            chan interface{}
	flush            chan *sync.WaitGroup
}

type Option func(*Trigger)

// NewTrigger create the Trigger, the handler param is the callback for the batch elements which is return by trigger reached.
// Option provides trigger setting, such as max_time, max_count or condition defined by yourself.
// One option would be setting at least, all options could be set together yet.
// All methods of Trigger are thread-safe.
func NewTrigger(handler func([]interface{}), ops ...Option) *Trigger {
	c := &Trigger{
		statusController: status.NewController(),
		runner:           runner.NewRunner(),
		queue:            queue.New(),
		handler:          handler,
		addCh:            make(chan interface{}),
		flush:            make(chan *sync.WaitGroup),
	}
	for _, op := range ops {
		op(c)
	}
	if c.conf.maxCount == 0 && c.conf.maxTime == 0 && c.conf.condition == nil {
		panic("trigger max_count, max_time and condition are all not be set")
	}
	return c
}

// WithMaxTime sets a timer, when it's expired, elements in Trigger will be packed as batch and callback function is called.
func WithMaxTime(t time.Duration) Option {
	return func(tr *Trigger) {
		tr.conf.maxTime = t
	}
}

// WithMaxCount sets the count, when elements in Trigger reach it, the Trigger will pack all elements in it to batch and do callback.
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
func (this *Trigger) Start() {
	if !this.statusController.Starting() {
		return
	}
	defer this.statusController.Started()
	this.runner.Mark()
	go this.running()
}

// Stop is required to called in pair with Start for shutdown the Trigger.
func (this *Trigger) Stop() {
	if !this.statusController.Stopping() {
		return
	}
	defer this.statusController.Stopped()
	this.runner.CloseWait()
	close(this.addCh)
	close(this.flush)
}

// Add put an element into Trigger.
func (this *Trigger) Add(e interface{}) {
	if !this.statusController.KeepRunning() {
		return
	}
	defer this.statusController.ReleaseRunning()
	this.addCh <- e
}

// Flush will do pack all elements in Trigger to batch manually.
func (this *Trigger) Flush() {
	if !this.statusController.KeepRunning() {
		return
	}
	defer this.statusController.ReleaseRunning()
	var wg sync.WaitGroup
	this.flush <- &wg
	wg.Wait()
}

func (this *Trigger) running() {
	var timer *time.Timer
	defer func() {
		timer.Stop()
		var ee []interface{}
		for this.queue.Length() != 0 {
			ee = append(ee, this.queue.Remove())
		}
		if len(ee) != 0 {
			this.handler(ee)
		}
		this.runner.Done()
	}()
	dur := this.conf.maxTime
	timer = time.NewTimer(dur)
	if dur == 0 {
		<-timer.C
	}
	for {
		select {
		case <-this.runner.Quit():
			return
		case e := <-this.addCh:
			this.queue.Add(e)
			if this.schemeCondition(e) != 0 && dur != 0 {
				timer.Reset(dur)
			}
			if this.schemeCount() != 0 && dur != 0 {
				timer.Reset(dur)
			}
		case wg := <-this.flush:
			this.doFlush(wg)
		case <-timer.C:
			this.schemeExpired()
			timer.Reset(dur)
		}
	}
}

func (this *Trigger) schemeExpired() int {
	ee := this.popCount(this.queue.Length())
	if len(ee) != 0 {
		this.runner.Mark()
		go func() {
			this.handler(ee)
			this.runner.Done()
		}()
		this.notifyCondition(EventTimeReached, ee)
	}
	return len(ee)
}

func (this *Trigger) doFlush(wg *sync.WaitGroup) int {
	ee := this.popCount(this.queue.Length())
	if len(ee) != 0 {
		this.runner.Mark()
		wg.Add(1)
		go func() {
			defer wg.Done()
			this.handler(ee)
			this.runner.Done()
		}()
	}
	return len(ee)
}

func (this *Trigger) schemeCount() int {
	if this.conf.maxCount == 0 {
		return 0
	}
	var count int
	for this.queue.Length() >= this.conf.maxCount {
		ee := this.popCount(this.conf.maxCount)
		this.runner.Mark()
		go func() {
			this.handler(ee)
			this.runner.Done()
		}()
		count += len(ee)
		this.notifyCondition(EventCountReached, ee)
	}
	return count
}

func (this *Trigger) schemeCondition(e interface{}) int {
	condition := this.conf.condition
	if condition == nil {
		return 0
	}
	var count int
	n := condition.f(condition.c, EventConditionScheme, e)
	if n != 0 {
		ee := this.popCount(n)
		this.runner.Mark()
		go func() {
			this.handler(ee)
			this.runner.Done()
		}()
		count = len(ee)
	}
	return count
}

func (this *Trigger) notifyCondition(evt string, ee []interface{}) {
	if this.conf.condition != nil {
		this.conf.condition.f(this.conf.condition.c, evt, ee...)
	}
}

func (this *Trigger) popCount(count int) []interface{} {
	ee := make([]interface{}, count, count)
	var n int
	for this.queue.Length() != 0 {
		ee[n] = this.queue.Remove()
		n++
	}
	return ee
}
