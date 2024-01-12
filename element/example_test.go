package element

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

type item struct {
	*Element
	tags  map[string][]interface{}
	key   map[string]string
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
				key:     map[string]string{keyUniqueValue: "same key"},
				value:   i,
			}
			// all insert item has the same key, so only one item could be inserted successfully.
			itm.SetKey(keyUniqueValue, itm.key[keyUniqueValue])
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
func TestKeyFind(t *testing.T) {
	num := 100
	testItems := make([]*testdataItem, 100)
	for i := 0; i != num; i++ {
		testItems[i] = &testdataItem{
			key: map[string]string{
				keySeq: strconv.Itoa(i),
			},
			tags:  nil,
			value: i,
		}
	}
	mgr := NewManager()
	insertItemsWithTestData(mgr, testItems)

	for i := 0; i != num*2; i++ {
		key := strconv.Itoa(i)
		e := mgr.Find(keySeq, key)
		if i < num {
			if e == nil {
				t.Errorf("primary key[%s] is not found as expected", key)
			}
		} else {
			if e != nil {
				t.Errorf("primary key[%s] is found which isn't as expected", key)
			}
		}
	}
}

func TestIndexSearch(t *testing.T) {
	testItems := []*testdataItem{
		{nil, map[string][]interface{}{indexMath: {"odd"}}, 1},
		{nil, map[string][]interface{}{indexMath: {"even"}}, 2},
		{nil, map[string][]interface{}{indexMath: {"odd"}}, 3},
		{nil, map[string][]interface{}{indexMath: {"even"}}, 4},
		{nil, map[string][]interface{}{indexMath: {"odd"}}, 5},
		{nil, map[string][]interface{}{indexMath: {"even"}}, 6},
		{nil, map[string][]interface{}{indexMath: {"odd"}}, 7},
		{nil, map[string][]interface{}{indexMath: {"even"}}, 8},
	}

	mgr := NewManager()
	insertItemsWithTestData(mgr, testItems)

	ee := mgr.Search(indexMath, "odd")
	for _, e := range ee {
		v := e.(*item).value
		if v%2 == 0 {
			t.Errorf("Search by index 'odd' return[%d] is not expected", v)
		}
	}

	ee = mgr.Search(indexMath, "even")
	for _, e := range ee {
		v := e.(*item).value
		if v%2 != 0 {
			t.Errorf("Search by index 'even' return[%d] is not expected", v)
		}
	}
}

func TestMultipleKeys(t *testing.T) {
	num := 100
	testItems := make([]*testdataItem, 100)
	for i := 0; i != num; i++ {
		testItems[i] = &testdataItem{
			key: map[string]string{
				keySeq: strconv.Itoa(i),
				// "0" or "1", so only two items will be inserted, others will fail with exists check.
				keyUniqueValue: strconv.Itoa(i % 2),
			},
			tags:  nil,
			value: i,
		}
	}
	mgr := NewManager()
	insertItemsWithTestData(mgr, testItems)
	if mgr.Count() != 2 {
		t.Fatalf("Manager.Count()[%d] is not expected[%d]", mgr.Count(), 2)
	}
	for _, e := range mgr.Snapshot() {
		itm := e.(*item)
		if itm.value != 0 && itm.value != 1 {
			t.Errorf("invalid item inserted value[%d]", itm.value)
		}
	}
}

func TestMultipleIndex(t *testing.T) {
	num := 100
	testItems := make([]*testdataItem, 100)
	for i := 0; i != num; i++ {
		var math []interface{}
		if i%2 == 0 {
			math = append(math, "even")
		} else {
			math = append(math, "odd")
		}
		if i%3 == 0 {
			math = append(math, "mt")
		}
		testItems[i] = &testdataItem{
			key: nil,
			tags: map[string][]interface{}{
				indexDecimal: {i / 10 * 10},
				indexMath:    math,
			},
			value: i,
		}
	}
	mgr := NewManager()
	insertItemsWithTestData(mgr, testItems)
	if mgr.Count() != num {
		t.Fatalf("Manager.Count()[%d] is not expected[%d]", mgr.Count(), num)
	}

	// search the num which is 10-20 or 90-100 or is an even
	ee := mgr.SearchEx(map[string][]interface{}{
		indexDecimal: {10, 20, 90},
		indexMath:    {"even"},
	}, RelationOR)

	var expectedResult int
	for i := 0; i != num; i++ {
		decimal := i / 10 * 10
		if (decimal >= 10 && decimal < 30) || (decimal >= 90 && decimal < 100) ||
			i%2 == 0 {
			expectedResult++
			continue
		}
	}
	if len(ee) != expectedResult {
		t.Fatalf("SearchEx find multiple three numbers result[%d] is not expected[%d]", len(ee), expectedResult)
	}

	// search the num which is 30-40, then is both odd and mt
	ee = mgr.SearchEx(map[string][]interface{}{
		indexDecimal: {30},
		indexMath:    {"odd", "mt"},
	}, RelationAND)

	expectedResult = 0
	for i := 0; i != num; i++ {
		decimal := i / 10 * 10
		if (decimal >= 30 && decimal < 40) &&
			i%2 != 0 && i%3 == 0 {
			expectedResult++
			continue
		}
	}
	if len(ee) != expectedResult {
		t.Fatalf("SearchEx find multiple three numbers result[%d] is not expected[%d]", len(ee), expectedResult)
	}
}

const (
	keySeq         = "item.key.seq"
	keyUniqueValue = "item.key.unique_value"
	indexMath      = "item.index.math"
	indexDecimal   = "item.index.decimal"
)

type testdataItem struct {
	key   map[string]string
	tags  map[string][]interface{}
	value int
}

func insertItemsWithTestData(mgr *Manager, testItems []*testdataItem) {
	for _, tm := range testItems {
		itm := &item{
			Element: mgr.NewElement(),
			tags:    tm.tags,
			key:     tm.key,
			value:   tm.value,
		}
		if tm.key != nil {
			for k, v := range tm.key {
				itm.SetKey(k, v)
			}
		}
		if tm.tags != nil {
			for k, vv := range tm.tags {
				for _, v := range vv {
					itm.SetIndex(k, v)
				}
			}
		}
		mgr.Join(itm)
	}
}
