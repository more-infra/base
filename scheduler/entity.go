package scheduler

import (
	"context"
	"github.com/more-infra/base/element"
	"sync"
	"time"
)

// Entity is the executor wrapper for Scheduler used.See Scheduler.NewEntity for more details.
// It provides querying execution status and result of the executor.
type Entity struct {
	element       *element.Element
	s             *Scheduler
	executor      Executor
	c             context.Context
	cancel        context.CancelFunc
	delay         time.Duration
	listenCtx     context.Context
	runningCtx    context.Context
	runningCancel context.CancelFunc
	rw            sync.RWMutex
	result        *Result
	timing        *timing
	listener      *listener
}

type EntityOption func(*Entity)

// WithEntityContext defines the execution controller by context.
// When the context is canceled or timeout, the executor could be notified.
func WithEntityContext(c context.Context) EntityOption {
	return func(entity *Entity) {
		entity.listenCtx = c
	}
}

// WithEntityDelay defines the time of delay executing,
// when the Entity pushed to Scheduler, it will be scheduled to wait "delay" time
func WithEntityDelay(delay time.Duration) EntityOption {
	return func(entity *Entity) {
		entity.delay = delay
	}
}

type Status string

const (
	// StatusWaiting means Entity is waiting for schedule.
	StatusWaiting = "waiting"

	// StatusRunning means Entity is running.
	StatusRunning = "running"

	// StatusCanceling means Entity is canceling by Entity context in option or Scheduler is Stopping.
	StatusCanceling = "canceling"

	// StatusDone means Entity has executed done.
	StatusDone = "done"

	// StatusCanceled means Entity has canceled before scheduling for running.
	StatusCanceled = "canceled"

	// StatusAborted means Entity is aborted when running.
	StatusAborted = "aborted"
)

// Result is the information of Entity, it could be acquired when the Entity is running or done.
// See Entity.Result() method for more details.
type Result struct {
	Status    Status
	Err       error
	Waiting   time.Duration
	Executing time.Duration
}

// Done returns the signal chan for executed done or canceled, even aborted.
func (e *Entity) Done() <-chan struct{} {
	return e.c.Done()
}

// Cancel could be called to cancel the executing proc.
// Result() method will acquire a context.Canceled error.
func (e *Entity) Cancel() {
	e.CancelWithError(context.Canceled)
}

// CancelWithError is the same as Cancel but with an error, which could be acquired in Result() method.
func (e *Entity) CancelWithError(err error) {
	e.rw.Lock()
	switch e.result.Status {
	case StatusWaiting:
		e.result.Status = StatusCanceled
		defer func() {
			e.result.Err = err
			e.executor.Abandon()
			e.dispose()
		}()
	case StatusRunning:
		e.result.Status = StatusCanceling
		defer e.runningCancel()
	}
	e.rw.Unlock()
}

// Dispatch will schedule the Entity to Schedule for running, it's the same as Scheduler.Push() method.
func (e *Entity) Dispatch() error {
	return e.s.Push(e)
}

// Result acquires the information of Entity, it could be called when it's running or done.
func (e *Entity) Result() *Result {
	e.rw.RLock()
	result := &Result{
		Status:    e.result.Status,
		Err:       e.result.Err,
		Waiting:   e.result.Waiting,
		Executing: e.result.Executing,
	}
	e.rw.RUnlock()
	return result
}

func (e *Entity) dispatch() {
	e.s.schedule(e)
}

func (e *Entity) onExecute(c context.Context) {
	var (
		exec          bool
		runningCtx    context.Context
		runningCancel context.CancelFunc
	)
	e.rw.Lock()
	if e.result.Status == StatusWaiting {
		runningCtx, runningCancel = context.WithCancel(c)
		defer runningCancel()
		e.runningCtx = runningCtx
		e.runningCancel = runningCancel
		e.timing.run = time.Now()
		e.result.Waiting = e.timing.run.Sub(e.timing.created)
		e.result.Status = StatusRunning
		exec = true
	}
	e.rw.Unlock()
	if !exec {
		return
	}
	var abort bool
	err := e.executor.Do(runningCtx)
	select {
	case <-runningCtx.Done():
		abort = true
	default:
	}
	e.rw.Lock()
	e.result.Err = err
	e.timing.done = time.Now()
	e.result.Executing = e.timing.done.Sub(e.timing.run)
	if abort {
		e.result.Status = StatusAborted
	} else {
		e.result.Status = StatusDone
	}
	e.rw.Unlock()
	e.dispose()
}

func (e *Entity) dispose() {
	e.cancel()
	if e.listener != nil {
		e.listener.remove()
	}
	e.element.Leave()
}

func (e *Entity) Meta() *element.Element {
	return e.element
}

type timing struct {
	created time.Time
	run     time.Time
	done    time.Time
}
