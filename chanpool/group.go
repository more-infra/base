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

func (g *group) startup() {
	g.reactor.Start()
}

func (g *group) shutdown() {
	g.reactor.Stop()
}

func (g *group) reset() {
	g.cases[0].Chan = reflect.ValueOf(g.group.done)
	g.pos = 1
}

func (g *group) push(ctx interface{}, ch interface{}) bool {
	if g.pos == groupMaxCount {
		return false
	}
	g.cases[g.pos].Chan = reflect.ValueOf(ch)
	g.ctxs[g.pos] = ctx
	g.pos++
	return true
}

func (g *group) pushSelect(wg *sync.WaitGroup) {
	if err := g.reactor.Push(func(context.Context) {
		defer wg.Done()
		n, _, _ := reflect.Select(g.cases[:g.pos])
		if n == 0 {
			return
		}
		select {
		case g.group.result <- g.ctxs[n]:
		default:
		}
	}); err != nil {
		log.Println("chanpool.group::pushSelect failed with reactor Push", err)
	}
}
