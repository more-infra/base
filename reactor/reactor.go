package reactor

import (
	"context"
	"errors"
	"github.com/more-infra/base"
	"github.com/more-infra/base/queue"
	"github.com/more-infra/base/runner"
	"github.com/more-infra/base/status"
	"sync"
)

const (
	ErrTypeHandlerCanceled = "reactor.handle_canceled"
)

var (
	ErrHandlerCanceled = errors.New("reactor handle not run for canceling, the Reactor has been stopped")
)

// Reactor provides a "reactor" design mode for function called.
// It's similar to event loop, each function submit to the Reactor is an event, and Reactor calls the function as event process.
// If you need to process many functions calls having concurrent or sync lock scenes, Reactor will help you make each function call into an
// ordering queue and calls them one by one, which would not required to consider the sync lock or concurrent question.
// It's like a goroutine pool with single goroutine.See example for more usages and details.
type Reactor struct {
	runner           *runner.Runner
	statusController *status.Controller
	c                context.Context
	cancel           context.CancelFunc
	queue            *queue.Buffer
	priority         *queue.Buffer
}

func NewReactor(options ...Option) *Reactor {
	r := &Reactor{
		runner:           runner.NewRunner(),
		statusController: status.NewController(),
		queue:            queue.NewBuffer(),
		priority:         queue.NewBuffer(),
	}
	for _, op := range options {
		op(r)
	}
	ctx := context.Background()
	if r.c != nil {
		ctx = r.c
	}
	c, cancel := context.WithCancel(ctx)
	r.c = c
	r.cancel = cancel
	return r
}

type Option func(*Reactor)

// WithContext defines a context for controlling the Reactor in another way except Start/Stop method.
func WithContext(c context.Context) Option {
	return func(r *Reactor) {
		r.c = c
	}
}

// Start is required to call before Push or Send Handler to the Reactor.
// It will be called with Stop in pair.
func (r *Reactor) Start() {
	if !r.statusController.Starting() {
		return
	}
	defer r.statusController.Started()
	r.runner.Mark()
	go r.running()
}

// Stop is called for shutdown the Reactor.
// The Handlers which are not will return an ErrHandlerCanceled error typed with ErrTypeHandlerCanceled.
// When Stop returned, every Handler Push or Send to the Reactor will be run completed or canceled.
func (r *Reactor) Stop() {
	if !r.statusController.Stopping() {
		return
	}
	defer r.statusController.Stopped()
	r.cancel()
	r.runner.CloseWait()
	for _, v := range r.priority.Dispose() {
		task := v.(*reactorTask)
		task.cancel(base.NewErrorWithType(ErrTypeHandlerCanceled, ErrHandlerCanceled).
			WithFields(task.KV()))
	}
	for _, v := range r.queue.Dispose() {
		task := v.(*reactorTask)
		task.cancel(base.NewErrorWithType(ErrTypeHandlerCanceled, ErrHandlerCanceled).
			WithFields(task.KV()))
	}
}

type Handler func(context.Context)

// Push will insert the handler to Reactor's queue and return immediately.
// If the Reactor has benn stopped, it will return ErrInvalidStatus error with typed ErrTypeInvalidStatus.
func (r *Reactor) Push(handler Handler) error {
	if !r.statusController.KeepRunning() {
		return base.NewErrorWithType(status.ErrTypeInvalidStatus, status.ErrInvalidStatus).
			WithField("handler", handler).
			WithStack()
	}
	defer r.statusController.ReleaseRunning()
	r.queue.Push(r.newReactorTask(handler))
	return nil
}

// PushPriority is the same as Push, but the Handler is higher priority than Push.
func (r *Reactor) PushPriority(handler Handler) error {
	if !r.statusController.KeepRunning() {
		return base.NewErrorWithType(status.ErrTypeInvalidStatus, status.ErrInvalidStatus).
			WithField("handler", handler).
			WithStack()
	}
	defer r.statusController.ReleaseRunning()
	r.priority.Push(r.newReactorTask(handler))
	return nil
}

// Send will insert the handler to Reactor's queue and wait for the Handler run completed.
// If the Reactor has benn stopped, it will return ErrInvalidStatus error with typed ErrTypeInvalidStatus.
// If the Handler inserted to the queue and waiting for run, but the Reactor is Stop,
// it will return ErrHandlerCanceled error with ErrTypeHandlerCanceled.
func (r *Reactor) Send(handler Handler) error {
	if !r.statusController.KeepRunning() {
		return base.NewErrorWithType(status.ErrTypeInvalidStatus, status.ErrInvalidStatus).
			WithField("handler", handler).
			WithStack()
	}
	task := r.newReactorTask(handler)
	r.queue.Push(task)
	r.statusController.ReleaseRunning()
	task.wait()
	if err := task.err(); err != nil {
		return err
	}
	return nil
}

// SendPriority is the same as Send, but the Handler is higher priority than Send.
func (r *Reactor) SendPriority(handler Handler) error {
	if !r.statusController.KeepRunning() {
		return base.NewErrorWithType(status.ErrTypeInvalidStatus, status.ErrInvalidStatus).
			WithField("handler", handler).
			WithStack()
	}
	task := r.newReactorTask(handler)
	r.priority.Push(task)
	r.statusController.ReleaseRunning()
	task.wait()
	if err := task.err(); err != nil {
		return err
	}
	return nil
}

// Waiting return the count of Handlers which are in the queue and waiting for run.
func (r *Reactor) Waiting() int {
	return r.queue.Size() + r.priority.Size()
}

func (r *Reactor) newReactorTask(handler Handler) *reactorTask {
	task := &reactorTask{
		handler: handler,
		ctx:     r.c,
	}
	task.wg.Add(1)
	return task
}

func (r *Reactor) running() {
	var (
		chQueue         = r.queue.Channel()
		chPriorityQueue = r.priority.Channel()
	)
	defer r.runner.Done()
	for {
		select {
		case <-r.runner.Quit():
			return
		case <-r.c.Done():
			go r.Stop()
			return
		case v := <-chPriorityQueue:
			task := v.(*reactorTask)
			task.run()
			if len(chPriorityQueue) != 0 {
				chQueue = nil
			} else {
				chQueue = r.queue.Channel()
			}
		case v := <-chQueue:
			task := v.(*reactorTask)
			task.run()
			if len(chPriorityQueue) != 0 {
				chQueue = nil
			} else {
				chQueue = r.queue.Channel()
			}
		}
	}
}

type reactorTask struct {
	handler   Handler
	ctx       context.Context
	wg        sync.WaitGroup
	errCancel error
}

func (t *reactorTask) run() {
	t.handler(t.ctx)
	t.wg.Done()
}

func (t *reactorTask) wait() {
	t.wg.Wait()
}

func (t *reactorTask) cancel(err error) {
	t.errCancel = err
	t.wg.Done()
}

func (r *reactorTask) err() error {
	return r.errCancel
}

func (r *reactorTask) KV() map[string]interface{} {
	return map[string]interface{}{
		"reactor_task.handler": r.handler,
	}
}
