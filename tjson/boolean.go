package tjson

import (
	"encoding/json"
	"github.com/more-infra/base"
	"reflect"
	"strconv"
	"strings"
)

// Boolean supports json.Marshaler and json.Unmarshaler by string "true" or "false" or bool type true,false
type Boolean bool

// Bool returns the bool type value of the Boolean
func (b *Boolean) Bool() bool {
	return (bool)(*b)
}

// String returns the string type value of the Boolean
func (b *Boolean) String() string {
	return strconv.FormatBool((bool)(*b))
}

// UnmarshalJSON implements the json.Unmarshaler interface
// It supports the following types:
// - string: "true" or "false"
// - bool: true or false
// - nil: false
// - other: return error
func (b *Boolean) UnmarshalJSON(data []byte) error {
	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	switch v := temp.(type) {
	case string:
		v = strings.ToLower(v)
		switch v {
		case "true":
			*b = true
		case "false":
			*b = false
		case "":
			*b = false
		default:
			return base.NewErrorWithType(ErrTypeBooleanUnmarshalFailed, ErrBooleanTypeStringInvalid).
				WithField("string.value", v)
		}
	case bool:
		if v {
			*b = true
		} else {
			*b = false
		}
	case nil:
		*b = false
	default:
		return base.NewErrorWithType(ErrTypeBooleanUnmarshalFailed, ErrBooleanTypeUnSupported).
			WithField("value.type", reflect.TypeOf(v).String())
	}
	return nil
}
