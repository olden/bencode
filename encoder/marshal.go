package encoder

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func Encode(value interface{}) ([]byte, error) {
	var buf strings.Builder
	if err := encodeValue(&buf, value); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func encodeValue(buf *strings.Builder, v interface{}) error {
	switch v := v.(type) {
	case string:
		buf.WriteString(fmt.Sprintf("%d:%s", len(v), v))
	case []byte:
		buf.WriteString(fmt.Sprintf("%d:", len(v)))
		buf.Write(v)
	case int:
		buf.WriteString("i")
		buf.WriteString(strconv.Itoa(v))
		buf.WriteString("e")
	case int64:
		buf.WriteString("i")
		buf.WriteString(strconv.FormatInt(v, 10))
		buf.WriteString("e")
	case []interface{}:
		buf.WriteString("l")
		for _, item := range v {
			if err := encodeValue(buf, item); err != nil {
				return err
			}
		}
		buf.WriteString("e")
	case []string:
		buf.WriteString("l")
		for _, s := range v {
			if err := encodeValue(buf, s); err != nil {
				return err
			}
		}
		buf.WriteString("e")
	case map[string]interface{}:
		buf.WriteString("d")
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if err := encodeValue(buf, k); err != nil {
				return err
			}
			if err := encodeValue(buf, v[k]); err != nil {
				return err
			}
		}
		buf.WriteString("e")
	default:
		rv := reflect.ValueOf(v)
		rt := rv.Type()
		if rt.Kind() == reflect.Struct {
			buf.WriteString("d")
			type fieldKV struct {
				key string
				val reflect.Value
			}
			var fields []fieldKV
			for i := 0; i < rt.NumField(); i++ {
				field := rt.Field(i)
				if field.PkgPath != "" { // unexported
					continue
				}
				key := field.Tag.Get("bencode")
				if key == "" {
					key = field.Name
				}
				fields = append(fields, fieldKV{key, rv.Field(i)})
			}
			sort.Slice(fields, func(i, j int) bool {
				return fields[i].key < fields[j].key
			})
			for _, f := range fields {
				if err := encodeValue(buf, f.key); err != nil {
					return err
				}
				if err := encodeValue(buf, f.val.Interface()); err != nil {
					return err
				}
			}
			buf.WriteString("e")
			return nil
		}
		return fmt.Errorf("unsupported type: %T", v)
	}
	return nil
}
