package kv

import (
	"testing"
	"time"
)

func TestUnmarshalInterfaceField(t *testing.T) {
	type InterfaceField struct {
		V interface{} `kv:"v"`
	}
	obj := &InterfaceField{}
	m := NewMapper()
	if err := m.MapToObject(map[string]interface{}{
		"v": "string_value",
	}, obj); err != nil {
		t.Fatal(err)
	}
	if obj.V != "string_value" {
		t.Fatalf("interface field's value is not expected")
	}
}

func TestUnmarshalMultiplePointer(t *testing.T) {
	type MultipleLevelPointer struct {
		MPointer *****int `kv:"m_pointer"`
	}
	obj := &MultipleLevelPointer{}
	m := NewMapper()
	if err := m.MapToObject(map[string]interface{}{
		"m_pointer": 3,
	}, obj); err != nil {
		t.Fatal(err)
	}
	if *****obj.MPointer != 3 {
		t.Fatalf("multiple pointer value is not expected")
	}
}

func TestUnmarshalNestStruct(t *testing.T) {
	type NestLevel2Object struct {
		NS string `kv:"ns"`
	}
	type NestObject struct {
		NS  string            `kv:"ns"`
		L2  NestLevel2Object  `kv:"l2"`
		L2P *NestLevel2Object `kv:"l2p"`
	}
	type Object struct {
		Nest      NestObject         `kv:"nest"`
		LevelNest NestObject         `kv:"level_nest"`
		PNest     *NestObject        `kv:"p_nest"`
		MPNest    ********NestObject `kv:"mp_nest"`
	}
	m := NewMapper()
	obj := &Object{}
	if err := m.MapToObject(map[string]interface{}{
		"nest_ns":           "ns",
		"level_nest_ns":     "l1_ns",
		"level_nest_l2_ns":  "l2_ns",
		"level_nest_l2p_ns": "l2p_ns",
		"p_nest_ns":         "p_ns",
		"mp_nest_ns":        "mp_ns",
	}, obj); err != nil {
		t.Fatal(err)
	}
	if obj.Nest.NS != "ns" {
		t.Fatalf("Nest field is not expected")
	}
	if obj.LevelNest.NS != "l1_ns" ||
		obj.LevelNest.L2.NS != "l2_ns" ||
		obj.LevelNest.L2P.NS != "l2p_ns" {
		t.Fatalf("Level Nest field is not expected")
	}
	if obj.PNest == nil || obj.PNest.NS != "p_ns" {
		t.Fatalf("PNest field is not expected")
	}
	if obj.MPNest == nil || (********obj.MPNest).NS != "mp_ns" {
		t.Fatalf("MPNest field is not expected")
	}
}

func TestUnmarshalMap(t *testing.T) {
	m := NewMapper()
	type NestObject struct {
		NS string `kv:"ns"`
	}
	type Object struct {
		Map       map[string]interface{} `kv:"map"`
		ObjectMap map[string]NestObject  `kv:"obj_map"`
	}
	obj := &Object{
		Map: map[string]interface{}{
			"exists": "init",
		},
	}
	kv := map[string]interface{}{
		"map_exists":    "recover",
		"map_string":    "string_value",
		"map_int":       99,
		"obj_map_11_ns": "11",
		"obj_map_22_ns": "22",
	}
	if err := m.MapToObject(kv, obj); err != nil {
		t.Fatal(err)
	}
	assertMap(t, obj.Map, map[string]interface{}{
		"exists": "recover",
		"string": "string_value",
		"int":    99,
	})
	if obj.ObjectMap["11"].NS != "11" ||
		obj.ObjectMap["22"].NS != "22" {
		t.Fatal("ObjectMap is not expected")
	}
}

