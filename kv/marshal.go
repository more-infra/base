package kv

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"
)

func (m *Mapper) objectToMap(obj interface{}) map[string]interface{} {
	if obj == nil {
		return map[string]interface{}{}
	}
	kv := make(map[string]interface{})
	m.handleField(&context{
		kv: kv,
		meta: &fieldMeta{
			t:         reflect.TypeOf(obj),
			key:       "",
			omitempty: true,
		},
		value: reflect.ValueOf(obj),
	})
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

	var (
		v          = ctx.value.Interface()
		marshaller MapperMarshaller
		ok         bool
	)
	if ctx.value.CanAddr() {
		marshaller, ok = ctx.value.Addr().Interface().(MapperMarshaller)
	}
	if !ok {
		marshaller, ok = v.(MapperMarshaller)
	}
	if ok {
		ctx.value = reflect.ValueOf(marshaller.MapperMarshal())
		ctx.meta.t = ctx.value.Type()
		m.handleField(ctx)
		return
	}

	switch v.(type) {
	case time.Time:
		m.handleBasic(ctx)
		return
	case time.Duration:
		m.handleBasic(ctx)
		return
	default:
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
	t := ctx.value.Type()
	for n := 0; n != t.NumField(); n++ {
		fieldType := t.Field(n)
		meta := m.parseMeta(fieldType)
		if len(meta.key) == 0 {
			continue
		}
		k := ctx.meta.key
		if len(k) != 0 {
			k += m.nestConcat + meta.key
		} else {
			k = meta.key
		}
		m.handleField(&context{
			kv: ctx.kv,
			meta: &fieldMeta{
				t:         fieldType.Type,
				key:       k,
				omitempty: meta.omitempty,
			},
			value: ctx.value.Field(n),
		})
	}
}

func (m *Mapper) handleMap(ctx *context) {
	for _, key := range ctx.value.MapKeys() {
		v := ctx.value.MapIndex(key)
		k := key.String()
		if len(ctx.meta.key) != 0 {
			k = ctx.meta.key + m.nestConcat + k
		}
		m.handleField(&context{
			kv: ctx.kv,
			meta: &fieldMeta{
				t:         v.Type(),
				key:       k,
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
			if len(ctx.meta.key) == 0 {
				return fmt.Sprintf("%d", i+1)
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
