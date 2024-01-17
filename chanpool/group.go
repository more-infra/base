package chanpool

import (
	"context"
	"github.com/more-infra/base/reactor"
	"log"
	"reflect"
	"sync"
)

type group struct {
	reactor *reactor.Reactor
	group   *Pool
	cases   []reflect.SelectCase
	ctxs    []interface{}
	pos     int
}

func (this *group) startup() {
	this.reactor.Start()
}

func (this *group) shutdown() {
	this.reactor.Stop()
}

func (this *group) reset() {
	this.cases[0].Chan = reflect.ValueOf(this.group.done)
	this.pos = 1
}

func (this *group) push(ctx interface{}, ch interface{}) bool {
	if this.pos == groupMaxCount {
		return false
	}
	this.cases[this.pos].Chan = reflect.ValueOf(ch)
	this.ctxs[this.pos] = ctx
	this.pos++
	return true
}

func (this *group) pushSelect(wg *sync.WaitGroup) {
	if err := this.reactor.Push(func(context.Context) {
		defer wg.Done()
		n, _, _ := reflect.Select(this.cases[:this.pos])
		if n == 0 {
			return
		}
		select {
		case this.group.result <- this.ctxs[n]:
		default:
		}
	}); err != nil {
		log.Println("chanpool.group::pushSelect failed with reactor Push", err)
	}
}
