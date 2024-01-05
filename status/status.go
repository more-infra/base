package status

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
)

// Controller is used for controlling the service or worker that has status, and widely used in projects.
// A service may have typical status, such as waiting_for_start, starting, running, stopping, stopped.
// When the service is not running, it could not be accepted operations, and it could not be operated when it's stopped.
// It's not allowed two or more called in difference goroutine do start a service at the same time,
// so the "starting" status is designed for protecting it and a sync.RWMutex is also required.
// Similarly, "stopping" status is useful when do stop operation.
// Typical usage is embed in an object for status controller. See testing example for more detail.
type Controller struct {
	status int
	rw     sync.RWMutex
	stop   sync.RWMutex
	down   bool
	err    atomic.Value
}

func NewController() *Controller {
	return &Controller{
		status: Ready,
	}
}

// Starting is called for start the service. It's always called with Started or Failed in pair.
// It returns true means the current goroutine got the permission for the service start,
// then you should call Started when the service start success or Failed with error when failed.
// The false return value means the service had been running, stopped or other goroutine has got the permission for starting.
// * Notice: forget to call Started or Failed when Starting return true will cause deadlock.
// * The usual usage is:
/*
	if !Controller.Starting() {
	    return err
	}

	err = service.StartImp()    // do service start operation...

	if err != nil {
  		Controller.Failed(err)
	} else {
  		Controller.Started()
	}
*/
func (c *Controller) Starting() bool {
	c.rw.Lock()
	if c.status != Ready {
		c.rw.Unlock()
		return false
	}
	c.status = Starting
	return true
}

// Failed is always called with Starting in pair. See Starting for more details.
func (c *Controller) Failed(err error) {
	if c.status == Ready {
		c.status = Stopped
	} else if c.status == Starting {
		c.status = Stopped
		c.rw.Unlock()
	} else {
		panic("incorrect status in calling Failed")
	}
	c.err.Store(err)
}

// Started is always called with Starting in pair. See Starting for more details.
func (c *Controller) Started() {
	if c.status == Ready {
		c.status = Running
	} else if c.status == Starting {
		c.status = Running
		c.rw.Unlock()
	} else {
		panic("incorrect status in calling Started")
	}
}

// Stopping is called for stop the service.
// It returns false means the service is not running or had been stopped or other goroutine has got the permission for stopping.
// If the return value is true, you will do service stop operation and then call Stopped to change the status to "stopped".
// * Notice: forget to call Stopped when Stopping return true will cause deadlock.
// * The usual usage is:
/*
	if !Controller.Stopping() {
	    return err
	}
	defer Controller.Stopped()
	err = service.StopImp()    // do service stop operation...
*/
func (c *Controller) Stopping() bool {
	c.stop.Lock()
	c.down = true
	c.stop.Unlock()
	c.rw.Lock()
	if c.status != Running {
		c.rw.Unlock()
		return false
	}
	c.status = Stopping
	return true
}

// Stopped is always called with Stopping in pair. See Stopping for more details.
func (c *Controller) Stopped() {
	if c.status != Stopping {
		panic("incorrect status in calling Stopped")
	}
	c.status = Stopped
	c.rw.Unlock()
}

// KeepRunning is used for caller when requests the service to guarantee the service status is running.
// When caller request the service, follow step will happen
// 1. check if the service is running(if it's not running, the request is failed).
// 2. prevent the service stopping until the request is done.
// 3. remove the service stopping preventing.
// When the KeepRunning return false, it means the service is not running, you could not do any request to the service.
// When it returns true, caller must call ReleaseRunning after request is done.
// * Notice: forget to call ReleaseRunning when KeepRunning return true will cause deadlock.
func (c *Controller) KeepRunning() bool {
	c.stop.RLock()
	if c.down {
		c.stop.RUnlock()
		return false
	}
	c.rw.RLock()
	if c.status == Running {
		return true
	}
	c.rw.RUnlock()
	c.stop.RUnlock()
	return false
}

// KeepRunningWithContext accept a context for waiting control for KeepRunning.
// When the service is not running, KeepRunning will return false, then caller could not do any requests,
// use KeepRunningWithContext, caller can wait service status from "ready" to "running" by input context param.
// * Notice: If the service had been "stopped" or it's "stopping", KeepRunningWithContext will return false.
func (c *Controller) KeepRunningWithContext(ctx context.Context) bool {
	c.stop.RLock()
	if c.down {
		c.stop.RUnlock()
		return false
	}
	for {
		c.rw.RLock()
		if c.status == Running {
			return true
		} else if c.status == Ready {
			c.rw.RUnlock()
			select {
			case <-ctx.Done():
				c.stop.RUnlock()
				return false
			default:
			}
			runtime.Gosched()
		} else {
			c.rw.RUnlock()
			c.stop.RUnlock()
			return false
		}
	}
}

// ReleaseRunning is always called with KeepRunning or KeepRunningWithContext in pair. See KeepRunning for more details.
func (c *Controller) ReleaseRunning() {
	if c.status != Running {
		panic("incorrect status in calling ReleaseRunning")
	}
	c.rw.RUnlock()
	c.stop.RUnlock()
}

// StatusError return the error which set by Failed, if the error is nil, return the input error.
func (c *Controller) StatusError(err error) error {
	x := c.err.Load()
	if x == nil {
		return err
	}
	return x.(error)
}

const (
	Ready = iota
	Starting
	Running
	Stopping
	Stopped
)
