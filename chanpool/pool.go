package chanpool

import (
	"github.com/more-infra/base/reactor"
	"reflect"
	"sync"
)

// Pool is used for multiple chan select scenes.
//
// If you want to select several chan which is clear, the code is like this
/*
	var ch1,ch2,ch3
	select {
	case <-ch1:
	case <-ch2:
	case <-ch3:
	}
*/
// When the chan is ambiguous, it's hard to write the code for select them.
// Putting these chan into a Pool, and then calls Select method will select them together.
// See examples or testing cases for more details.
type Pool struct {
	done   chan struct{}
	result chan interface{}
	cases  []reflect.SelectCase
	groups []*group
	pos    int
}

// NewPool creates a Pool, quit and refresh input param is required for controlling when do Select.
// The quit chan is used for notifying Select method when the pool requires to be shutdown.
// The refresh chan is used for notifying Select method when the chan in pool requires to be updated.
// The typical usage is:
/*
	var contexts [3]interface{}
	var chs [3]chan interface{}
	p := NewPool(quit, refresh)
	defer p.Dispose()
	for {
		// reset the pool and push channels again
		p.Rest()

		// push the channels with context to the Pool
		for i := 0; i != len(chs); i++ {
			p.Push(contexts[i], chs[i])
		}

		// do Select the channels pushed to the Pool
		ret, ctx := p.Select()
		switch ret {
			case SelectQuitReturned:
				// quit chan is signed, return the loop
				return
			case SelectRefreshReturned:
				// refresh chan is signed, reset and re-push channels
				continue
			case SelectKeyReturned:
				// one chan in chs signed, the ctx value is the first param in Push method
		}
	}
*/
// Notice: Dispose() will be called when the Pool is not using.
func NewPool(quit chan struct{}, refresh chan struct{}) *Pool {
	p := &Pool{
		pos:    -1,
		result: make(chan interface{}),
	}
	result := make(chan interface{})
	p.result = result
	p.cases = []reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(quit),
		},
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(refresh),
		},
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(result),
		},
	}
	return p
}

// Reset will clear all channels in the Pool and recover the Pool to initial.
func (this *Pool) Reset() {
	this.done = make(chan struct{})
	if this.pos != -1 {
		this.pos = -1
	}
}

// Push will insert a channel with context to the Pool.
// The ch param is the chan to be Select, and the ctx param will associate with the ch param,
// Select will return the ctx when the ch is signal.
// When quit or refresh chan is signal, the return ctx is the chan itself.
func (this *Pool) Push(ctx interface{}, ch interface{}) {
	var curGroup *group
	if this.pos == -1 {
		if len(this.groups) == 0 {
			curGroup = this.addGroup()
			curGroup.push(ctx, ch)
			return
		}
		this.pos++
		curGroup = this.groups[this.pos]
		curGroup.reset()
		curGroup.push(ctx, ch)
		return
	}
	curGroup = this.groups[this.pos]
	if curGroup.push(ctx, ch) {
		return
	}
	if this.pos == len(this.groups)-1 {
		curGroup = this.addGroup()
		curGroup.push(ctx, ch)
		return
	}
	this.pos++
	curGroup = this.groups[this.pos]
	curGroup.reset()
	curGroup.push(ctx, ch)
}

// Select will check all channels in the Pool as select do.
// It will return when the channels signal or quit, refresh chan signal, the SelectResult will tell the reason.
func (this *Pool) Select() (interface{}, SelectResult) {
	var wg sync.WaitGroup
	for i := 0; i != this.pos+1; i++ {
		group := this.groups[i]
		wg.Add(1)
		group.pushSelect(&wg)
	}
	n, v, _ := reflect.Select(this.cases)
	close(this.done)
	wg.Wait()
	if n == 0 {
		return nil, SelectQuitReturned
	}
	if n == 1 {
		return nil, SelectRefreshReturned
	}
	return v.Interface(), SelectKeyReturned
}

// Dispose clear the Pool when it's not using.
// It should be called, otherwise goroutine leak will be happened.
func (this *Pool) Dispose() {
	var wg sync.WaitGroup
	for _, group := range this.groups {
		group := group
		wg.Add(1)
		go func() {
			group.shutdown()
			wg.Done()
		}()
	}
	wg.Wait()
	close(this.result)
}

func (this *Pool) addGroup() *group {
	group := this.newGroup()
	group.startup()
	group.reset()
	this.groups = append(this.groups, group)
	this.pos++
	return group
}

func (this *Pool) newGroup() *group {
	lg := &group{
		reactor: reactor.NewReactor(),
		group:   this,
		cases:   make([]reflect.SelectCase, groupMaxCount, groupMaxCount),
		ctxs:    make([]interface{}, groupMaxCount, groupMaxCount),
		pos:     0,
	}
	for i := 0; i != groupMaxCount; i++ {
		lg.cases[i] = reflect.SelectCase{
			Dir: reflect.SelectRecv,
		}
	}
	return lg
}
