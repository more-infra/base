package runner

import (
	"context"
	"sync"
)

// Runner is a useful feature for background goroutine life cycle control.
// It wraps the sync.WaitGroup and provides a channel for receiving the close notify signal.
// It's used in background go routine loop task as usual.
// All method are thread safe and reentrant.
type Runner struct {
	c      context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewRunner create a Runner object, the typical usage is embed in an object. See testing example for more detail.
func NewRunner() *Runner {
	return NewRunnerWithContext(context.Background())
}

// NewRunnerWithContext create a Runner object which could control by context or call CloseWait method.
func NewRunnerWithContext(ctx context.Context) *Runner {
	c, cancel := context.WithCancel(ctx)
	return &Runner{
		c:      c,
		cancel: cancel,
	}
}

// CloseWait will stop the runner by close the signal channel and wait for the sync.WaitGroup all Done Synchronously
func (r *Runner) CloseWait() {
	r.cancel()
	r.wg.Wait()
}

// Wait is the same as sync.WaitGroup.Wait()
func (r *Runner) Wait() {
	r.wg.Wait()
}

// Mark is the same as sync.WaitGroup.Add()
func (r *Runner) Mark() {
	r.wg.Add(1)
}

// Done is the same as sync.WaitGroup.Done()
func (r *Runner) Done() {
	r.wg.Done()
}

// Quit defines a channel for signal receiving, when the CloseWait is called, this channel will be close
func (r *Runner) Quit() <-chan struct{} {
	return r.c.Done()
}
