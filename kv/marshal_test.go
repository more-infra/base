package kv

import (
	"testing"
)

func TestMarshalNilPointer(t *testing.T) {
	m := NewMapper()
	type Object struct {
		N int `kv:"n"`
	}
	var nilObject *Object
	kv := m.ObjectToMap(nilObject)
	assertMap(t, kv, map[string]interface{}{})
}

func TestMarshalEmptyInterface(t *testing.T) {
	m := NewMapper()
	var v interface{}
	kv := m.ObjectToMap(v)
	assertMap(t, kv, map[string]interface{}{})
}

func TestMarshalFieldEmptyInterface(t *testing.T) {
	m := NewMapper()
	type Object struct {
		V interface{} `kv:"n,omitempty"`
	}
	kv := m.ObjectToMap(&Object{})
	assertMap(t, kv, map[string]interface{}{})
}

func TestMarshalEmptyTagNameFormat(t *testing.T) {
	type Object struct {
		FieldName string
	}

	m := NewMapper().
		WithEmptyTagFormat(Ignore)
	kv := m.ObjectToMap(Object{FieldName: "field_value"})
	assertMap(t, kv, map[string]interface{}{})

	m = NewMapper().
		WithEmptyTagFormat(OriginFormat)
	kv = m.ObjectToMap(Object{FieldName: "field_value"})
	assertMap(t, kv, map[string]interface{}{
		"FieldName": "field_value",
	})

	m = NewMapper().
		WithEmptyTagFormat(CamelCaseFormat)
	kv = m.ObjectToMap(Object{FieldName: "field_value"})
	assertMap(t, kv, map[string]interface{}{
		"FieldName": "field_value",
	})

	m = NewMapper().
		WithEmptyTagFormat(UnderScoreCaseFormat)
	kv = m.ObjectToMap(Object{FieldName: "field_value"})
	assertMap(t, kv, map[string]interface{}{
		"field_name": "field_value",
	})
}

func TestMarshalNestStruct(t *testing.T) {
	m := NewMapper().
		WithNestConcat(".")
	type NestObject struct {
		NS string `kv:"ns"`
	}
	type Object struct {
		Nest NestObject `kv:"nest"`
	}
	kv := m.ObjectToMap(&Object{
		Nest: NestObject{NS: "ns_value"},
	})
	assertMap(t, kv, map[string]interface{}{
		"nest.ns": "ns_value",
	})
}

func TestMarshalSlice(t *testing.T) {
	m := NewMapper().
		WithSliceOrderConcat("*")
	type Object struct {
		Slice []string `kv:"slice,omitempty"`
	}
	kv := m.ObjectToMap(&Object{
		Slice: []string{"1", "2"},
	})
	assertMap(t, kv, map[string]interface{}{
		"slice*1": "1",
		"slice*2": "2",
	})

	kv = m.ObjectToMap(&Object{
		Slice: []string{},
	})
	assertMap(t, kv, map[string]interface{}{})

	type EmptyObject struct {
		Slice []string `kv:"slice"`
	}
	kv = m.ObjectToMap(&EmptyObject{
		Slice: []string{},
	})
	assertMap(t, kv, map[string]interface{}{})

	type NilObject struct {
		Slice []string `kv:"slice"`
	}
	kv = m.ObjectToMap(&NilObject{
		Slice: nil,
	})
	assertMap(t, kv, map[string]interface{}{
		"slice": nil,
	})
}

func TestMarshalMap(t *testing.T) {
	m := NewMapper()
	type NestObject struct {
		NS string `kv:"ns"`
	}
	type Object struct {
		Map map[string]interface{} `kv:"map"`
	}
	kv := m.ObjectToMap(&Object{
		Map: map[string]interface{}{
			"string": "string_value",
			"nest_object": NestObject{
				NS: "ns_value",
			},
			"nest_object_pointer": &NestObject{
				NS: "ns_value",
			},
			"nest_slice": []string{"slice_value_a", "slice_value_b"},
		},
	})
	assertMap(t, kv, map[string]interface{}{
		"map_string":                 "string_value",
		"map_nest_object_ns":         "ns_value",
		"map_nest_object_pointer_ns": "ns_value",
		"map_nest_slice_1":           "slice_value_a",
		"map_nest_slice_2":           "slice_value_b",
	})
}

type ObjectMarshalInt struct {
	N int                `kv:"n"`
	F ObjectMarshalFloat `kv:"float"`
}

type ObjectMarshalFloat struct {
	F float64 `kv:"f"`
}

func (this *ObjectMarshalFloat) MapperMarshal() interface{} {
	return this.F
}

func TestMarshaller(t *testing.T) {
	type Object struct {
		Int ObjectMarshalInt `kv:"int"`
		S   string           `kv:"s"`
	}
	mapper := NewMapper()
	m := mapper.ObjectToMap(&Object{
		Int: ObjectMarshalInt{
			N: 99,
			F: ObjectMarshalFloat{
				F: 66.66,
			},
		},
		S: "88",
	})
	assertMap(t, m, map[string]interface{}{
		"s":           "88",
		"int_n":       99,
		"int_float_f": 66.66,
	})
}

func TestSplitWords(t *testing.T) {
	valueExpected := map[string][]string{
		"FirstDay":     {"First", "Day"},
		"Firstday":     {"Firstday"},
		"FirstOneDay":  {"First", "One", "Day"},
		"First123Day":  {"First123", "Day"},
		"First_OneDay": {"First_", "One", "Day"},
	}
	for v, e := range valueExpected {
		words := splitWords(v)
		if len(words) != len(e) {
			t.Errorf("word[%s] spilit is not expected[%s]", v, e)
		}
		for i := 0; i != len(words); i++ {
			if words[i] != e[i] {
				t.Errorf("word[%s] spilit is not expected[%s]", v, e)
			}
		}
	}
}

func assertMap(t *testing.T, result map[string]interface{}, expected map[string]interface{}) {
	if len(result) != len(expected) {
		t.Fatal("map len is not equal")
		return
	}
	for k, v := range result {
		ev, ok := expected[k]
		if !ok {
			t.Fatalf("expected key[%s] is not found", k)
		}
		if v != ev {
			t.Fatalf("expected key[%s] value[%v] is not equal[%v]", k, v, ev)
		}
	}
}
