package encoder

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/olden/bencode"
)

// Encode returns canonical bencode for integers within int64, strings, byte
// slices/arrays, lists, string-keyed maps and structs, including named types.
// Non-nil pointers/interfaces are dereferenced. Nil maps/slices encode as empty
// containers; nil pointers/interfaces and unsupported types return errors.
// Struct tags support field names and "-"; options and duplicate names are errors.
// Container nesting is limited to bencode.MaxDepth. No partial output is returned.
func Encode(value interface{}) ([]byte, error) {
	var buf strings.Builder
	if err := encodeValue(&buf, reflect.ValueOf(value), 0); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func encodeValue(buf *strings.Builder, v reflect.Value, depth int) error {
	for hops := 0; v.IsValid() && (v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr); hops++ {
		if hops >= bencode.MaxDepth {
			return fmt.Errorf("pointer/interface chain exceeds limit %d", bencode.MaxDepth)
		}
		if v.IsNil() {
			return fmt.Errorf("cannot encode nil %s", v.Type())
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return fmt.Errorf("cannot encode nil")
	}
	switch v.Kind() {
	case reflect.String:
		writeString(buf, v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		buf.WriteByte('i')
		buf.WriteString(strconv.FormatInt(v.Int(), 10))
		buf.WriteByte('e')
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n := v.Uint()
		if n > math.MaxInt64 {
			return fmt.Errorf("integer %d exceeds int64 range", n)
		}
		buf.WriteByte('i')
		buf.WriteString(strconv.FormatUint(n, 10))
		buf.WriteByte('e')
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			buf.WriteString(strconv.Itoa(v.Len()))
			buf.WriteByte(':')
			for i := 0; i < v.Len(); i++ {
				buf.WriteByte(byte(v.Index(i).Uint()))
			}
			return nil
		}
		if depth >= bencode.MaxDepth {
			return fmt.Errorf("nesting exceeds limit %d", bencode.MaxDepth)
		}
		buf.WriteByte('l')
		for i := 0; i < v.Len(); i++ {
			if err := encodeValue(buf, v.Index(i), depth+1); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
		buf.WriteByte('e')
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("dictionary keys must be strings, got %s", v.Type().Key())
		}
		if depth >= bencode.MaxDepth {
			return fmt.Errorf("nesting exceeds limit %d", bencode.MaxDepth)
		}
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
		buf.WriteByte('d')
		for _, key := range keys {
			writeString(buf, key.String())
			if err := encodeValue(buf, v.MapIndex(key), depth+1); err != nil {
				return fmt.Errorf("key %q: %w", key.String(), err)
			}
		}
		buf.WriteByte('e')
	case reflect.Struct:
		if depth >= bencode.MaxDepth {
			return fmt.Errorf("nesting exceeds limit %d", bencode.MaxDepth)
		}
		type fieldValue struct {
			key   string
			value reflect.Value
		}
		var fields []fieldValue
		seen := make(map[string]bool)
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if field.PkgPath != "" {
				continue
			}
			key := field.Tag.Get("bencode")
			if key == "-" {
				continue
			}
			if strings.Contains(key, ",") {
				return fmt.Errorf("field %s: bencode tag options are unsupported", field.Name)
			}
			if key == "" {
				key = field.Name
			}
			if seen[key] {
				return fmt.Errorf("duplicate bencode field name %q", key)
			}
			seen[key] = true
			fields = append(fields, fieldValue{key, v.Field(i)})
		}
		sort.Slice(fields, func(i, j int) bool { return fields[i].key < fields[j].key })
		buf.WriteByte('d')
		for _, field := range fields {
			writeString(buf, field.key)
			if err := encodeValue(buf, field.value, depth+1); err != nil {
				return fmt.Errorf("field %q: %w", field.key, err)
			}
		}
		buf.WriteByte('e')
	default:
		return fmt.Errorf("unsupported type: %s", v.Type())
	}
	return nil
}

func writeString(buf *strings.Builder, value string) {
	buf.WriteString(strconv.Itoa(len(value)))
	buf.WriteByte(':')
	buf.WriteString(value)
}
