package kv

import (
	"errors"
)

const (
	ErrTypeUnmarshalInvalidType = "kv.unmarshal_invalid_type"
)

var (
	ErrObjectInvalidType    = errors.New("object to unmarshal is not struct type")
	ErrUnsupportedFieldType = errors.New("type of field to unmarshal is not supported")
)

// MapperMarshaller is used for custom defined type for marshal to a general interface{} type,
// which could be handled by Mapper. It works as the yaml.MarshalYAML interface do.
// NOTICE: Implement the MapperMarshaller interface with Non-Pointer receiver as usual,
// or it will not be called when the field pointer could not be used.
type MapperMarshaller interface {
	MapperMarshal() interface{}
}

// Mapper helps transform struct object between map[string]interface{} with custom options.
// It supports nested map, slice, array and struct, syntax is similar as json/yaml tag used by Marshal.
// The format key could be self-defined by options. See WithXXX functions for more details.
// It's used for converting from struct object and map[string]interface{} which is saved to database,
// So the elements in map[string]interface{} are basic type, not complex type(such as struct,map,slice)
type Mapper struct {
	tagName          string
	ignoreTagKey     map[string]bool
	emptyTagFormat   EmptyTagNameFormat
	nestConcat       string
	sliceOrderConcat string
}

// NewMapper make a Mapper with default options, use WithXXX set options as custom.
func NewMapper() *Mapper {
	m := &Mapper{
		tagName: "kv",
		ignoreTagKey: map[string]bool{
			"-": true,
		},
		emptyTagFormat:   Ignore,
		nestConcat:       "_",
		sliceOrderConcat: "_",
	}
	return m
}

// WithTagName set the tag name of fields, it's similar as "json", "yaml", "pb"...
// The default value is "kv".
func (m *Mapper) WithTagName(n string) *Mapper {
	m.tagName = n
	return m
}

// WithIgnoreTagKey defines the ignored field key. The key in field's tag with this value will be ignored to format.
// The default value is "-".
func (m *Mapper) WithIgnoreTagKey(t string) *Mapper {
	m.ignoreTagKey[t] = true
	return m
}

// WithEmptyTagFormat defines the field format method when the tag is not defined.
//
// Ignore means this field is not required to format.
//
// OriginFormat means use the origin field's name to format.
//
// CamelCaseFormat will use the CamelCase to format the field's name, for example, "FieldAttr1" will be formatted to "FieldAttr1"
//
// UnderScoreCaseFormat will use the UnderScoreCase to format field's name, for example, "FieldAttr1" will be formatted to "field_attr1"
func (m *Mapper) WithEmptyTagFormat(f EmptyTagNameFormat) *Mapper {
	m.emptyTagFormat = f
	return m
}

// WithNestConcat defines the nest complex field(such as struct, map, slice) concat.
//
//	type NestObject struct {
//		File string `kv:"file"`
//	}
//	type Object struct {
//		Nest NestObject `kv:"nest"`
//	}
//
//	Format WithNestConcat("_")
//		Object{Nest:{File:"example.tmp"}}
//	will return
//		map[string]interface{}{
//			"nest_file": "example.tmp"
//		}
//
// The default value is "_".
func (m *Mapper) WithNestConcat(prefix string) *Mapper {
	m.nestConcat = prefix
	return m
}

// WithSliceOrderConcat defines the nest slice elements order index concat.
//
//	type Object struct {
//		Files []string `kv:"files"`
//	}
//
//	Format WithSliceOrderConcat("_")
//		Object{Files:[]string{"a.tmp","b.mp4","c.html"}
//	will return
//		map[string]interface{}{
//			"files": "a.tmp",
//			"files_2": "b.mp4",
//			"files_3": "c.html",
//		}
//
// The default value is "_".
func (m *Mapper) WithSliceOrderConcat(concat string) *Mapper {
	m.sliceOrderConcat = concat
	return m
}

type EmptyTagNameFormat string

func (t EmptyTagNameFormat) String() string {
	return string(t)
}

const (
	Ignore               EmptyTagNameFormat = "ignore"
	OriginFormat         EmptyTagNameFormat = "origin"
	CamelCaseFormat      EmptyTagNameFormat = "camel"
	UnderScoreCaseFormat EmptyTagNameFormat = "under_score"
)

// ObjectToMap converts object to a map[string]interface{} which is used to save into database.
// It will process pointer, interface type by auto dereference, when tag "omitempty" is defined, it will ignore the field when it's nil.
func (m *Mapper) ObjectToMap(obj interface{}) map[string]interface{} {
	return m.structToMap(obj)
}

// MapToObject converts a map[string]interface{} which is from database to the object.
func (m *Mapper) MapToObject(kv map[string]interface{}, obj interface{}) error {
	return m.mapToObject(kv, obj)
}
