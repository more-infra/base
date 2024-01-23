package scheduler

import (
	"context"
	"github.com/more-infra/base"
	"github.com/more-infra/base/element"
	"github.com/more-infra/base/status"
	"time"
)

// Scheduler provides a goroutine execution pool which supports controlling by options, includes:
//
// goroutine pool num limited
//
// context timeout and cancel controlling for the execution
//
// delay execution with time
//
// graceful shutdown scheduler with execution task notify.
type Scheduler struct {
	statusController *status.Controller
	entities         *element.Manager
	delayMgr         *delayManager
	listenerMgr      *listenerManager
	workerMgr        *workerManager
	option           option
}

func NewScheduler(options ...Option) *Scheduler {
	s := &Scheduler{
		statusController: status.NewController(),
		entities:         element.NewManager(),
		delayMgr:         newDelayManager(),
		listenerMgr:      newListenerManager(),
	}
	for _, op := range options {
		op(s)
	}
	var workerMgrOptions []workerManagerOptionFunc
	if s.option.poolSize != nil {
		workerMgrOptions = append(workerMgrOptions, withWorkerMaxCount(*s.option.poolSize))
	}
	if s.option.poolReduceDuration != nil {
		workerMgrOptions = append(workerMgrOptions, withReduceDuration(*s.option.poolReduceDuration))
	}
	s.workerMgr = newWorkerManager(workerMgrOptions...)
	return s
}

type Option func(*Scheduler)

// WithPoolSize defines the scheduler's goroutine pool max size.
// The goroutines in pool can auto increase or reduce by load situation, the max pool size control the upper limit.
// The default value is runtime.NumCPU() * 2
func WithPoolSize(size int) Option {
	if size == 0 {
		panic("scheduler pool size could not be zero")
	}
	return func(s *Scheduler) {
		s.option.poolSize = &size
	}
}

// WithPoolReduceDuration controls the pool reduce duration when the Scheduler is idle.
// The default value is 120s
func WithPoolReduceDuration(dur time.Duration) Option {
	return func(s *Scheduler) {
		s.option.poolReduceDuration = &dur
	}
}

// Start should be called before Push Entity to Scheduler for executing.
// It's thread-safe.
func (s *Scheduler) Start() {
	if !s.statusController.Starting() {
		return
	}
	defer s.statusController.Started()
}

// Stop should be called when Scheduler is not used and do shutting down.
// When Stop called, Scheduler will cancel all entities which is executing or waiting for scheduling and wait for them done,
// then Stop return finally, it's graceful.
// It's also thread-safe.
func (s *Scheduler) Stop() {
	if !s.statusController.Stopping() {
		return
	}
	defer s.statusController.Stopped()
	s.delayMgr.shutdown()
	s.listenerMgr.shutdown()
	s.workerMgr.shutdown()
	snapShot := s.entities.Snapshot()
	for _, e := range snapShot {
		entity := e.(*Entity)
		go func() {
			entity.Cancel()
		}()
	}
	for _, e := range snapShot {
		entity := e.(*Entity)
		<-entity.Done()
	}
}

// Executor is the running unit in scheduler which is wrapped by Scheduler.NewEntity.
// Caller must implement the method of it for scheduling or executing by Scheduler.
// See Scheduler.NewEntity for more details.
type Executor interface {
	// Do will be called when the Entity is executed in Scheduler.
	// Params description:
	//
	// c is the context provides controlling by Scheduler.
	// When context.Done() is signal, it means Scheduler should cancel the executing of this Executor.
	//
	// Return description:
	// The result of execution, the error could be acquired by Entity's Result method
	Do(c context.Context) error

	// Abandon will be called when the Entity could not be executed.
	// It always happened when the Scheduler Stop called before the Entity execute.
	Abandon()
}

// ExecutorWrapper is a wrapper of Executor.Use function instead of create object and implements Executor interface.
type ExecutorWrapper struct {
	DoFunc      func(context.Context) error
	AbandonFunc func()
}

func (ew *ExecutorWrapper) Do(c context.Context) error {
	return ew.DoFunc(c)
}

func (ew *ExecutorWrapper) Abandon() {
	if ew.AbandonFunc != nil {
		ew.AbandonFunc()
	}
}

// NewEntity create an execution entity of executor in Scheduler.
// The options param defines the control of Executor.
//
// WithEntityContext defines the context controller.
//
// WithEntityDelay defines the delay scheduling of the Executor.
//
// When an Entity created, it will not be scheduled immediately, call Push to insert it to Scheduler and prepare for execute.
func (s *Scheduler) NewEntity(executor Executor, options ...EntityOption) *Entity {
	c, cancel := context.WithCancel(context.Background())
	entity := &Entity{
		element:  s.entities.NewElement(),
		s:        s,
		executor: executor,
		c:        c,
		cancel:   cancel,
		result: &Result{
			Status: StatusWaiting,
		},
		timing: &timing{
			created: time.Now(),
		},
	}
	for _, opt := range options {
		opt(entity)
	}
	return entity
}

// Push will insert an Entity to the Scheduler, then scheduler it with policy defines by its options.
// Entity's Dispatch method gets the same effect.
// If the Scheduler is not Start or has Stop, Push will return a "status invalid" error.
func (s *Scheduler) Push(e *Entity) error {
	if !s.statusController.KeepRunning() {
		return base.NewErrorWithType(status.ErrTypeInvalidStatus, status.ErrInvalidStatus).
			WithMessage("Scheduler Push fail with stopped status").
			WithStack()
	}
	if e.delay != 0 {
		s.delayMgr.add(e)
	} else {
		s.schedule(e)
	}
	s.statusController.ReleaseRunning()
	return nil
}

func (s *Scheduler) schedule(entity *Entity) {
	s.entities.Join(entity)
	if entity.listenCtx != nil {
		s.listenerMgr.add(entity)
	}
	s.workerMgr.push(entity)
}

type option struct {
	poolSize           *int
	poolReduceDuration *time.Duration
}
