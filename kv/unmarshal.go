package kv

import (
	"fmt"
	"github.com/more-infra/base"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func (m *Mapper) mapToObject(kv map[string]interface{}, obj interface{}) error {
	if obj == nil {
		return nil
	}
	v := reflect.ValueOf(obj)
	kind := v.Type().Kind()
	if kind != reflect.Pointer && kind != reflect.Interface {
		return nil
	}
	elm := v.Elem()
	t := elm.Type()
	if t.Kind() != reflect.Struct {
		return base.NewErrorWithType(ErrTypeUnmarshalInvalidType, ErrObjectInvalidType)
	}
	for n := 0; n != t.NumField(); n++ {
		fieldType := t.Field(n)
		meta := m.parseMeta(fieldType)
		if len(meta.key) == 0 {
			continue
		}
		if err := m.unmarshalField(&context{
			kv:    kv,
			meta:  meta,
			value: elm.Field(n),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m *Mapper) unmarshalField(ctx *context) error {
	t := ctx.value.Type()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		return m.unmarshalStruct(ctx)
	case reflect.Map:
		return m.unmarshalMap(ctx)
	case reflect.Slice:
		return m.unmarshalSlice(ctx)
	case reflect.Array:
		return m.unmarshalArray(ctx)
	case reflect.Chan:
		fallthrough
	case reflect.Func:
		return base.NewErrorWithType(ErrTypeUnmarshalInvalidType, ErrUnsupportedFieldType).
			WithField("field.name", ctx.meta.key).
			WithField("field.type", ctx.meta.t.String())
	default:
		return m.unmarshalBasic(ctx)
	}
}

func (m *Mapper) unmarshalStruct(ctx *context) error {
	if len(prefixIncludeKeys(ctx.kv, ctx.meta.key+m.nestConcat, func(s string) bool {
		return len(s) > 0
	})) == 0 {
		return nil
	}
	val := m.newValueIfNilPointer(ctx.value)
	t := val.Type()
	for n := 0; n != t.NumField(); n++ {
		fieldType := t.Field(n)
		meta := m.parseMeta(fieldType)
		if len(meta.key) == 0 {
			continue
		}
		meta.key = ctx.meta.key + m.nestConcat + meta.key
		if err := m.unmarshalField(&context{
			kv:    ctx.kv,
			meta:  meta,
			value: val.Field(n),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m *Mapper) unmarshalMap(ctx *context) error {
	prefix := ctx.meta.key + m.nestConcat
	keys := prefixIncludeKeys(ctx.kv, prefix, func(s string) bool {
		return len(s) != 0
	})
	if len(keys) == 0 {
		return nil
	}
	val := m.newValueIfNilPointer(ctx.value)
	if val.IsNil() {
		val.Set(reflect.MakeMap(val.Type()))
	}
	if val.Type().Key().Kind() != reflect.String {
		return nil
	}
	elmType := val.Type().Elem()
	unProcessKeys := make(map[string]bool)
	for _, k := range keys {
		unProcessKeys[k] = true
	}
	for key := range unProcessKeys {
		if elmType.Kind() == reflect.Interface {
			val.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(ctx.kv[prefix+key]))
			delete(unProcessKeys, key)
			continue
		}
		if complexType(elmType) {
			// fixme:
			// nest concat may confound with map key string if key include the concat
			i := strings.Index(key, m.nestConcat)
			if i == -1 {
				continue
			}
			key = key[:i]
			for k := range unProcessKeys {
				if strings.HasPrefix(k, key) {
					delete(unProcessKeys, k)
				}
			}
		} else {
			delete(unProcessKeys, key)
		}
		k := prefix + key
		elmVal := reflect.New(elmType).Elem()
		if err := m.unmarshalField(&context{
			kv: ctx.kv,
			meta: &fieldMeta{
				t:         elmType,
				key:       k,
				omitempty: false,
			},
			value: elmVal,
		}); err != nil {
			return err
		}
		val.SetMapIndex(reflect.ValueOf(key), elmVal)
	}
	return nil
}

func (m *Mapper) unmarshalSlice(ctx *context) error {
	prefix := ctx.meta.key
	keys := prefixIncludeKeys(ctx.kv, prefix, func(s string) bool {
		if len(s) == 0 {
			return false
		}
		if s[:1] != m.sliceOrderConcat {
			return false
		}
		if unicode.IsDigit(rune(s[1])) {
			return true
		}
		return false
	})
	if len(keys) == 0 {
		return nil
	}
	type indexKey struct {
		n int
		k string
	}
	parseIndex := func(s string) *indexKey {
		k := strings.TrimPrefix(s, m.sliceOrderConcat)
		i := strings.IndexFunc(k, func(r rune) bool {
			return !unicode.IsDigit(r)
		})
		var v string
		if i == -1 {
			v = k
		} else {
			v = k[:i]
		}
		n, _ := strconv.Atoi(v)
		return &indexKey{
			n: n,
			k: s,
		}
	}
	groupKeys := make(map[int]*indexKey)
	for _, k := range keys {
		ik := parseIndex(k)
		groupKeys[ik.n] = ik
	}
	indexKeys := make([]*indexKey, len(groupKeys))
	var n int
	for _, k := range groupKeys {
		indexKeys[n] = k
		n++
	}
	sort.Slice(indexKeys, func(i, j int) bool {
		return indexKeys[i].n < indexKeys[j].n
	})
	val := m.newValueIfNilPointer(ctx.value)
	if val.IsNil() {
		val.Set(reflect.MakeSlice(val.Type(), 0, 0))
	}
	elmType := val.Type().Elem()
	for _, ik := range indexKeys {
		elmVal := reflect.New(elmType).Elem()
		key := prefix + ik.k
		if complexType(elmType) {
			key = fmt.Sprintf("%s%s%d", prefix, m.sliceOrderConcat, ik.n)
		}
		if err := m.unmarshalField(&context{
			kv: ctx.kv,
			meta: &fieldMeta{
				t:         elmType,
				key:       key,
				omitempty: false,
			},
			value: elmVal,
		}); err != nil {
			return err
		}
		val.Set(reflect.Append(val, elmVal))
	}
	return nil
}

func (m *Mapper) unmarshalArray(ctx *context) error {
	return m.unmarshalSlice(ctx)
}

func (m *Mapper) unmarshalBasic(ctx *context) error {
	v, ok := ctx.kv[ctx.meta.key]
	if !ok {
		return nil
	}
	val := m.newValueIfNilPointer(ctx.value)
	val.Set(reflect.ValueOf(v))
	return nil
}

func (m *Mapper) newValueIfNilPointer(val reflect.Value) reflect.Value {
	for val.Type().Kind() == reflect.Pointer {
		if val.IsNil() {
			newVal := reflect.New(val.Type().Elem())
			val.Set(newVal)
			val = val.Elem()
		} else {
			break
		}
	}
	return val
}

func prefixIncludeKeys(kv map[string]interface{}, prefix string, validator func(string) bool) []string {
	var keys []string
	for k := range kv {
		if strings.HasPrefix(k, prefix) {
			key := k[len(prefix):]
			if validator(key) {
				keys = append(keys, k[len(prefix):])
			}
		}
	}
	return keys
}

var complexTypes = map[reflect.Kind]bool{
	reflect.Struct: true,
	reflect.Slice:  true,
	reflect.Map:    true,
	reflect.Array:  true,
}

func complexType(t reflect.Type) bool {
	if t.Kind() == reflect.Interface {
		panic("complex type check could not input interface type")
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return complexTypes[t.Kind()]
}
