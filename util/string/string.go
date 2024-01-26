package stringutil

import (
	"errors"
	"strconv"
	"time"
)

func ToString(v interface{}) (string, error) {
	switch v := v.(type) {
	case string:
		return v, nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint:
		// TODO: 'uint' should be converted to writing as an unsigned integer,
		// but we cannot since that would break backwards compatibility.
		return strconv.FormatUint(uint64(v), 10), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case bool:
		return strconv.FormatBool(v), nil
	case []byte:
		return string(v), nil
	case time.Duration:
		return v.String(), nil
	case time.Time:
		return v.String(), nil
	case error:
		return v.Error(), nil
	case nil:
		return "nil", nil
	default:
		return "", errors.New("could not convert to string type")
	}
}
