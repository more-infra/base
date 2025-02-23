package tjson

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimeMarshalJSON(t *testing.T) {
	type Data struct {
		Nano             Time  `json:"nano"`
		Micro            Time  `json:"micro"`
		Milli            Time  `json:"milli"`
		Second           Time  `json:"second"`
		DateTime         Time  `json:"datetime"`
		RFC3339          Time  `json:"rfc3339"`
		RFC3339Nano      Time  `json:"rfc3339nano"`
	}
	var d = Data{
		Nano:     *NewTime(WithFormat(time.RFC3339Nano), WithTime(time.Date(2025, 2, 22, 15, 12, 33, 123456789, time.UTC))),
		Micro:    *NewTime(WithFormat(time.RFC3339Nano), WithTime(time.Date(2025, 2, 22, 15, 12, 33, 123456, time.UTC))),
		Milli:    *NewTime(WithFormat(time.RFC3339Nano), WithTime(time.Date(2025, 2, 22, 15, 12, 33, 123, time.UTC))),
		Second:   *NewTime(WithFormat("2006-01-02X15:04:05TTT"), WithTime(time.Date(2025, 2, 22, 15, 12, 33, 0, time.UTC))),
		DateTime: *NewTime(WithFormat(time.DateTime), WithTime(time.Date(2025, 2, 22, 15, 12, 33, 0, time.UTC))),
		RFC3339:  *NewTime(WithFormat(time.RFC3339), WithTime(time.Date(2025, 2, 22, 15, 12, 33, 0, time.UTC))),
		RFC3339Nano: *NewTime(WithFormat(time.RFC3339Nano), WithTime(time.Date(2025, 2, 22, 15, 12, 33, 123456789, time.UTC))),
	}
	b, err := json.Marshal(&d)
	if err != nil {
		t.Fatal(err)
	}
	keyValues := make(map[string]interface{})
	if err := json.Unmarshal(b, &keyValues); err != nil {
		t.Fatal(err)
	}
	if keyValues["nano"] != "2025-02-22T15:12:33.123456789Z" {
		t.Fatal("nano is not expected", keyValues["nano"])
	}
	if keyValues["micro"] != "2025-02-22T15:12:33.000123456Z" {
		t.Fatal("micro is not expected", keyValues["micro"])
	}
	if keyValues["milli"] != "2025-02-22T15:12:33.000000123Z" {
		t.Fatal("milli is not expected", keyValues["milli"])
	}
	if keyValues["second"] != "2025-02-22X15:12:33TTT" {
		t.Fatal("second is not expected", keyValues["second"])
	}
	if keyValues["datetime"] != "2025-02-22 15:12:33" {
		t.Fatal("datetime is not expected", keyValues["datetime"])
	}
	if keyValues["rfc3339"] != "2025-02-22T15:12:33Z" {
		t.Fatal("rfc3339 is not expected", keyValues["rfc3339"])
	}
	if keyValues["rfc3339nano"] != "2025-02-22T15:12:33.123456789Z" {
		t.Fatal("rfc3339nano is not expected", keyValues["rfc3339nano"])
	}
}

func TestTimeUnmarshalJSON(t *testing.T) {
	type Data struct {
		Nano             Time  `json:"nano"`
		Micro            Time  `json:"micro"`
		Milli            Time  `json:"milli"`
		Second           Time  `json:"second"`
		DateTime         Time  `json:"datetime"`
		RFC3339          Time  `json:"rfc3339"`
		RFC3339Nano      Time  `json:"rfc3339nano"`
	}
	var d = Data{
		DateTime: Time{
			f: time.DateTime,
		},
		RFC3339: Time{
			f: time.RFC3339,
		},
		RFC3339Nano: Time{
			f: time.RFC3339Nano,
		},
	}
	if err := json.Unmarshal([]byte(
		`{"nano":1740270753000000000,"micro":1740270753000000,"milli":1740270753000,"second":1740270753,"datetime":"2025-02-22 15:12:33","rfc3339":"2025-02-22T15:12:33Z", "rfc3339nano":"2025-02-22T15:12:33.123456789Z"}`), &d); err != nil {
		t.Fatal(err)
	}
	if d.Nano.UnixNano() != 1740270753000000000 {
		t.Fatal("Nano is not expected")
	}
	if d.Micro.UnixMicro() != 1740270753000000 {
		t.Fatal("Micro is not expected")
	}
	if d.Milli.UnixMilli() != 1740270753000 {
		t.Fatal("Milli is not expected")
	}
	if d.Second.Unix() != 1740270753 {
		t.Fatal("Second is not expected")
	}
	if d.DateTime.Format(time.DateTime) != "2025-02-22 15:12:33" {
		t.Fatal("DateTime is not expected")
	}
	if d.RFC3339.Format(time.RFC3339) != "2025-02-22T15:12:33Z" {
		t.Fatal("RFC3339 is not expected")
	}
	if v := d.RFC3339Nano.Format(time.RFC3339Nano); v != "2025-02-22T15:12:33.123456789Z" {
		t.Fatal("RFC3339Nano is not expected", v)
	}
}
