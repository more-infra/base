package kv

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// Mapper helps transform a struct object to a map[string]interface{} with custom options.
// It supports nested map, slice, array and struct, syntax is similar as json/yaml tag used by Marshal.
// The format key could be self-defined by options. See WithXXX functions for more details.
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

// ObjectToMap is the entry method for Mapper.Call this method to convert object to a map[string]interface{}.
// It will process pointer, interface type by auto dereference, when tag "omitempty" is defined, it will ignore the field when it's nil.
func (m *Mapper) ObjectToMap(obj interface{}) map[string]interface{} {
	return m.structToMap(obj)
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

func (m *Mapper) structToMap(obj interface{}) map[string]interface{} {
	if obj == nil {
		return make(map[string]interface{})
	}
	elm := reflect.ValueOf(obj)
	for t := elm.Type().Kind(); t == reflect.Pointer || t == reflect.Interface; {
		if elm.IsZero() {
			return make(map[string]interface{})
		}
		elm = elm.Elem()
		t = elm.Type().Kind()
	}
	t := elm.Type()
	if t.Kind() != reflect.Struct {
		return make(map[string]interface{})
	}
	kv := make(map[string]interface{})
	for n := 0; n != t.NumField(); n++ {
		fieldType := t.Field(n)
		meta := m.parseMeta(fieldType)
		if len(meta.key) == 0 {
			continue
		}
		m.handleField(&context{
			kv:    kv,
			meta:  meta,
			value: elm.Field(n),
		})
	}
	return kv
}

func (m *Mapper) handleField(ctx *context) {
	for t := ctx.value.Type().Kind(); t == reflect.Pointer || t == reflect.Interface; {
		if ctx.value.IsZero() {
			if !ctx.meta.omitempty {
				ctx.kv[ctx.meta.key] = nil
			}
			return
		}
		ctx.value = ctx.value.Elem()
		t = ctx.value.Type().Kind()
	}
	switch ctx.value.Type().Kind() {
	case reflect.Struct:
		m.handleStruct(ctx)
	case reflect.Map:
		if !ctx.value.IsNil() {
			m.handleMap(ctx)
		}
	case reflect.Slice:
		if ctx.value.IsNil() {
			if !ctx.meta.omitempty {
				ctx.kv[ctx.meta.key] = nil
			}
		} else {
			m.handleSlice(ctx)
		}
	case reflect.Array:
		m.handleArray(ctx)
	case reflect.Chan:
		return
	case reflect.Func:
		return
	default:
		m.handleBasic(ctx)
	}
}

func (m *Mapper) handleStruct(ctx *context) {
	fieldKV := m.structToMap(ctx.value.Interface())
	for k, v := range fieldKV {
		ctx.kv[ctx.meta.key+m.nestConcat+k] = v
	}
}

func (m *Mapper) handleMap(ctx *context) {
	for _, key := range ctx.value.MapKeys() {
		v := ctx.value.MapIndex(key)
		m.handleField(&context{
			kv: ctx.kv,
			meta: &fieldMeta{
				t:         v.Type(),
				key:       ctx.meta.key + m.nestConcat + key.String(),
				omitempty: false,
			},
			value: v,
		})
	}
}

func (m *Mapper) handleSlice(ctx *context) {
	for i := 0; i != ctx.value.Len(); i++ {
		v := ctx.value.Index(i)
		key := func() string {
			if i == 0 {
				return ctx.meta.key
			}
			return fmt.Sprintf("%s%s%d", ctx.meta.key, m.sliceOrderConcat, i+1)
		}()
		m.handleField(&context{
			kv: ctx.kv,
			meta: &fieldMeta{
				t:         v.Type(),
				key:       key,
				omitempty: false,
			},
			value: v,
		})
	}
}

func (m *Mapper) handleArray(ctx *context) {
	m.handleSlice(ctx)
}

func (m *Mapper) handleBasic(ctx *context) {
	if !ctx.value.IsZero() || !ctx.meta.omitempty {
		ctx.kv[ctx.meta.key] = ctx.value.Interface()
	}
}

func (m *Mapper) parseMeta(field reflect.StructField) *fieldMeta {
	tag := field.Tag.Get(m.tagName)
	va := strings.Split(tag, ",")
	key := va[0]
	if len(key) == 0 {
		key = m.emptyTagKeyName(field)
	} else {
		if m.ignoreTagKey[key] {
			key = ""
		}
	}
	meta := &fieldMeta{
		t:   field.Type,
		key: key,
	}
	for i := 1; i != len(va); i++ {
		switch va[i] {
		case "omitempty":
			meta.omitempty = true
		}
	}
	return meta
}

func (m *Mapper) emptyTagKeyName(field reflect.StructField) string {
	switch m.emptyTagFormat {
	case Ignore:
		return ""
	case OriginFormat:
		return field.Name
	case CamelCaseFormat:
		return strings.Join(splitWords(field.Name), "")
	case UnderScoreCaseFormat:
		words := splitWords(field.Name)
		for n, w := range words {
			words[n] = strings.ToLower(w)
		}
		return strings.Join(words, "_")
	default:
		panic(fmt.Sprintf("unknown EmptyTagFormat value: %s", m.emptyTagFormat.String()))
	}
}

type fieldMeta struct {
	t         reflect.Type
	key       string
	omitempty bool
}

type context struct {
	kv    map[string]interface{}
	meta  *fieldMeta
	value reflect.Value
}

func splitWords(w string) []string {
	var (
		words   []string
		builder strings.Builder
	)
	i := 0
	for _, r := range w {
		if i == 0 {
			builder.WriteByte(byte(r))
			i++
			continue
		}
		if unicode.IsUpper(r) {
			words = append(words, builder.String())
			builder.Reset()
			builder.WriteByte(byte(r))
			continue
		}
		builder.WriteByte(byte(r))
	}
	if builder.Len() != 0 {
		words = append(words, builder.String())
	}
	return words
}
