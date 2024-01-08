package element

import (
	"sync"
	"sync/atomic"
)

// Manager is designed for elements manager which like a simple database used, provides CRUD operations.
// It's the container of elements, and manages they with index(unique index is also support).
type Manager struct {
	autoId   uint64
	rw       sync.RWMutex
	elements map[uint64]ELEMENT
	unique   map[string]map[interface{}]uint64
	index    map[string]map[interface{}]map[uint64]bool
}

func NewManager() *Manager {
	return &Manager{
		autoId:   0,
		elements: make(map[uint64]ELEMENT),
		unique:   make(map[string]map[interface{}]uint64),
		index:    make(map[string]map[interface{}]map[uint64]bool),
	}
}

// NewElement is the creator of element, every element wants to be managed by Manager must be created by this method.
// Element implements the interface of ELEMENT. See Element for more details.
// Each Element has a unique autoincrement id.
func (m *Manager) NewElement() *Element {
	return &Element{
		id:     atomic.AddUint64(&m.autoId, 1),
		in:     0,
		mgr:    m,
		unique: make(map[string][]interface{}),
		index:  make(map[string][]interface{}),
	}
}

// Join is used to insert an element to the manager.
// This method is thread safe and concurrent supported.
// When multiple threads/goroutine call Join with the same input ELEMENT, only one ELEMENT will be inserted.
// If the ELEMENT is already exists in the manager(which judges by the autoincrement id of the ELEMENT), it will return the exists ELEMENT.
// So return value is the inserted ELEMENT or the ELEMENT already exists.
// * ---------------- About the Initialization ---------------- *
// The initialization function will be called only one times when the ELEMENT inserted into Manager successfully, join will wait it complete and return.
// The recommended way is using the returned ELEMENT object, and do ELEMENT.Initialization().Wait(),
// because the Join will return immediately when the ELEMENT is already exists.
func (m *Manager) Join(e ELEMENT) ELEMENT {
	meta := e.Meta()
	if atomic.CompareAndSwapUint32(&meta.in, 1, 1) {
		return e
	}
	m.rw.Lock()
	if atomic.CompareAndSwapUint32(&meta.in, 1, 1) {
		m.rw.Unlock()
		return e
	}
	ee, ok := m.elements[meta.id]
	if ok {
		m.rw.Unlock()
		return ee
	}
	// insert unique
	for f, vv := range meta.unique {
		_, ok := m.unique[f]
		if !ok {
			m.unique[f] = make(map[interface{}]uint64)
		}
		for _, v := range vv {
			id, ok := m.unique[f][v]
			if ok {
				e := m.elements[id]
				m.rw.Unlock()
				return e
			}
			m.unique[f][v] = meta.id
		}
	}
	// insert index
	for f, vv := range meta.index {
		_, ok := m.index[f]
		if !ok {
			m.index[f] = make(map[interface{}]map[uint64]bool)
		}
		for _, v := range vv {
			ids, ok := m.index[f][v]
			if !ok {
				ids = make(map[uint64]bool)
			}
			ids[meta.id] = true
			m.index[f][v] = ids
		}
	}
	m.elements[meta.id] = e
	atomic.StoreUint32(&meta.in, 1)
	m.rw.Unlock()
	initial := meta.initial
	if initial != nil {
		initial.do()
	}
	return e
}

// Get finds the element by unique autoincrement id
func (m *Manager) Get(id uint64) ELEMENT {
	m.rw.RLock()
	defer m.rw.RUnlock()
	e, ok := m.elements[id]
	if !ok {
		return nil
	}
	return e
}

// Find query the ELEMENT with unique index.
// It will return the found ELEMENT, nil will be returned if ELEMENT not found.
func (m *Manager) Find(unique string, value interface{}) ELEMENT {
	m.rw.RLock()
	defer m.rw.RUnlock()
	ref, ok := m.unique[unique]
	if !ok {
		return nil
	}
	id, ok := ref[value]
	if !ok {
		return nil
	}
	return m.elements[id]
}

