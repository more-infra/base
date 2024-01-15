package mcontext

import (
	"context"
	"testing"
)

func TestMultipleContext(t *testing.T) {
	var (
		cc      []context.Context
		cancels []context.CancelFunc
	)
	defer func() {
		for _, cancel := range cancels {
			cancel()
		}
	}()
	for i := 0; i != 100; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		ctx = context.WithValue(ctx, "NO", i)
		cc = append(cc, ctx)
		cancels = append(cancels, cancel)
	}
	mc := NewMultipleContext(cc...)
	mc.Listen()
	defer mc.Dispose()

	index := 88
	go func() {
		cancels[index]()
	}()

	select {
	case <-mc.Done():
	}
	c := mc.Hit()
	if c == nil {
		t.Fatal("context Hit is nil")
	}

	if c.Err() != mc.Err() {
		t.Fatal("context.Err() is not equal")
	}

	n := c.Value("NO").(int)
	if n != index {
		t.Fatalf("context Hit[%d] is not expected[%d]", n, index)
	}
}
