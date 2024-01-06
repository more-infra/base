package status

import (
	"context"
	"fmt"
	"github.com/more-infra/base"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type service struct {
	*Controller
	chReq chan string
	t     *testing.T
	wg    sync.WaitGroup
	done  chan struct{}
}

func newService(t *testing.T) *service {
	return &service{
		Controller: NewController(),
		chReq:      make(chan string),
		t:          t,
		done:       make(chan struct{}),
	}
}

func (s *service) startup() error {
	if !s.Controller.Starting() {
		return base.NewErrorWithType(ErrTypeInvalidStatus, ErrInvalidStatus).
			WithMessage("service startup failed").
			WithStack()
	}
	defer s.Controller.Started()
	s.t.Log("service is starting")
	s.wg.Add(1)
	go s.running()
	s.t.Log("service is started")
	return nil
}

func (s *service) shutdown() error {
	if !s.Controller.Stopping() {
		return base.NewErrorWithType(ErrTypeInvalidStatus, ErrInvalidStatus).
			WithMessage("service shutdown failed").
			WithStack()
	}
	defer s.Controller.Stopped()
	s.t.Log("service is stopping")
	close(s.done)
	close(s.chReq)
	s.wg.Wait()
	s.t.Log("service is stopped")
	return nil
}

func (s *service) sendRequest(v string) error {
	if !s.Controller.KeepRunning() {
		return base.NewErrorWithType(ErrTypeInvalidStatus, ErrInvalidStatus).
			WithMessage("service is not running for sendRequest").
			WithField("input.v", v)
	}
	defer s.Controller.ReleaseRunning()
	s.chReq <- v
	return nil
}

func (s *service) sendRequestWithContext(ctx context.Context, v string) error {
	if !s.Controller.KeepRunningWithContext(ctx) {
		return base.NewErrorWithType(ErrTypeInvalidStatus, ErrInvalidStatus).
			WithMessage("service is not running for sendRequest").
			WithField("input.v", v)
	}
	defer s.Controller.ReleaseRunning()
	s.chReq <- v
	return nil
}

func (s *service) running() {
	timer := time.NewTimer(100 * time.Millisecond)
	defer func() {
		timer.Stop()
		s.wg.Done()
		s.t.Log("service running down")
	}()
	for {
		select {
		case <-s.done:
			s.t.Log("service running receive down signal")
			return
		case v := <-s.chReq:
			s.t.Log("service.input:", v)
		case <-timer.C:
			s.t.Log("service is running")
			timer.Reset(100 * time.Millisecond)
		}
	}
}

func TestConcurrentStart(t *testing.T) {
	srv := newService(t)
	var (
		success, failed int32
		wg              sync.WaitGroup
	)
	defer func() {
		_ = srv.shutdown()
	}()
	for i := 0; i != 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srv.startup(); err != nil {
				t.Log(err)
				atomic.AddInt32(&failed, 1)
			} else {
				atomic.AddInt32(&success, 1)
			}
		}()
	}
	wg.Wait()
	if success != 1 {
		t.Fatal(fmt.Sprintf("success[%d] is not expected[1]", success))
	}
	if failed != 9 {
		t.Fatal(fmt.Sprintf("failed[%d] is not expected[9]", failed))
	}
}

func TestConcurrentStop(t *testing.T) {
	srv := newService(t)
	var (
		success, failed int32
		wg              sync.WaitGroup
	)
	if err := srv.startup(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i != 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srv.shutdown(); err != nil {
				t.Log(err)
				atomic.AddInt32(&failed, 1)
			} else {
				atomic.AddInt32(&success, 1)
			}
		}()
	}
	wg.Wait()
	if success != 1 {
		t.Fatal(fmt.Sprintf("success[%d] is not expected[1]", success))
	}
	if failed != 9 {
		t.Fatal(fmt.Sprintf("failed[%d] is not expected[9]", failed))
	}
}

func TestRequest(t *testing.T) {
	srv := newService(t)
	var (
		success, failed int32
		wg              sync.WaitGroup
	)
	if err := srv.startup(); err != nil {
		t.Fatal(err)
	}

	for i := 0; i != 10; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srv.sendRequest(strconv.Itoa(i)); err != nil {
				t.Error(err)
				atomic.AddInt32(&failed, 1)
			} else {
				atomic.AddInt32(&success, 1)
			}
		}()
	}
	wg.Wait()
	if err := srv.shutdown(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i != 10; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srv.sendRequest(strconv.Itoa(i)); err != nil {
				t.Log(err)
				atomic.AddInt32(&failed, 1)
			} else {
				atomic.AddInt32(&success, 1)
			}
		}()
	}
	wg.Wait()
	if success != 10 {
		t.Fatal(fmt.Sprintf("success[%d] is not expected[10]", success))
	}
	if failed != 10 {
		t.Fatal(fmt.Sprintf("failed[%d] is not expected[10]", failed))
	}
}

func TestStartWithContext(t *testing.T) {
	srv := newService(t)
	if err := srv.sendRequest("sendRequest failed because of the service is not startup"); err != nil {
		t.Log(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second)
		if err := srv.startup(); err != nil {
			t.Error(err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	if err := srv.sendRequestWithContext(ctx, "timeout = 1 * NanoSecond"); err != nil {
		t.Log(err)
	} else {
		t.Fatal("sendRequest should failed for context timeout before the service startup")
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.sendRequestWithContext(ctx, "timeout = 5 * Second"); err != nil {
		t.Fatal("sendRequest should success after the service startup")
	}

	wg.Wait()

	_ = srv.shutdown()
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := srv.sendRequestWithContext(ctx, "timeout = 1 * Second"); err != nil {
		t.Log(err)
	} else {
		t.Fatal("sendRequest should failed for service is shutdown")
	}
}
