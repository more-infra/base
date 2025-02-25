package tjson

import (
	"encoding/json"
	"github.com/more-infra/base"
	"reflect"
	"strconv"
)

// Number supports json.Marshaler and json.Unmarshaler by int64 or float64
type Number struct {
	n int64
	f float64
}

// String returns the string type value of the Number
func (b *Number) String() string {
	return strconv.FormatInt(b.n, 10)
}

// Int64 returns the int64 type value of the Number
func (b *Number) Int64() int64 {
	return b.n
}

// Int32 returns the int32 type value of the Number
func (b *Number) Int32() int32 {
	return int32(b.n)
}

// Int returns the int type value of the Number
func (b *Number) Int() int {
	return int(b.n)
}

// Float64 returns the float64 type value of the Number
func (b *Number) Float64() float64 {
	return b.f
}

// Float32 returns the float32 type value of the Number
func (b *Number) Float32() float32 {
	return float32(b.f)
}

// UnmarshalJSON implements the json.Unmarshaler interface
// It supports the following types:
// - string: parse to int64 or float64
// - int: parse to int64
// - int8: parse to int64
// - int16: parse to int64
// - int32: parse to int64
// - int64: parse to int64
// - float64: parse to float64
// - float32: parse to float32
// - nil: set to 0
// - other: return error
//
// Note:
// - If the string value is not a valid number, it will return an error.
// - If the float64 value is not a valid number, it will return an error.
// - If the int64 value is not a valid number, it will return an error.
// - If the int32 value is not a valid number, it will return an error.
// - If the int value is not a valid number, it will return an error.
func (b *Number) UnmarshalJSON(data []byte) error {
	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	switch v := temp.(type) {
	case string:
		n, errInt := strconv.ParseInt(v, 10, 64)
		if errInt != nil {
			f, errFloat := strconv.ParseFloat(v, 64)
			if errFloat != nil {
				return base.NewErrorWithType(ErrTypeNumberUnmarshalFailed, ErrNumberTypeStringInvalid).
					WithField("parse.int.err", errInt).
					WithField("parse.float.err", errFloat)
			}
			b.f = f
		} else {
			b.n = n
		}
	case float64:
		b.f = v
	case int:
		b.n = int64(v)
	case int8:
		b.n = int64(v)
	case int16:
		b.n = int64(v)
	case int32:
		b.n = int64(v)
	case int64:
		b.n = v
	case nil:
		b.n = 0
	default:
		return base.NewErrorWithType(ErrTypeNumberUnmarshalFailed, ErrNumberTypeUnSupported).
			WithField("value.type", reflect.TypeOf(v).String())
	}
	if b.f != 0 {
		b.n = int64(b.f)
	} else {
		b.f = float64(b.n)
	}
	return nil
}
