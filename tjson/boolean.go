package tjson

import (
	"encoding/json"
	"github.com/more-infra/base"
	"reflect"
	"strconv"
	"strings"
)

type Boolean bool

func (b *Boolean) Bool() bool {
	return (bool)(*b)
}

func (b *Boolean) String() string {
	return strconv.FormatBool((bool)(*b))
}

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
