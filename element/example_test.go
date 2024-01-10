package element

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

type item struct {
	*Element
	tags  []string
	key   string
	value int
}

type testdataItem struct {
	key   string
	tags  []string
	value int
}

func TestJoinRemove(t *testing.T) {
	mgr := NewManager()
	var items []*item
	for i := 0; i != 100; i++ {
		itm := &item{
			Element: mgr.NewElement(),
			value:   i,
		}
		mgr.Join(itm)
		items = append(items, itm)
	}
	if mgr.Count() != 100 {
		t.Fatalf("Manager.Count()[%d] is not expected[%d]", mgr.Count(), 100)
	}

	sum := func(st map[uint64]ELEMENT) int {
		var s int
		for _, e := range st {
			s += e.(*item).value
		}
		return s
	}
	snapShotBefore := mgr.Snapshot()
	// remove first 50 elements
	for i := 0; i != 50; i++ {
		// use Element.Leave instead of mgr.Remove is more simply
		items[i].Element.Leave()
	}

	if mgr.Count() != 50 {
		t.Fatalf("Manager.Count()[%d] is not expected[%d] after removed", mgr.Count(), 50)
	}

	snapShotAfter := mgr.Snapshot()

	beforeSum := sum(snapShotBefore)
	expectedBeforeSum := (0 + 99) * len(snapShotBefore) / 2
	if beforeSum != expectedBeforeSum {
		t.Fatalf("before snapshot sum[%d] is not expected[%d]", beforeSum, expectedBeforeSum)
	}

	afterSum := sum(snapShotAfter)
	expectedAfterSum := (50 + 99) * len(snapShotAfter) / 2

	if afterSum != expectedAfterSum {
		t.Fatalf("after snapshot sum[%d] is not expected[%d]", afterSum, expectedAfterSum)
	}

	mgr.Clear()

	if mgr.Count() != 0 {
		t.Fatal("Manager.Count() is not zero after Clear() called")
	}

	if len(mgr.Snapshot()) != 0 {
		t.Fatal("Manager.Snapshot is not empty after Clean() called")
	}
}

func TestConcurrentJoinAndInitialization(t *testing.T) {
	mgr := NewManager()
	var (
		inserted        int32
		initCalled      int32
		exists          int32
		insertItemValue int
		initItemValue   int
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i != 10; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			itm := &item{
				Element: mgr.NewElement(),
				key:     "same key",
				value:   i,
			}
			// all insert item has the same key, so only one item could be inserted successfully.
			itm.SetKey("item.key", itm.key)
			itm.SetInitialization(ctx, func(c context.Context) error {
				initItemValue = i
				t.Log("Initialization callback called")
				atomic.AddInt32(&initCalled, 1)
				return nil
			})
			e := mgr.Join(itm)
			if e != itm {
				atomic.AddInt32(&exists, 1)
			} else {
				atomic.AddInt32(&inserted, 1)
				insertItemValue = i
			}
			if err := e.Meta().Initialization().Wait(); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if inserted != 1 {
		t.Errorf("inserted[%d] is not expexted[%d]", inserted, 1)
	}
	if exists != 9 {
		t.Errorf("inserted[%d] is not expexted[%d]", exists, 9)
	}
	if initCalled != 1 {
		t.Errorf("initCalled[%d] is not expexted[%d]", inserted, 1)
	}
	if mgr.Count() != 1 {
		t.Errorf("elements[%d] count is not expected[%d]", mgr.Count(), 1)
	}
	if insertItemValue != initItemValue {
		t.Errorf("insertItemValue[%d] is not equal to initItemValue[%d]", insertItemValue, initItemValue)
	}
}

func TestIndexSearch(t *testing.T) {
	testItems := []*testdataItem{
		{"", []string{"odd"}, 1},
		{"", []string{"even"}, 2},
		{"", []string{"odd"}, 3},
		{"", []string{"even"}, 4},
		{"", []string{"odd"}, 5},
		{"", []string{"even"}, 6},
		{"", []string{"odd"}, 7},
		{"", []string{"even"}, 8},
	}

	mgr := NewManager()
	insertItemsWithTestData(mgr, testItems)

	ee := mgr.Search("item.index.tag", "odd")
	for _, e := range ee {
		v := e.(*item).value
		if v%2 == 0 {
			t.Errorf("Search by index 'odd' return[%d] is not expected", v)
		}
	}

	ee = mgr.Search("item.index.tag", "even")
	for _, e := range ee {
		v := e.(*item).value
		if v%2 != 0 {
			t.Errorf("Search by index 'even' return[%d] is not expected", v)
		}
	}
}

func insertItemsWithTestData(mgr *Manager, testItems []*testdataItem) {
	for _, tm := range testItems {
		itm := &item{
			Element: mgr.NewElement(),
			tags:    tm.tags,
			key:     tm.key,
			value:   tm.value,
		}
		itm.SetKey("item.primary.key", tm.key)
		for _, tag := range tm.tags {
			itm.SetIndex("item.index.tag", tag)
		}
		mgr.Join(itm)
	}
}
