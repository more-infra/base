package scheduler

import (
	"github.com/more-infra/base/chanpool"
	"github.com/more-infra/base/element"
	"github.com/more-infra/base/runner"
	"sync"
	"time"
)

type delayManager struct {
	*runner.Runner
	items   *element.Manager
	refresh chan struct{}
	once    sync.Once
}

func newDelayManager() *delayManager {
	return &delayManager{
		Runner:  runner.NewRunner(),
		items:   element.NewManager(),
		refresh: make(chan struct{}, 1),
	}
}

func (dm *delayManager) startup() {
	dm.Runner.Mark()
	go dm.running()
}

func (dm *delayManager) shutdown() {
	dm.Runner.CloseWait()
	close(dm.refresh)
}

func (dm *delayManager) add(e *Entity) {
	dm.once.Do(func() {
		dm.startup()
	})
	item := &delayItem{
		Element: dm.items.NewElement(),
		entity:  e,
		timer:   time.NewTimer(e.delay),
	}
	dm.items.Join(item)
	select {
	case dm.refresh <- struct{}{}:
	default:
	}
}

func (dm *delayManager) running() {
	pool := chanpool.NewPool(dm.Runner.Quit(), dm.refresh)
	defer func() {
		pool.Dispose()
		dm.Runner.Done()
	}()
	for {
		pool.Reset()
		snapShot := dm.items.Snapshot()
		for _, e := range snapShot {
			item := e.(*delayItem)
			pool.Push(item, item.expired())
		}
		e, flag := pool.Select()
		if flag == chanpool.SelectQuitReturned {
			return
		}
		if flag == chanpool.SelectRefreshReturned {
			continue
		}
		item := e.(*delayItem)
		item.dispose()
	}
}

type delayItem struct {
	*element.Element
	entity *Entity
	timer  *time.Timer
}

func (di *delayItem) expired() <-chan time.Time {
	return di.timer.C
}

func (di *delayItem) dispose() {
	di.entity.dispatch()
	di.Element.Leave()
}
