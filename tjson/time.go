package tjson

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/more-infra/base"
)

// Time supports json.Marshaler and json.Unmarshaler by string with time format such as RFC3339,DateTime
// and unix timestamp in nanosecond,microsecond,millisecond,second
type Time struct {
	time.Time
	f string
}

// Option is the option for Time New function
type Option func(*Time)

// WithTime sets the time.Time value for the Time, it's optional, the default value is zero value of time.Time
func WithTime(tm time.Time) Option {
	return func(t *Time) {
		t.Time = tm
	}
}

// WithFormat sets the format for the Time, it's optional, the default value is time.DateTime
//
// Note:
// - If the format is not set, the default value is time.DateTime
// - If the format is set, the format will be used
func WithFormat(format string) Option {
	return func(t *Time) {
		t.f = format
	}
}

// NewTime creates a new Time, options are optional
// You can use WithTime and WithFormat to set the time and format
func NewTime(options ...Option) *Time {
	time := &Time{
		f:    time.DateTime,
	}
	for _, option := range options {
		option(time)
	}
	return time
}

// String returns the string type value of the Time
func (t *Time) String() string {
	return t.Format(t.format())
}

func (t *Time) format() string {
	if len(t.f) != 0 {
		return t.f
	}
	return time.DateTime
}

// MarshalJSON implements the json.Marshaler interface
func (t *Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format(t.format()))
}

// UnmarshalJSON implements the json.Unmarshaler interface
// It supports the following types:
// - string: parse to time.Time
// - float64: parse to time.Time
// - int: parse to time.Time
// - int32: parse to time.Time
// - int64: parse to time.Time
// - nil: set to zero value of time.Time
// - other: return error
//
// Note:
// - If the string value is not a valid time, it will return an error.
// - If the float64 value is not a valid time, it will return an error.
// - If the int value is not a valid time, it will return an error.
// - If the int32 value is not a valid time, it will return an error.
// - If the int64 value is not a valid time, it will return an error.
// - If the format is not set, the default value is time.DateTime
// - If the format is set, the format will be used
func (t *Time) UnmarshalJSON(data []byte) error {
	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	switch v := temp.(type) {
	case string:
		tm, err := time.Parse(t.format(), v)
		if err != nil {
			return base.NewErrorWithType(ErrTypeTimeUnmarshalFailed, err).
				WithField("time.String", v)
		}
		t.Time = tm
	case float64, float32, int, int32, int64:
		var (
			n  int64
			tm time.Time
		)
		switch v := v.(type) {
		case int:
			n = int64(v)
		case int32:
			n = int64(v)
		case int64:
			n = v
		case float64:
			n = int64(v)
		case float32:
			n = int64(v)
		}
		if n < 2147483647 {
			tm = time.Unix(n, 0)
		} else if n < 2147483647*1000 {
			tm = time.UnixMilli(n)
		} else if n < 2147483647*1000*1000 {
			tm = time.UnixMicro(n)
		} else {
			tm = time.Unix(0, n)
		}
		t.Time = tm
	case nil:
	default:
		return base.NewErrorWithType(ErrTypeTimeUnmarshalFailed, ErrTimeTypeUnSupported).
			WithField("value.type", reflect.TypeOf(v).String())
	}
	return nil
}