// SearchEx enhances multiple indexes searching with relationship than Search.
func (m *Manager) SearchEx(indexes map[string][]interface{}, relation SearchIndexRelation) []ELEMENT {
	m.rw.RLock()
	defer m.rw.RUnlock()
	elIds := make(map[uint64]bool)
	var init bool
	for field, values := range indexes {
		ref, ok := m.index[field]
		if !ok {
			if relation == RelationAND {
				return []ELEMENT{}
			}
			continue
		}
		for _, value := range values {
			ids, ok := ref[value]
			if !ok {
				if relation == RelationAND {
					return []ELEMENT{}
				}
				continue
			}
			switch relation {
			case RelationAND:
				if !init {
					for id := range ids {
						elIds[id] = true
					}
					init = true
				} else {
					for id := range elIds {
						if !ids[id] {
							delete(elIds, id)
						}
					}
				}
			case RelationOR:
				for id := range ids {
					elIds[id] = true
				}
			}
		}
	}
	if len(elIds) == 0 {
		return []ELEMENT{}
	}
	els := make([]ELEMENT, len(elIds), len(elIds))
	n := 0
	for id := range elIds {
		els[n] = m.elements[id]
		n++
	}
	return els
}

// Search is used for find the ELEMENTS by index. It will return empty array(nil) when no ELEMENTS found.
func (m *Manager) Search(index string, value interface{}) []ELEMENT {
	m.rw.RLock()
	defer m.rw.RUnlock()
	var els []ELEMENT
	ref, ok := m.index[index]
	if !ok {
		return els
	}
	ids, ok := ref[value]
	if !ok {
		return els
	}
	for id := range ids {
		els = append(els, m.elements[id])
	}
	return els
}

// GroupByIndex groups elements by input index, the return map is always no-nil
func (m *Manager) GroupByIndex(index string) map[interface{}][]ELEMENT {
	m.rw.RLock()
	defer m.rw.RUnlock()
	els := make(map[interface{}][]ELEMENT)
	ref, ok := m.index[index]
	if !ok {
		return els
	}
	for v, ids := range ref {
		var ee []ELEMENT
		for id := range ids {
			ee = append(ee, m.elements[id])
		}
		els[v] = ee
	}
	return els
}

// Snapshot makes a copy of current elements in Manager
func (m *Manager) Snapshot() map[uint64]ELEMENT {
	copys := make(map[uint64]ELEMENT)
	m.rw.RLock()
	for k, v := range m.elements {
		copys[k] = v
	}
	m.rw.RUnlock()
	return copys
}

// Count returns the elements count in the Manager.
func (m *Manager) Count() int {
	m.rw.RLock()
	count := len(m.elements)
	m.rw.RUnlock()
	return count
}

func (m *Manager) Empty() bool {
	m.rw.RLock()
	empty := len(m.elements) == 0
	m.rw.RUnlock()
	return empty
}

// Remove is used to remove an Element in Manager.
// * Notice: the input param type is *Element not ELEMENT.
func (m *Manager) Remove(e *Element) {
	if atomic.CompareAndSwapUint32(&e.in, 0, 0) {
		return
	}
	m.rw.Lock()
	defer m.rw.Unlock()
	if atomic.CompareAndSwapUint32(&e.in, 0, 0) {
		return
	}
	defer atomic.StoreUint32(&e.in, 0)
	id := e.id
	_, ok := m.elements[id]
	if !ok {
		return
	}
	for f, vv := range e.index {
		for _, v := range vv {
			delete(m.index[f][v], id)
		}
	}
	for f, vv := range e.unique {
		for _, v := range vv {
			delete(m.unique[f], v)
		}
	}
	delete(m.elements, id)
}

// Clear will reset the Manager and clean all ELEMENTS in it.
func (m *Manager) Clear() {
	m.rw.Lock()
	defer m.rw.Unlock()
	m.elements = make(map[uint64]ELEMENT)
	m.unique = make(map[string]map[interface{}]uint64)
	m.index = make(map[string]map[interface{}]map[uint64]bool)
}
