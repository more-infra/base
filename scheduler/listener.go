package scheduler

import (
	"github.com/more-infra/base/chanpool"
	"github.com/more-infra/base/element"
	"github.com/more-infra/base/runner"
	"sync"
)

type listenerManager struct {
	*runner.Runner
	listeners *element.Manager
	refresh   chan struct{}
	once      sync.Once
}

func newListenerManager() *listenerManager {
	return &listenerManager{
		Runner:    runner.NewRunner(),
		listeners: element.NewManager(),
		refresh:   make(chan struct{}, 1),
	}
}

func (lm *listenerManager) startup() {
	lm.Runner.Mark()
	go lm.running()
}

func (lm *listenerManager) shutdown() {
	lm.Runner.CloseWait()
	close(lm.refresh)
}

func (lm *listenerManager) add(entity *Entity) {
	lm.once.Do(func() {
		lm.startup()
	})
	listener := &listener{
		Element: lm.listeners.NewElement(),
		entity:  entity,
	}
	entity.listener = listener
	lm.listeners.Join(listener)
	select {
	case lm.refresh <- struct{}{}:
	default:
	}
}

func (lm *listenerManager) running() {
	pool := chanpool.NewPool(lm.Runner.Quit(), lm.refresh)
	defer func() {
		pool.Dispose()
		lm.Runner.Done()
	}()
	for {
		pool.Reset()
		snapShot := lm.listeners.Snapshot()
		for _, e := range snapShot {
			l := e.(*listener)
			pool.Push(l, l.done())
		}
		e, flag := pool.Select()
		if flag == chanpool.SelectQuitReturned {
			return
		}
		if flag == chanpool.SelectRefreshReturned {
			continue
		}
		l := e.(*listener)
		l.dispose(l.err())
	}
}

type listener struct {
	*element.Element
	entity *Entity
}

func (l *listener) done() <-chan struct{} {
	return l.entity.listenCtx.Done()
}

func (l *listener) err() error {
	return l.entity.listenCtx.Err()
}

func (l *listener) dispose(err error) {
	l.entity.CancelWithError(err)
	l.Element.Leave()
}

func (l *listener) remove() {
	l.Element.Leave()
}
