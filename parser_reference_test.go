package bencode

import (
	"bytes"
	"testing"
)

// Independently written fixtures inspired by these upstream test scenarios:
//
// zeebo/bencode, TestDecode, TestRawDecode and TestNestedRawDecode:
// https://github.com/zeebo/bencode/blob/14e20e3c5944f90565c41c2333c409d0f547ba4f/decode_test.go
// Inspected mirror of that revision:
// https://chromium.googlesource.com/external/github.com/zeebo/bencode/+/14e20e3c5944f90565c41c2333c409d0f547ba4f/decode_test.go
//
// jackpal/bencode-go, decodeTests and unmarshalTests:
// https://github.com/jackpal/bencode-go/blob/master/bencode_test.go
//
// Fixtures and assertions use our BValue API. Unlike jackpal's permissive
// unmarshal cases, fractional and scientific notation must be rejected here.
// A nil expected value denotes malformed input (bencode has no null value).
var referenceParserCases = []struct {
	name  string
	input string
	want  BValue
}{
	{
		"zeebo/list_of_dictionaries",
		"ld2:idi4eed2:idi9eee",
		BList{BDict{"id": BInt(4)}, BDict{"id": BInt(9)}},
	},
	{
		"zeebo/dictionary_of_lists",
		"d3:catli4ei6ee3:dogli8ei10eee",
		BDict{"cat": BList{BInt(4), BInt(6)}, "dog": BList{BInt(8), BInt(10)}},
	},
	{
		"zeebo/shared_key_prefix",
		"d1:a3:one2:aa3:twoe",
		BDict{"a": BString("one"), "aa": BString("two")},
	},
	{
		"zeebo/nested_empty_lists",
		"lllleeee",
		BList{BList{BList{BList{}}}},
	},
	{
		"zeebo/mixed_nested_containers",
		"li7e4:windd1:k1:veli9eee",
		BList{BInt(7), BString("wind"), BDict{"k": BString("v")}, BList{BInt(9)}},
	},
	{"zeebo/short_string_payload", "7:short", nil},
	{"zeebo/string_token_inside_integer", "i21:ab", nil},
	{"zeebo/negative_dictionary_key_length", "d-2:ab", nil},
	{"zeebo/unfinished_list_after_complete_children", "l4:moon3:sun", nil},
	{"jackpal/empty_list", "le", BList{}},
	{"jackpal/empty_dictionary", "de", BDict{}},
	{"jackpal/quoted_string", "4:a\"bc", BString("a\"bc")},
	{"jackpal/multi_digit_length", "12:planet\"earth", BString("planet\"earth")},
	{"jackpal/mixed_list", "li17e4:notee", BList{BInt(17), BString("note")}},
	{"jackpal/reject_fraction", "i2.75e", nil},
	{"jackpal/reject_negative_fraction", "i-2.75e", nil},
	{"jackpal/reject_positive_exponent", "i2.75E+3e", nil},
	{"jackpal/reject_negative_exponent", "i-2.75E-3e", nil},
}

func TestParser_ReferenceCases(t *testing.T) {
	for _, tt := range referenceParserCases {
		t.Run(tt.name, func(t *testing.T) {
			defer failOnParserPanic(t)
			p, err := NewParser([]byte(tt.input))
			var got BValue
			if err == nil {
				got, err = p.Parse()
			}
			if tt.want == nil {
				if err == nil {
					t.Fatalf("expected error for %q, got %#v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("valid document %q rejected: %v", tt.input, err)
			}
			if !equalBValue(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

// Compare decoded content without imposing a nil-versus-empty allocation
// convention on lists, dictionaries or byte strings.
func equalBValue(got, want BValue) bool {
	switch expected := want.(type) {
	case BInt:
		actual, ok := got.(BInt)
		return ok && actual == expected
	case BString:
		actual, ok := got.(BString)
		return ok && bytes.Equal(actual, expected)
	case BList:
		actual, ok := got.(BList)
		if !ok || len(actual) != len(expected) {
			return false
		}
		for i, item := range expected {
			if !equalBValue(actual[i], item) {
				return false
			}
		}
		return true
	case BDict:
		actual, ok := got.(BDict)
		if !ok || len(actual) != len(expected) {
			return false
		}
		for key, item := range expected {
			value, exists := actual[key]
			if !exists || !equalBValue(value, item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
