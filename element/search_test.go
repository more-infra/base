package element

import "testing"

func TestSearch(t *testing.T) {
	m := NewManager()
	ee := []*entity{
		{
			Element: m.NewElement(),
			index: map[string][]interface{}{
				"index.A": {"1", "2", "3"},
				"index.B": {"4", "5", "6"},
				"index.C": {7, 8, 9},
			},
		},
		{
			Element: m.NewElement(),
			index: map[string][]interface{}{
				"index.A": {"1", "2", "3"},
				"index.B": {"4", "5", "66"},
				"index.C": {7, 8, 99},
			},
		},
		{
			Element: m.NewElement(),
			index: map[string][]interface{}{
				"index.A": {"1", "222", "3"},
				"index.B": {"4", "555", "6"},
				"index.C": {7, 888, 9},
			},
		},
		{
			Element: m.NewElement(),
			index: map[string][]interface{}{
				"index.A": {"1", "222", "3333"},
				"index.B": {"4", "555", "6666"},
				"index.C": {7, 888, 9999},
			},
		},
		{
			Element: m.NewElement(),
			index: map[string][]interface{}{
				"index.A": {"1", "222", "3333"},
				"index.B": {"4", "555", "6666"},
				"index.C": {7, 88888, 9999},
			},
		},
	}
	for n, e := range ee {
		e.init(n)
		m.Join(e)
	}

	assert(t, m.SearchEx(map[string][]interface{}{
		"index.A": {"1"},
	}, RelationOR), []int{
		0, 1, 2, 3, 4,
	})
	assert(t, m.SearchEx(map[string][]interface{}{
		"index.A": {"1"},
	}, RelationAND), []int{
		0, 1, 2, 3, 4,
	})

	assert(t, m.SearchEx(map[string][]interface{}{
		"index.A": {"1"},
		"index.C": {9999},
	}, RelationOR), []int{
		0, 1, 2, 3, 4,
	})
	assert(t, m.SearchEx(map[string][]interface{}{
		"index.A": {"1"},
		"index.C": {9999},
	}, RelationAND), []int{
		3, 4,
	})

	assert(t, m.SearchEx(map[string][]interface{}{
		"index.B": {"555"},
	}, RelationAND), []int{
		2, 3, 4,
	})

	assert(t, m.SearchEx(map[string][]interface{}{
		"index.A": {"1"},
		"index.B": {"555"},
	}, RelationOR), []int{
		0, 1, 2, 3, 4,
	})

	assert(t, m.SearchEx(map[string][]interface{}{
		"index.A": {"1"},
		"index.B": {"555"},
	}, RelationAND), []int{
		2, 3, 4,
	})

	assert(t, m.SearchEx(map[string][]interface{}{
		"index.A": {"1", "3"},
		"index.B": {"6"},
		"index.C": {8},
	}, RelationAND), []int{
		0,
	})

	assert(t, m.SearchEx(map[string][]interface{}{
		"index.C": {"88888"},
	}, RelationOR), []int{
		// 4,
	})
}

func assert(t *testing.T, ee []ELEMENT, expected []int) {
	result := make(map[int]bool)
	for _, e := range ee {
		result[e.(*entity).n] = true
	}
	if len(expected) != len(result) {
		t.Fatal()
	}
	for _, n := range expected {
		if !result[n] {
			t.Fatal()
		}
	}
}

type entity struct {
	*Element
	n     int
	index map[string][]interface{}
}

func (e *entity) init(n int) {
	e.n = n
	for k, vv := range e.index {
		for _, v := range vv {
			e.Element.SetIndex(k, v)
		}
	}
}
