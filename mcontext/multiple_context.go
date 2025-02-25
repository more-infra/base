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
func (mc *MultipleContext) Listen() {
	mc.runner.Mark()
	go mc.running()
}

// Dispose is called with Listen in pair.
func (mc *MultipleContext) Dispose() {
	mc.runner.CloseWait()
}

// Hit return the context had Done.If there are no context Done, nil will be returned.
func (mc *MultipleContext) Hit() context.Context {
	v := mc.hit.Load()
	if v == nil {
		return nil
	}
	return v.(context.Context)
}

// Done is the implement of context.Context interface
func (mc *MultipleContext) Done() <-chan struct{} {
	return mc.c.Done()
}

// Value is the implement of context.Context interface
func (mc *MultipleContext) Value(key interface{}) interface{} {
	return mc.c.Value(key)
}

// Deadline is the implement of context.Context interface
func (mc *MultipleContext) Deadline() (time.Time, bool) {
	c := mc.Hit()
	if c != nil {
		return c.Deadline()
	}
	return mc.c.Deadline()
}

// Err is the implement of context.Context interface
func (mc *MultipleContext) Err() error {
	c := mc.Hit()
	if c != nil {
		return c.Err()
	}
	return mc.c.Err()
}

func (mc *MultipleContext) running() {
	defer func() {
		mc.cancel()
		mc.runner.Done()
	}()
	cases := []reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(mc.runner.Quit()),
		},
	}
	for _, c := range mc.cc {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(c.Done()),
		})
	}
	chosen, _, _ := reflect.Select(cases)
	if chosen == 0 {
		return
	}
	mc.hit.Store(mc.cc[chosen-1])
}
