package mcontext

import (
	"context"
	"github.com/more-infra/base/runner"
	"reflect"
	"sync/atomic"
	"time"
)

// MultipleContext is used in multiple contexts select scenes.
// When several contexts(the number is uncertain) are required to listen with select, the code is not easy to write.
// This object helps you listen contexts by select only one. It implements the context interface such as Done(), Err(), Deadline(), Value().
// So it could be used as a context.Context interface.
type MultipleContext struct {
	runner *runner.Runner
	c      context.Context
	cancel context.CancelFunc
	cc     []context.Context
	hit    atomic.Value
}

func NewMultipleContext(c ...context.Context) *MultipleContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &MultipleContext{
		runner: runner.NewRunner(),
		cc:     c,
		c:      ctx,
		cancel: cancel,
	}
}

// Listen is required to called before using the MultipleContext object.
// Dispose is required to called when the object is not need.
func (this *MultipleContext) Listen() {
	this.runner.Mark()
	go this.running()
}

// Dispose is called with Listen in pair.
func (this *MultipleContext) Dispose() {
	this.runner.CloseWait()
}

// Hit return the context had Done.If there are no context Done, nil will be returned.
func (this *MultipleContext) Hit() context.Context {
	v := this.hit.Load()
	if v == nil {
		return nil
	}
	return v.(context.Context)
}

// Done is the implement of context.Context interface
func (this *MultipleContext) Done() <-chan struct{} {
	return this.c.Done()
}

// Value is the implement of context.Context interface
func (this *MultipleContext) Value(key interface{}) interface{} {
	return this.c.Value(key)
}

// Deadline is the implement of context.Context interface
func (this *MultipleContext) Deadline() (time.Time, bool) {
	c := this.Hit()
	if c != nil {
		return c.Deadline()
	}
	return this.c.Deadline()
}

// Err is the implement of context.Context interface
func (this *MultipleContext) Err() error {
	c := this.Hit()
	if c != nil {
		return c.Err()
	}
	return this.c.Err()
}

func (this *MultipleContext) running() {
	defer func() {
		this.cancel()
		this.runner.Done()
	}()
	cases := []reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(this.runner.Quit()),
		},
	}
	for _, c := range this.cc {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(c.Done()),
		})
	}
	chosen, _, _ := reflect.Select(cases)
	if chosen == 0 {
		return
	}
	this.hit.Store(this.cc[chosen-1])
}
