package decoder

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/olden/bencode"
)

// UnmarshalBDict decodes a dictionary into a non-nil pointer to a struct.
// Known fields must have compatible types; integer overflow is an error.
// Unknown keys are ignored and absent fields retain their existing values.
// Tags support field names and "-"; tag options and duplicate names are errors.
// Nested structs, pointers, typed collections and empty interfaces are supported.
// Byte strings and collections are copied. On error, out is left unchanged.
func UnmarshalBDict(dict bencode.BDict, out interface{}) error {
	v := reflect.ValueOf(out)
	if !v.IsValid() || v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("output must be a non-nil pointer")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("output must be a pointer to struct")
	}
	// Updates to pointers and collections allocate new values below, so this
	// shallow starting copy also preserves existing output on nested errors.
	result := reflect.New(v.Type()).Elem()
	result.Set(v)
	if err := decodeValue(dict, result, 0); err != nil {
		return err
	}
	v.Set(result)
	return nil
}

func decodeValue(value interface{}, dst reflect.Value, depth int) error {
	if value == nil {
		return fmt.Errorf("cannot decode nil into %s", dst.Type())
	}
	// Validate source nodes before expanding interface destinations. Otherwise
	// an unsupported pointer stored in BValue could recursively expand forever.
	switch value.(type) {
	case bencode.BInt, bencode.BString, bencode.BList, bencode.BDict, []byte:
	default:
		return fmt.Errorf("unsupported bencode value: %T", value)
	}
	for hops := 0; dst.Kind() == reflect.Ptr; hops++ {
		if hops >= bencode.MaxDepth {
			return fmt.Errorf("pointer chain exceeds limit %d", bencode.MaxDepth)
		}
		next := reflect.New(dst.Type().Elem())
		if !dst.IsNil() {
			next.Elem().Set(dst.Elem())
		}
		dst.Set(next)
		dst = next.Elem()
	}
	if dst.Kind() == reflect.Interface && dst.NumMethod() == 0 {
		concrete := reflect.New(reflect.TypeOf(value)).Elem()
		if err := decodeValue(value, concrete, depth); err != nil {
			return err
		}
		dst.Set(concrete)
		return nil
	}
	switch source := value.(type) {
	case bencode.BInt:
		switch dst.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if dst.OverflowInt(int64(source)) {
				return fmt.Errorf("integer %d overflows %s", source, dst.Type())
			}
			dst.SetInt(int64(source))
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if source < 0 || dst.OverflowUint(uint64(source)) {
				return fmt.Errorf("integer %d overflows %s", source, dst.Type())
			}
			dst.SetUint(uint64(source))
			return nil
		}
	case bencode.BString:
		return decodeBytes([]byte(source), dst)
	case []byte: // Preserve the existing convenience for manually built dictionaries.
		return decodeBytes(source, dst)
	case bencode.BList:
		if depth >= bencode.MaxDepth {
			return fmt.Errorf("nesting exceeds limit %d", bencode.MaxDepth)
		}
		if (dst.Kind() == reflect.Slice || dst.Kind() == reflect.Array) && dst.Type().Elem().Kind() != reflect.Uint8 {
			var result reflect.Value
			if dst.Kind() == reflect.Array {
				if dst.Len() != len(source) {
					return fmt.Errorf("list length %d does not match %s", len(source), dst.Type())
				}
				result = reflect.New(dst.Type()).Elem()
			} else {
				result = reflect.MakeSlice(dst.Type(), len(source), len(source))
			}
			for i, item := range source {
				if err := decodeValue(item, result.Index(i), depth+1); err != nil {
					return fmt.Errorf("[%d]: %w", i, err)
				}
			}
			dst.Set(result)
			return nil
		}
	case bencode.BDict:
		if depth >= bencode.MaxDepth {
			return fmt.Errorf("nesting exceeds limit %d", bencode.MaxDepth)
		}
		if dst.Kind() == reflect.Struct {
			return decodeStruct(source, dst, depth+1)
		}
		if dst.Kind() == reflect.Map && dst.Type().Key().Kind() == reflect.String {
			result := reflect.MakeMapWithSize(dst.Type(), len(source))
			keys := make([]string, 0, len(source))
			for key := range source {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				item := reflect.New(dst.Type().Elem()).Elem()
				if err := decodeValue(source[key], item, depth+1); err != nil {
					return fmt.Errorf("key %q: %w", key, err)
				}
				mapKey := reflect.New(dst.Type().Key()).Elem()
				mapKey.SetString(key)
				result.SetMapIndex(mapKey, item)
			}
			dst.Set(result)
			return nil
		}
	}
	return fmt.Errorf("cannot decode %T into %s", value, dst.Type())
}

func decodeBytes(source []byte, dst reflect.Value) error {
	if dst.Kind() == reflect.String {
		dst.SetString(string(source))
		return nil
	}
	if (dst.Kind() != reflect.Slice && dst.Kind() != reflect.Array) || dst.Type().Elem().Kind() != reflect.Uint8 {
		return fmt.Errorf("cannot decode byte string into %s", dst.Type())
	}
	var result reflect.Value
	if dst.Kind() == reflect.Array {
		if dst.Len() != len(source) {
			return fmt.Errorf("byte string length %d does not match %s", len(source), dst.Type())
		}
		result = reflect.New(dst.Type()).Elem()
	} else {
		result = reflect.MakeSlice(dst.Type(), len(source), len(source))
	}
	for i, b := range source {
		result.Index(i).SetUint(uint64(b))
	}
	dst.Set(result)
	return nil
}

func decodeStruct(dict bencode.BDict, dst reflect.Value, depth int) error {
	seen := make(map[string]bool)
	for i := 0; i < dst.NumField(); i++ {
		fieldType := dst.Type().Field(i)
		if fieldType.PkgPath != "" {
			continue
		}
		key := fieldType.Tag.Get("bencode")
		if key == "-" {
			continue
		}
		if strings.Contains(key, ",") {
			return fmt.Errorf("field %s: bencode tag options are unsupported", fieldType.Name)
		}
		if key == "" {
			key = fieldType.Name
		}
		if seen[key] {
			return fmt.Errorf("duplicate bencode field name %q", key)
		}
		seen[key] = true
		value, exists := dict[key]
		if !exists {
			continue
		}
		if err := decodeValue(value, dst.Field(i), depth); err != nil {
			return fmt.Errorf("field %q: %w", key, err)
		}
	}
	return nil
}
