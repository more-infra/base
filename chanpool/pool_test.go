package chanpool

import (
	"math/rand"
	"sync"
	"testing"
)

func TestQuit(t *testing.T) {
	var (
		quit    = make(chan struct{})
		refresh = make(chan struct{})
	)
	defer close(refresh)
	p := NewPool(quit, refresh)
	defer p.Dispose()
	const (
		chanCount = 100
	)

	p.Reset()
	var triggerChan []chan int
	for j := 0; j != chanCount; j++ {
		ch := make(chan int, 1)
		p.Push(j, ch)
		triggerChan = append(triggerChan, ch)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(quit)
	}()
	ctx, ret := p.Select()
	if ret != SelectQuitReturned && ctx != quit {
		t.Fatalf("Select return[%s] is not expected SelectQuitReturned", ret.String())
	}
	wg.Wait()
	for _, ch := range triggerChan {
		close(ch)
	}
}

func TestRefresh(t *testing.T) {
	var (
		quit    = make(chan struct{})
		refresh = make(chan struct{})
	)
	defer close(quit)
	p := NewPool(quit, refresh)
	defer p.Dispose()
	const (
		chanCount = 100
	)

	p.Reset()
	var triggerChan []chan int
	for j := 0; j != chanCount; j++ {
		ch := make(chan int, 1)
		p.Push(j, ch)
		triggerChan = append(triggerChan, ch)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(refresh)
	}()
	ctx, ret := p.Select()
	if ret != SelectRefreshReturned && ctx != quit {
		t.Fatalf("Select return[%s] is not expected SelectRefreshReturned", ret.String())
	}
	wg.Wait()
	for _, ch := range triggerChan {
		close(ch)
	}
}

func TestBigPool(t *testing.T) {
	var (
		quit    = make(chan struct{})
		refresh = make(chan struct{})
	)
	p := NewPool(quit, refresh)
	defer p.Dispose()

	const (
		testNum   = 3
		chanCount = 6553700
	)

	for i := 0; i != testNum; i++ {
		p.Reset()
		var triggerChan []chan int
		for j := 0; j != chanCount; j++ {
			ch := make(chan int, 1)
			p.Push(j, ch)
			triggerChan = append(triggerChan, ch)
		}
		n := int(rand.Int31n(chanCount))
		triggerChan[n] <- n
		ctx, ret := p.Select()
		switch ret {
		case SelectQuitReturned:
			t.Fatal("unexpected PoolSelectQuit received")
		case SelectRefreshReturned:
			t.Fatal("unexpected PoolSelectRefresh received")
		case SelectKeyReturned:
			v := ctx.(int)
			if v != n {
				t.Fatalf("trigger ctx[%d] is not expected[%d]", v, n)
			}
		default:
			t.Fatal("invalid select return received")
		}
		for _, ch := range triggerChan {
			close(ch)
		}
	}
}
