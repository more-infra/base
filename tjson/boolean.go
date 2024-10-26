package tjson

import (
	"encoding/json"
	"fmt"
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
			return fmt.Errorf("invalid boolean string: %s", v)
		}
	case bool:
		if v {
			*b = true
		} else {
			*b = false
		}
	default:
		return fmt.Errorf("unsupported type: %T UnmarshalJSON to Boolean", v)
	}
	return nil
}
