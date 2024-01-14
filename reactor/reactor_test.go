package reactor

import (
	"context"
	"sync"
	"testing"
)

func TestOrder(t *testing.T) {
	r := NewReactor()
	r.Start()
	defer r.Stop()
	const (
		num = 1000
	)
	var (
		result []int
		wg     sync.WaitGroup
	)
	for i := 0; i != num; i++ {
		i := i
		wg.Add(1)
		var err error
		if i%7 != 0 {
			err = r.Push(func(context.Context) {
				defer wg.Done()
				result = append(result, i)
			})
		} else {
			err = r.Send(func(context.Context) {
				defer wg.Done()
				result = append(result, i)
			})
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
	if len(result) != num {
		t.Fatalf("result count[%d] is not expected[%d]", len(result), num)
	}
	for i := 0; i != num; i++ {
		if result[i] != i {
			t.Errorf("result[%d] is not expected[%d]", result[i], i)
		}
	}
}

func TestPriority(t *testing.T) {
	r := NewReactor()
	r.Start()
	defer r.Stop()
	const (
		num          = 100
		priorityBase = 1000
	)

	var (
		result []int
		wg     sync.WaitGroup
	)
	ctxStartLine, ctxStartLineCancel := context.WithCancel(context.Background())

	if err := r.Push(func(context.Context) {
		<-ctxStartLine.Done()
	}); err != nil {
		t.Fatal(err)
	}

	if err := r.PushPriority(func(context.Context) {
		<-ctxStartLine.Done()
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i != num; i++ {
		n := i
		wg.Add(1)
		if err := r.Push(func(context.Context) {
			defer wg.Done()
			result = append(result, n)
		}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i != num; i++ {
		n := i + priorityBase
		wg.Add(1)
		if err := r.PushPriority(func(context.Context) {
			defer wg.Done()
			result = append(result, n)
		}); err != nil {
			t.Fatal(err)
		}
	}
	ctxStartLineCancel()
	wg.Wait()
	if len(result) != 2*num {
		t.Fatalf("result count[%d] is not expected[%d]", len(result), 2*num)
	}
	for i, n := range result {
		if i < num {
			// priority queue result
			if n != priorityBase+i {
				t.Errorf("result[%d] is not expected[%d]", n, priorityBase+i)
			}
		} else {
			if n != i-num {
				t.Errorf("result[%d] is not expected[%d]", n, i-num)
			}
		}
	}
}

func TestWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := NewReactor(WithContext(ctx))
	r.Start()
	defer r.Stop()
	var done int

	if err := r.Send(func(context.Context) {
		done++
	}); err != nil {
		t.Fatal(err)
	}
	if done != 1 {
		t.Fatal("Send Handler is not run")
	}

	cancel()
	err := r.Send(func(context.Context) {
		done++
	})
	if err == nil && done == 2 {
		t.Fatal("Send Handler is run after context canceled")
	}
}
