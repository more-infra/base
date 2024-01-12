package element

import (
	"context"
	"errors"
	"sync/atomic"
)

// ELEMENT is the interface Manager saved.Objects want to be managed by Manager must implement this interface.
// It only has one method, Meta() return the *Element object of the ELEMENT.
type ELEMENT interface {
	// Meta returns the *Element object for Manager.
	Meta() *Element
}

// Element is the meta information for object in Manager.
// It's often used as an embed struct of the object, the Element struct implement the ELEMENT interface.
// So user does not need implement it, only embed *Element object is enough.See testing case more details and example for usage.
// The usual usage is like the follow code, see more details in testing casing.
/*
	type Item struct {
		*element.Element
		serialNum string
		tags []string
		data void
		initTime time.Time
	}

	mgr := element.NewManager()

	// create item...
    itm := &Item{
		Element: mgr.NewElement(),
		serialNum: "2024-01-01.87516",
		tags: []string{"tag1", "tag2", "tag3"}.
		data: // set private data
	}

	// set indexes....
	itm.SetUnique("serial_num", itm.serialNum)
	for _, t := range itm.tags {
		itm.SetIndex("tags", t)
	}

	// set initialization....
	itm.SetInitialization(ctx, f func(context.Context) error{
		// do something with item init
		itm.initTime = time.Now()
		return nil
	})

	// insert into Manager...
	e := mgr.Join(item)
	if e == item {
		log.Println("item insert done")
	} else {
		log.Println("item is exists already, return the exists item")
	}

	// wait for init completed...
	e.Meta().Initialization.Wait()
*/
type Element struct {
	// id is the keys autoincrement id in Manager.
	id uint64

	// in is a flag mark the Element is in the Manager.
	in uint32

	// mgr is the reference of the Manager which this Element is in.
	mgr *Manager

	// initial controls the initialization operation of the Element, it's not required.
	initial *Initialization

	// keys defines all keys of the Element.
	keys map[string][]interface{}

	// indexes defines all indexes of the Element.
	indexes map[string][]interface{}
}

type SearchIndexRelation string

func (t SearchIndexRelation) String() string {
	return string(t)
}

const (
	RelationAND SearchIndexRelation = "and"
	RelationOR  SearchIndexRelation = "or"
)

// UId return the unique autoincrement id of the Element
func (e *Element) UId() uint64 {
	return e.id
}

// Leave is equal to Manager.Remove() method, but it's a method of Element.
// Element is useful in Element's embed scenes, Element can do Leave method instead of find Manager object and do Remove method.
// See testing case and example for more details.
func (e *Element) Leave() {
	e.mgr.Remove(e)
}

// SetKey will set a unique key for the Element. the value type must be types thar supports "==" operation.
func (e *Element) SetKey(field string, value interface{}) {
	e.keys[field] = append(e.keys[field], value)
}

// SetIndex will set an index for the Element. the value type must be types thar supports "==" operation.
func (e *Element) SetIndex(field string, value interface{}) {
	e.indexes[field] = append(e.indexes[field], value)
}

// SetInitialization defines the Element's initialization function.
// The initialization function should be call only once, the input context param will pass to the function.
func (e *Element) SetInitialization(c context.Context, f func(context.Context) error) {
	ctx, cancel := context.WithCancel(c)
	e.initial = &Initialization{
		f:      f,
		c:      ctx,
		cancel: cancel,
	}
}

// Meta implements the ELEMENT interface, so it's used in embed scenes.See testing case and example for more Details.
func (e *Element) Meta() *Element {
	return e
}

func (e *Element) Initialization() *Initialization {
	return e.initial
}

// Initialization controls the operation of the Element initialization.
type Initialization struct {
	f      func(context.Context) error
	c      context.Context
	cancel context.CancelFunc
	err    atomic.Value
}

func (i *Initialization) do() {
	err := i.f(i.c)
	if err != nil {
		i.err.Store(err)
	} else {
		i.err.Store(errNil)
	}
	i.cancel()
}

// Wait is called for waiting initialization completed.If it returns an error, that means the operation is failed.
// This method's error is from initialization function's return.
func (i *Initialization) Wait() error {
	return i.WaitWithContext(context.Background())
}

// WaitWithContext accepts an input context param for controlling the Wait.
func (i *Initialization) WaitWithContext(c context.Context) error {
	select {
	case <-c.Done():
		return c.Err()
	case <-i.c.Done():
		return i.Err()
	}
}

// Done is the signal channel for initialize function done notifying.
func (i *Initialization) Done() <-chan struct{} {
	return i.c.Done()
}

// Err return the error for initialize function.
func (i *Initialization) Err() error {
	v := i.err.Load()
	if v == nil {
		return i.c.Err()
	}
	err := v.(error)
	if err == errNil {
		return nil
	}
	return err
}

var (
	errNil = errors.New("nil")
)
