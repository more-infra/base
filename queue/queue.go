package queue

import (
	"context"
	"github.com/eapache/queue"
	"github.com/more-infra/base/runner"
	"sync"
	"sync/atomic"
	"time"
)

// Buffer provides the channel that capacity can be extended dynamically.
// When make a channel, the capacity is defined by make function, such as make(chan int, 8),
// it does not support capacity extending. This package is useful in these scene,
// the capacity of the channel could be set dynamically or set to unrestricted if the memory is enough.
//
// Buffer includes two buffers internal, a go chan and a self-defined queue struct.
// The input for the Buffer will be inserted to go chan firstly, when the go chan is full, the input elements will be
// inserted to the self-defined queue struct and then a background goroutine will put the elements in queue into go chan continuously.
// As you see, the self-defined queue is used for buffering the elements when the go chan is full, and it keeps the order of elements input.
type Buffer struct {
	runner        *runner.Runner
	queue         *queue.Queue
	mu            sync.Mutex
	sign          chan struct{}
	ch            chan interface{}
	closed        int32
	buffering     bool
	chCapacity    int
	queueCapacity int
	policy        Policy
	idleTime      time.Duration
}

// NewBuffer create a buffer with the options. The options have default value if inputs are not set.
// The Dispose method is required to call when the Buffer is not used, or leak of goroutine will be happened.
func NewBuffer(options ...BufferOption) *Buffer {
	b := &Buffer{
		runner:        runner.NewRunner(),
		queue:         queue.New(),
		sign:          make(chan struct{}, 1),
		idleTime:      DefaultBufferingIdleTime,
		chCapacity:    DefaultChannelCapacity,
		queueCapacity: 0,
		policy:        PolicyDrop,
	}
	for _, op := range options {
		op(b)
	}
	b.ch = make(chan interface{}, b.chCapacity)
	return b
}

type BufferOption func(*Buffer)

const (
	DefaultChannelCapacity   = 128
	DefaultBufferingIdleTime = 10 * time.Second
)

// WithChannelCapacity set the channel capacity, this value could not be changed after the Buffer is created.
// The default value is 128.
func WithChannelCapacity(cap int) BufferOption {
	return func(b *Buffer) {
		b.chCapacity = cap
	}
}

// WithQueueCapacity set the self-defined queue capacity, this value could be changed by SetCapacity method.
// The default value is unlimited, meaning the queue could be always extended when it's required.
func WithQueueCapacity(cap int) BufferOption {
	return func(b *Buffer) {
		b.queueCapacity = cap
	}
}

// WithBufferingIdleTime defines the idle time of the background goroutine keeping when the self-defined queue is empty.
// The default value is 10 seconds.
func WithBufferingIdleTime(dur time.Duration) BufferOption {
	return func(b *Buffer) {
		b.idleTime = dur
	}
}

// WithQueuePolicy defines the policy when the queue is full.
//
// PolicyDrop: drop the input element
//
// PolicyRemove: remove the tail of queue and insert the input element
//
// PolicyClear: clear the all queue, and insert element to the new queue.
//
// The default value is PolicyDrop
func WithQueuePolicy(policy Policy) BufferOption {
	return func(b *Buffer) {
		b.policy = policy
	}
}

// Push is input method for Buffer. It's thread-safe.
// After Dispose method is called, the input element will not be dropped instead of insert.
func (b *Buffer) Push(elm interface{}) PushResult {
	if atomic.CompareAndSwapInt32(&b.closed, 1, 1) {
		return PushDropped
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if atomic.CompareAndSwapInt32(&b.closed, 1, 1) {
		return PushDropped
	}
	if !b.buffering && b.queue.Length() == 0 {
		// send to channel directly when buffer is empty
		select {
		case b.ch <- elm:
			return PushToChan
		default:
		}
	}
	ret := PushToQueue
	if b.queueCapacity != 0 && b.queueCapacity == b.queue.Length() {
		// do action by policy when queue is full
		switch b.policy {
		case PolicyDrop:
			return PushDropped
		case PolicyRemove:
			b.queue.Remove()
			ret = PushToQueueReplace
		case PolicyClear:
			b.queue = queue.New()
			ret = PushToQueueReplace
		}
	}
	b.queue.Add(elm)
	if !b.buffering {
		b.buffering = true
		b.runner.Mark()
		go b.running()
	}
	select {
	case b.sign <- struct{}{}:
	default:
	}
	return ret
}

// Channel return the receiver chan. The chan will be close after Dispose method is called.
func (b *Buffer) Channel() <-chan interface{} {
	return b.ch
}

// SetCapacity set the self-defined queue's capacity dynamically.
func (b *Buffer) SetCapacity(cap int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.queueCapacity = cap
}

// Dispose is required to called when the Buffer is not used.
func (b *Buffer) Dispose() {
	if atomic.CompareAndSwapInt32(&b.closed, 0, 1) {
		b.runner.CloseWait()
		close(b.ch)
		close(b.sign)
	}
}

func (b *Buffer) running() {
	defer b.runner.Done()
	for {
		var e interface{}
		b.mu.Lock()
		if b.queue.Length() != 0 {
			e = b.queue.Remove()
		}
		b.mu.Unlock()
		if e == nil {
			// buffer's all element had been consumed
			var (
				c      = context.Background()
				cancel context.CancelFunc
			)
			if b.idleTime != 0 {
				c, cancel = context.WithTimeout(c, b.idleTime)
			}
			select {
			case <-b.runner.Quit():
				if cancel != nil {
					cancel()
				}
				return
			case <-b.sign:
			case <-c.Done():
			}
			if cancel != nil {
				cancel()
			}
			b.mu.Lock()
			if b.queue.Length() != 0 {
				e = b.queue.Remove()
			} else {
				b.buffering = false
			}
			b.mu.Unlock()
		}
		if e == nil {
			return
		}
		// sending element to called channel
		select {
		case <-b.runner.Quit():
			return
		case b.ch <- e:
		}
	}
}

type PushResult string

func (r PushResult) String() string {
	return string(r)
}

const (
	PushToChan         PushResult = "push to chan"
	PushToQueue        PushResult = "push to queue"
	PushToQueueReplace PushResult = "push to queue replace"
	PushDropped        PushResult = "push dropped"
)

type Policy string

func (p Policy) String() string {
	return string(p)
}

const (
	PolicyDrop   Policy = "drop"
	PolicyRemove Policy = "remove"
	PolicyClear  Policy = "clear"
)
