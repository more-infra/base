package tjson

import "errors"

const (
	ErrTypeBooleanUnmarshalFailed = "boolean.unmarshal_failed"
	ErrTypeNumberUnmarshalFailed  = "number.unmarshal_failed"
	ErrTypeTimeUnmarshalFailed    = "time.unmarshal_failed"
)

var (
	ErrBooleanTypeStringInvalid = errors.New("string value is invalid for paring to Boolean")
	ErrBooleanTypeUnSupported   = errors.New("type is unsupported in Boolean.Unmarshal")
	ErrNumberTypeStringInvalid  = errors.New("string value is invalid for parsing to Number")
	ErrNumberTypeUnSupported    = errors.New("type is unsupported in Number.Unmarshal")
	ErrTimeTypeUnSupported      = errors.New("type is unsupported in Time.Unmarshal")
)