package tjson

import (
	"encoding/json"
	"testing"
)

func TestBoolean_UnmarshalJSON(t *testing.T) {
	type Data struct {
		EnableStringTrue  Boolean `json:"enable_string_true"`
		EnableStringFalse Boolean `json:"enable_string_false"`
		EnableTrue        Boolean `json:"enable_true"`
		EnableFalse       Boolean `json:"enable_false"`
		EnableEmpty       Boolean `json:"enable_empty"`
	}
	var d Data
	if err := json.Unmarshal([]byte(
		`{"enable_string_true":"true","enable_string_false":"false","enable_true":true,"enable_false":"false","enable_empty":""}`), &d); err != nil {
		t.Fatal(err)
	}
	if !d.EnableStringTrue.Bool() {
		t.Fatal("enable_string_true is not expected")
	}
	if d.EnableStringFalse.Bool() {
		t.Fatal("enable_string_false is not expected")
	}
	if !d.EnableTrue.Bool() {
		t.Fatal("enable_true is not expected")
	}
	if d.EnableFalse.Bool() {
		t.Fatal("enable_false is not expected")
	}
	if d.EnableEmpty.Bool() {
		t.Fatal("enable_empty is not expected")
	}
}
