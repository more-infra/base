package tjson

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/more-infra/base"
)

type Time struct {
	time.Time
	f string
}

type Option func(*Time)

func WithTime(tm time.Time) Option {
	return func(t *Time) {
		t.Time = tm
	}
}

func WithFormat(format string) Option {
	return func(t *Time) {
		t.f = format
	}
}

func NewTime(options ...Option) Time {
	time := Time{
		f:    time.DateTime,
	}
	for _, option := range options {
		option(&time)
	}
	return time
}

func (t *Time) String() string {
	return t.Format(t.format())
}

func (t *Time) format() string {
	if len(t.f) != 0 {
		return t.f
	}
	return time.DateTime
}

func (t *Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format(t.format()))
}

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
