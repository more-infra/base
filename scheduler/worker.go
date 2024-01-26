package scheduler

import (
	"context"
	"github.com/more-infra/base/element"
	"github.com/more-infra/base/queue"
	"github.com/more-infra/base/runner"
	"runtime"
	"sync"
	"time"
)

type workerManager struct {
	runner   *runner.Runner
	option   workerManagerOption
	c        context.Context
	cancel   context.CancelFunc
	queue    *queue.Buffer
	taskChan chan func()
	workers  *element.Manager
	once     sync.Once
}

type workerManagerOption struct {
	count          int
	reduceDuration time.Duration
}

type workerManagerOptionFunc func(*workerManagerOption)

func withWorkerMaxCount(count int) workerManagerOptionFunc {
	return func(option *workerManagerOption) {
		option.count = count
	}
}

func withReduceDuration(dur time.Duration) workerManagerOptionFunc {
	return func(option *workerManagerOption) {
		option.reduceDuration = dur
	}
}

func newWorkerManager(optionFuncs ...workerManagerOptionFunc) *workerManager {
	c, cancel := context.WithCancel(context.Background())
	mgr := &workerManager{
		runner: runner.NewRunner(),
		option: workerManagerOption{
			count:          runtime.NumCPU() * 2,
			reduceDuration: 120 * time.Second,
		},
		c:        c,
		cancel:   cancel,
		taskChan: make(chan func()),
		queue:    queue.NewBuffer(),
		workers:  element.NewManager(),
	}
	for _, f := range optionFuncs {
		f(&mgr.option)
	}
	return mgr
}

func (wm *workerManager) startup() {
	wm.runner.Mark()
	go wm.running()
}

func (wm *workerManager) shutdown() {
	wm.cancel()
	wm.runner.CloseWait()
	wm.queue.Dispose()
	var wg sync.WaitGroup
	snapShot := wm.workers.Snapshot()
	for _, e := range snapShot {
		worker := e.(*worker)
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker.shutdown()
		}()
	}
	wg.Wait()
}

func (wm *workerManager) push(entity *Entity) {
	wm.once.Do(wm.startup)
	wm.queue.Push(entity)
}

func (wm *workerManager) running() {
	timer := time.NewTimer(wm.option.reduceDuration)
	defer func() {
		timer.Stop()
		wm.runner.Done()
	}()
	for {
		select {
		case <-wm.runner.Quit():
			return
		case v := <-wm.queue.Channel():
			entity := v.(*Entity)
			var overload bool
			select {
			case <-wm.runner.Quit():
				return
			case wm.taskChan <- func() { entity.onExecute(wm.c) }:
			default:
				overload = true
			}
			if overload {
				wm.grow()
				select {
				case <-wm.runner.Quit():
					return
				case wm.taskChan <- func() { entity.onExecute(wm.c) }:
				}
			}
		case <-timer.C:
			wm.reduce()
			timer.Reset(wm.option.reduceDuration)
		}
	}
}

func (wm *workerManager) capacity() int {
	return wm.workers.Count()
}

func (wm *workerManager) grow() {
	if wm.capacity() < wm.option.count {
		worker := wm.newWorker()
		worker.startup()
		wm.workers.Join(worker)
	}
}

func (wm *workerManager) reduce() {
	var wg sync.WaitGroup
	snapShot := wm.workers.Snapshot()
	for _, e := range snapShot {
		worker := e.(*worker)
		if worker.idle() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				worker.shutdown()
			}()
		}
	}
	wg.Wait()
}

func (wm *workerManager) newWorker() *worker {
	return &worker{
		element:  wm.workers.NewElement(),
		runner:   runner.NewRunner(),
		taskChan: wm.taskChan,
		idleChan: make(chan struct{}),
	}
}

type worker struct {
	element  *element.Element
	runner   *runner.Runner
	taskChan chan func()
	idleChan chan struct{}
}

func (w *worker) startup() {
	w.runner.Mark()
	go w.running()
}

func (w *worker) shutdown() {
	w.runner.CloseWait()
	close(w.idleChan)
	w.element.Leave()
}

func (w *worker) running() {
	defer w.runner.Done()
	for {
		select {
		case <-w.runner.Quit():
			return
		case f := <-w.taskChan:
			f()
		case <-w.idleChan:
		}
	}
}

func (w *worker) idle() bool {
	select {
	case w.idleChan <- struct{}{}:
		return true
	default:
		return false
	}
}

func (w *worker) Meta() *element.Element {
	return w.element
}
