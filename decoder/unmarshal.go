package decoder

import (
	"fmt"
	"reflect"
	"torantulla/pkg/bencode"
)

func decodeBytes(val interface{}) ([]byte, bool) {
	switch s := val.(type) {
	case bencode.BString:
		return []byte(s), true
	case []byte:
		return s, true
	default:
		return nil, false
	}
}

func decodeString(val interface{}) (string, bool) {
	switch s := val.(type) {
	case bencode.BString:
		return string(s), true
	case []byte:
		return string(s), true
	default:
		return "", false
	}
}

func decodeStringList(val interface{}) ([]string, bool) {
	list, ok := val.(bencode.BList)
	if !ok {
		return nil, false
	}
	var result []string
	for _, item := range list {
		s, ok := decodeString(item)
		if !ok {
			continue
		}
		result = append(result, s)
	}
	return result, true
}

func decodeNestedStringList(val interface{}) ([][]string, bool) {
	outer, ok := val.(bencode.BList)
	if !ok {
		return nil, false
	}
	var result [][]string
	for _, group := range outer {
		innerList, ok := group.(bencode.BList)
		if !ok {
			continue
		}
		inner, ok := decodeStringList(innerList)
		if !ok {
			continue
		}
		result = append(result, inner)
	}
	return result, true
}

func UnmarshalBDict(dict bencode.BDict, out interface{}) error {
	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("output must be a non-nil pointer")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("output must be a pointer to struct")
	}

	t := v.Type()

	for i := range v.NumField() {
		field := v.Field(i)
		fieldType := t.Field(i)

		tag := fieldType.Tag.Get("bencode")
		if tag == "" {
			tag = fieldType.Name
		}

		val, ok := dict[tag]
		if !ok {
			continue // field is optional
		}

		if !field.CanSet() {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			if bs, ok := val.(bencode.BString); ok {
				field.SetString(string(bs))
			}
		case reflect.Int:
			if bi, ok := val.(bencode.BInt); ok {
				field.SetInt(int64(bi))
			}
		case reflect.Slice:
			switch field.Type().Elem().Kind() {
			case reflect.String:
				if slice, ok := decodeStringList(val); ok {
					field.Set(reflect.ValueOf(slice))
				}
			case reflect.Slice:
				if field.Type().Elem().Elem().Kind() == reflect.String {
					if nested, ok := decodeNestedStringList(val); ok {
						field.Set(reflect.ValueOf(nested))
					}
				}
			case reflect.Uint8:
				if bytes, ok := decodeBytes(val); ok {
					field.SetBytes(bytes)
				}
			}
		case reflect.Struct:
			if bd, ok := val.(bencode.BDict); ok {
				if err := UnmarshalBDict(bd, field.Addr().Interface()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