func TestUnmarshalSlice(t *testing.T) {
	m := NewMapper()
	type NestObject struct {
		NS string `kv:"ns"`
	}
	type NestArrayObject struct {
		Objects []NestObject `kv:"objects"`
	}
	type Object struct {
		SliceString            []string          `kv:"slice_string"`
		SliceNestObject        []NestObject      `kv:"slice_nest_object"`
		SliceNestObjectPointer []*NestObject     `kv:"slice_nest_object_pointer"`
		SliceMap               []map[string]int  `kv:"slice_map"`
		SliceNestSlice         []NestArrayObject `kv:"slice_nest_slice"`
	}
	obj := &Object{}
	kv := map[string]interface{}{
		"slice_string_1":                  "string_1",
		"slice_string_2":                  "string_2",
		"slice_string_3":                  "string_3",
		"slice_nest_object_1_ns":          "ns_1",
		"slice_nest_object_2_ns":          "ns_2",
		"slice_nest_object_pointer_1_ns":  "pns_1",
		"slice_nest_object_pointer_2_ns":  "pns_2",
		"slice_map_1_111":                 111,
		"slice_map_2_222":                 222,
		"slice_nest_slice_1_objects_1_ns": "object_ns_1",
		"slice_nest_slice_1_objects_2_ns": "object_ns_2",
		"slice_nest_slice_2_objects_1_ns": "object_ns_3",
		"slice_nest_slice_2_objects_2_ns": "object_ns_4",
	}
	if err := m.MapToObject(kv, obj); err != nil {
		t.Fatal(err)
	}
	if obj.SliceString[0] != "string_1" ||
		obj.SliceString[1] != "string_2" ||
		obj.SliceString[2] != "string_3" {
		t.Fatalf("slice string is not unexpected")
	}
	if obj.SliceNestObject[0].NS != "ns_1" ||
		obj.SliceNestObject[1].NS != "ns_2" {
		t.Fatalf("slice nest object is not unexpected")
	}
	if obj.SliceNestObjectPointer[0].NS != "pns_1" ||
		obj.SliceNestObjectPointer[1].NS != "pns_2" {
		t.Fatalf("slice nest object pointer is not unexpected")
	}
	if obj.SliceMap[0]["111"] != 111 ||
		obj.SliceMap[1]["222"] != 222 {
		t.Fatalf("slice map is not unexpected")
	}
	if obj.SliceNestSlice[0].Objects[0].NS != "object_ns_1" ||
		obj.SliceNestSlice[0].Objects[1].NS != "object_ns_2" ||
		obj.SliceNestSlice[1].Objects[0].NS != "object_ns_3" ||
		obj.SliceNestSlice[1].Objects[1].NS != "object_ns_4" {
		t.Fatalf("slice nest slice is not unexpected")
	}
}

func TestUnMarshalTimeWithFormatTag(t *testing.T) {
	m := NewMapper()
	type Object struct {
		Time1s time.Time `kv:"time_with_format_tag_1s,time_fmt=trunc:1s"`
		Time5m time.Time `kv:"time_with_format_tag_5m,time_fmt=trunc:5m"`
		Time1h time.Time `kv:"time_with_format_tag_1h,time_fmt=trunc:1h"`
		Time1d time.Time `kv:"time_with_format_tag_1d,time_fmt=trunc:24h"`
		Time2d time.Time `kv:"time_with_format_tag_2d,time_fmt=trunc:48h"`
	}
	tm, _ := time.Parse("2006-01-02 15:04:05.999", "2024-05-20 17:23:52.345")
	excepted1s, _ := time.Parse("2006-01-02 15:04:05", "2024-05-20 17:23:52")
	excepted5m, _ := time.Parse("2006-01-02 15:04:05", "2024-05-20 17:20:00")
	excepted1h, _ := time.Parse("2006-01-02 15:04:05", "2024-05-20 17:00:00")
	excepted1d, _ := time.Parse("2006-01-02 15:04:05", "2024-05-20 00:00:00")
	excepted2d, _ := time.Parse("2006-01-02 15:04:05", "2024-05-19 00:00:00")
	
	obj := &Object{}
	if err := m.MapToObject(map[string]interface{}{
		"time_with_format_tag_1s": tm,
		"time_with_format_tag_5m": tm,
		"time_with_format_tag_1h": tm,
		"time_with_format_tag_1d": tm,
		"time_with_format_tag_2d": tm,
	}, obj); err != nil {
		t.Fatal(err)
	}
	if !obj.Time1s.Equal(excepted1s) ||
		!obj.Time5m.Equal(excepted5m) ||
		!obj.Time1h.Equal(excepted1h) ||
		!obj.Time1d.Equal(excepted1d) ||
		!obj.Time2d.Equal(excepted2d) {
		t.Fatalf("time is not expected")
	}
}
