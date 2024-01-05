package base

import (
	"fmt"
	stringutil "github.com/more-infra/base/util/string"
	"runtime/debug"
	"strings"
)

const (
	// ErrTypeUnknown is the default value for Error's Type field,
	// set a custom Type By NewErrorWithType or NewErrorType,WithType is recommended
	ErrTypeUnknown = "unknown"

	// ErrTypeConfig is a common error type using when the error is happened by config paring or checking.
	ErrTypeConfig = "config"
)

// Error struct is the basic error wrapper struct, which is widely used in projects.
// This error object is used as standard error return value of functions in projects.
// Functions in projects should return this error object instead of others.
// Use base.NewError or base.NewErrorWithType wraps the third module's error return which is not a base.Error typed object.
// Use base.WrapError add more information to the Error object when the function return is a base.Error typed object.
// The base.Error could format in struct object(k-v format) to backend database which could present by UI.
type Error struct {
	// Type defines the error's type as a self-define string, it will be set to "unknown" as default.
	// Using NewErrorWithType or NewError,WithType to create *base.Error is recommended.
	// This field could split errors by categories, it's a required field.
	// When do querying or present errors with UI, Type field is also the important tag or condition.
	Type string

	// Err is the error object return by functions, it's not typed base.Error as usual.
	Err error

	// Labels defines many labels of the error object, when used in searching scenes.
	// It will be added when call WithLabel method.
	// When do formatting, it will be split with ",".
	Labels []string

	// Msg defines additional comment for the error.
	// It will be added when call WithMessage method.
	// When do formatting, it will be split with "\n".
	Msg []string

	// Stack is the debug.stack information when the error is happened.
	// It will be set when call WithStack method
	Stack string

	// Fields is the additional params information of then error.
	// It will be added when call WithField or WithFields.
	// When do formatting to text, it will be joined with "=" and split with ",", such as k1=v1,k2=v2.
	// When do formatting to backend database, it will be marshals to json format.
	Fields map[string]interface{}
}

// NewErrorWithType create an error object wrapped input err with the given Type field.
// This function is used to create error in projects typically.
// A clear type for the error is very important, see the Type field comments.
func NewErrorWithType(t string, err error) *Error {
	return &Error{
		Type:   t,
		Err:    err,
		Fields: make(map[string]interface{}),
	}
}

// NewError create an error object wrapped input err,
// default Type is "unknown", you can set the Type by WithType method later.
// Using NewErrorWithType is more recommend than NewError, because a clear Type value is very important.
func NewError(err error) *Error {
	return &Error{
		Type:   ErrTypeUnknown,
		Err:    err,
		Fields: make(map[string]interface{}),
	}
}

// NewConfigError create an error object with "config" Type
func NewConfigError(err error) *Error {
	return NewErrorWithType(ErrTypeConfig, err)
}

// WrapError uses to create a new error object or wrapped exist *base.Error.
// It's used for wrapping the exists *base.Error object typically.
// When the input err object is not type of *base.Error, it will create a new error object with "unknown" Type,
// it should be set the custom Type with WithType method.
func WrapError(err error) *Error {
	e, ok := err.(*Error)
	if ok {
		return e.Clone()
	}
	return NewErrorWithType(ErrTypeUnknown, err)
}

// ErrorType return Type field of the Error object, it will return "unknown" when the input is not type of *base.Error
func ErrorType(err error) string {
	e, ok := err.(*Error)
	if !ok {
		return ErrTypeUnknown
	}
	return e.Type
}

// OriginalError return Err field of the Error object, it will return the input object self when it's not type of *base.Error
func OriginalError(err error) error {
	e, ok := err.(*Error)
	if !ok {
		return e
	}
	return e.Err
}

func (e *Error) WithType(t string) *Error {
	e.Type = t
	return e
}

func (e *Error) WithField(k string, v interface{}) *Error {
	str, err := stringutil.ToString(v)
	if err != nil {
		str = fmt.Sprintf("%+v", v)
	}
	e.Fields[k] = str
	return e
}

func (e *Error) WithFields(kv map[string]interface{}) *Error {
	for k, v := range kv {
		str, err := stringutil.ToString(v)
		if err != nil {
			str = fmt.Sprintf("%+v", v)
		}
		e.Fields[k] = str
	}
	return e
}

func (e *Error) WithStack() *Error {
	e.Stack = string(debug.Stack())
	return e
}

func (e *Error) WithMessage(msg string) *Error {
	e.Msg = append(e.Msg, msg)
	return e
}

func (e *Error) WithLabel(l string) *Error {
	e.Labels = append(e.Labels, l)
	return e
}

// Error defines the standard interface for error
func (e *Error) Error() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("err=%s", e.Err.Error()))
	if len(e.Msg) != 0 {
		builder.WriteString(fmt.Sprintf(", msg=%s", e.Message()))
	}
	builder.WriteString(fmt.Sprintf("labels=%s", strings.Join(e.Labels, ",")))
	for k, v := range e.Fields {
		builder.WriteString(fmt.Sprintf(", %s=%+v", k, v))
	}
	if len(e.Stack) != 0 {
		builder.WriteString(fmt.Sprintf(", stack:%s", e.Stack))
	}
	return builder.String()
}

func (e *Error) Message() string {
	return strings.Join(e.Msg, "\n")
}

func (e *Error) Clone() *Error {
	fields := make(map[string]interface{})
	for k, v := range e.Fields {
		fields[k] = v
	}
	return &Error{
		Type:   e.Type,
		Err:    e.Err,
		Labels: e.Labels,
		Msg:    e.Msg,
		Stack:  e.Stack,
		Fields: fields,
	}
}
