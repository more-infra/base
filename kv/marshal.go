package kv

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
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
	v := ctx.value.Interface()
	marshaller, ok := v.(MapperMarshaller)
	if ok {
		v := marshaller.MapperMarshal()
		ctx.value = reflect.ValueOf(v)
		ctx.meta.t = ctx.value.Type()
		m.handleField(ctx)
		return
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
