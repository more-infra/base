package tjson

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type Number struct {
	n int64
	f float64
}

func (b *Number) String() string {
	return strconv.FormatInt(b.n, 10)
}

func (b *Number) Int64() int64 {
	return b.n
}

func (b *Number) Int32() int32 {
	return int32(b.n)
}

func (b *Number) Int() int {
	return int(b.n)
}

func (b *Number) Float64() float64 {
	return b.f
}

func (b *Number) Float32() float32 {
	return float32(b.f)
}

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
				return fmt.Errorf("parse to integer or float both invalid, integer:%s, float64:%s",
					errInt.Error(), errFloat.Error())
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
	default:
		return fmt.Errorf("unsupported type: %T UnmarshalJSON to Number", v)
	}
	if b.f != 0 {
		b.n = int64(b.f)
	} else {
		b.f = float64(b.n)
	}
	return nil
}
