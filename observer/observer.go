package observer

import (
	"github.com/more-infra/base/element"
	"github.com/more-infra/base/event"
	"github.com/more-infra/base/queue"
	"github.com/more-infra/base/runner"
	"github.com/more-infra/base/status"
	"sync"
	"sync/atomic"
)

// Manager is the manager of the observer. 
// Use Add to add a new observer, and use Push to push the event to all observers in manager.
type Manager struct {
	observers       *element.Manager
	queueOptions    []queue.BufferOption
	observerOptions []ObserverOption
}

// Option is the option for the Manager.
// It includes the observer options and the queue options.	
type Option func(*Manager)

// WithObserverOption is the option for the observers in Manager.
func WithObserverOption(options ...ObserverOption) Option {
	return func(m *Manager) {
		m.observerOptions = append(m.observerOptions, options...)
	}
}

// WithQueueBufferOption is the option for the queue buffer in Observers of the Manager.
func WithQueueBufferOption(options ...queue.BufferOption) Option {
	return func(m *Manager) {
		m.queueOptions = append(m.queueOptions, options...)
	}
}

// ObserverOption is the option for the Observer.
type ObserverOption func(*Observer)

// WithNotifyChannelCapacity is the option for set the capacity of the notify channel in Observer.
// The default capacity is 0, which means the notify channel is unbuffered.
func WithNotifyChannelCapacity(capacity int) ObserverOption {
	return func(ob *Observer) {
		if ob.notifyCh != nil {
			close(ob.notifyCh)
		}
		ob.notifyCh = make(chan *event.Event, capacity)
	}
}

// NewManager creates a new Manager with the given options.
// Then use Add to add observers to the manager.
func NewManager(options ...Option) *Manager {
	mgr := &Manager{
		observers: element.NewManager(),
	}
	for _, op := range options {
		op(mgr)
	}
	return mgr
}

// Add adds a new observer to the manager.
func (m *Manager) Add() *Observer {
	ob := m.newObserver()
	m.observers.Join(ob)
	ob.startup()
	return ob
}

// Push pushes the event to all observers in the manager.
// Every Observer use chan returned by Notify() to receive the event.
func (m *Manager) Push(evt *event.Event) {
	snapShot := m.observers.Snapshot()
	for _, e := range snapShot {
		ob := e.(*Observer)
		ob.push(evt)
	}
}

// Dispose disposes the manager and the observers.
// Call it when you don't need the manager anymore.
func (m *Manager) Dispose() {
	var wg sync.WaitGroup
	snapShot := m.observers.Snapshot()
	for _, e := range snapShot {
		ob := e.(*Observer)
		wg.Add(1)
		go func() {
			ob.dispose()
			wg.Done()
		}()
	}
	wg.Wait()
}

func (m *Manager) newObserver() *Observer {
	ob := &Observer{
		element:          m.observers.NewElement(),
		runner:           runner.NewRunner(),
		statusController: status.NewController(),
		disposedCh:       make(chan struct{}),
		notifyCh:         make(chan *event.Event),
		eventQueue:       queue.NewBuffer(m.queueOptions...),
	}
	for _, op := range m.observerOptions {
		op(ob)
	}
	return ob
}

// Observer is the observer to receive the event from the manager.
type Observer struct {
	element          *element.Element
	runner           *runner.Runner
	statusController *status.Controller
	eventQueue       *queue.Buffer
	notifyCh         chan *event.Event
	disposedCh       chan struct{}
	disposed         int64
}

// Notify returns a channel to receive the event from the manager.
// WithNotifyChannelCapacity option can set the capacity of the notify channel.
// The default capacity is 0, which means the notify channel is unbuffered.
func (ob *Observer) Notify() <-chan *event.Event {
	return ob.notifyCh
}

// Disposed returns a channel to receive the signal when Observer is closed, which happpens when Close	is called.
func (ob *Observer) Disposed() <-chan struct{} {
	return ob.disposedCh
}

// Close will be called when the observer is not needed anymore.
// When it's called, the Disposed() channel will be closed and got a signal.
func (ob *Observer) Close() {
	ob.shutdown()
}

func (ob *Observer) Meta() *element.Element {
	return ob.element
}

func (ob *Observer) startup() {
	if !ob.statusController.Starting() {
		return
	}
	defer ob.statusController.Started()
	ob.runner.Mark()
	go ob.running()
}

func (ob *Observer) shutdown() {
	if !ob.statusController.Stopping() {
		return
	}
	defer ob.statusController.Stopped()
	ob.dispose()
	ob.runner.CloseWait()
	ob.eventQueue.Dispose()
	close(ob.notifyCh)
	ob.element.Leave()
}

func (ob *Observer) dispose() {
	if atomic.CompareAndSwapInt64(&ob.disposed, 0, 1) {
		close(ob.disposedCh)
	}
}

func (ob *Observer) push(evt *event.Event) {
	if !ob.statusController.KeepRunning() {
		return
	}
	ob.eventQueue.Push(evt)
	ob.statusController.ReleaseRunning()
}

func (ob *Observer) running() {
	defer ob.runner.Done()
	for {
		select {
		case <-ob.runner.Quit():
			return
		case v, ok := <-ob.eventQueue.Channel():
			if !ok {
				return
			}
			evt := v.(*event.Event)
			select {
			case <-ob.runner.Quit():
				return
			case ob.notifyCh <- evt:
			}
		}
	}
}
