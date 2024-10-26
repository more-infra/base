package tjson

import (
	"encoding/json"
	"testing"
)

func TestNumberUnmarshalJSON(t *testing.T) {
	type Data struct {
		String           Number  `json:"string"`
		Int              Number  `json:"int"`
		Zero             Number  `json:"zero"`
		Float            Number  `json:"float"`
		FloatString      Number  `json:"float_string"`
		PointFloatString *Number `json:"point_float_string"`
		NilInt           *Number `json:"nil_int"`
	}
	var d Data
	if err := json.Unmarshal([]byte(
		`{"string":"1234","int":5678,"zero":"0","float":9.9999, "float_string":"7777.7777", "point_float_string":"66.66"}`), &d); err != nil {
		t.Fatal(err)
	}
	if d.String.Int() != 1234 {
		t.Fatal("string is not expected")
	}
	if d.Int.Int() != 5678 {
		t.Fatal("int is not expected")
	}
	if d.Zero.Int() != 0 {
		t.Fatal("zero is not expected")
	}
	if d.Float.Float64() != 9.9999 {
		t.Fatal("float is not expected")
	}
	if d.FloatString.Float64() != 7777.7777 {
		t.Fatal("float_string is not expected")
	}
	if d.PointFloatString.Float64() != 66.66 {
		t.Fatal("float_string is not expected")
	}
	if d.NilInt != nil {
		t.Fatal("nil_int is not expected")
	}
}
