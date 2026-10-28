package bencode

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
)

// A panic must fail even when the test expects an error. Recovering here lets
// the remaining regression cases run and report their own failures.
func failOnParserPanic(t *testing.T) {
	t.Helper()
	if failure := recover(); failure != nil {
		t.Fatalf("parser panicked instead of returning a value or error: %v", failure)
	}
}

func TestParser_IntegerBoundaries(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  BInt
	}{
		{"i0e", 0},
		{"i1e", 1},
		{"i-1e", -1},
		{"i9223372036854775807e", 9223372036854775807},
		{"i-9223372036854775808e", -9223372036854775808},
	} {
		t.Run(tt.input, func(t *testing.T) {
			defer failOnParserPanic(t)
			p, err := NewParser([]byte(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			got, err := p.Parse()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %v (%T), want %v", got, got, tt.want)
			}
		})
	}
}

func TestParser_ByteStrings(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input string
		want  []byte
	}{
		{"empty", "0:", []byte{}},
		{"ascii", "4:spam", []byte("spam")},
		{"binary", "5:\x00\xff\x80:e", []byte{0, 255, 128, ':', 'e'}},
		{"utf8_byte_length", "2:é", []byte("é")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer failOnParserPanic(t)
			p, err := NewParser([]byte(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			value, err := p.Parse()
			if err != nil {
				t.Fatal(err)
			}
			got, ok := value.(BString)
			if !ok || !bytes.Equal(got, tt.want) {
				t.Fatalf("got %#v, want bytes %x", value, tt.want)
			}
		})
	}
}

func TestParser_DictionaryByteOrdering(t *testing.T) {
	p, err := NewParser([]byte("d1:\x00i0e1:Ai1e1:ai2e1:\xffi3ee"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	want := BDict{"\x00": BInt(0), "A": BInt(1), "a": BInt(2), "\xff": BInt(3)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestParser_RejectsMalformedDocuments(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"negative_string_length", "-1:x"},
		{"negative_zero_string_length", "-0:"},
		{"string_end_overflow", "9223372036854775807:x"},
		{"string_length_overflow", "9223372036854775808:x"},
		{"string_length_wraparound", "18446744073709551616:"},
		{"string_length_leading_zero", "01:x"},
		{"string_length_multiple_zeros", "00:"},
		// BInt is int64: out-of-range numbers must return errors, not wrap.
		{"integer_above_max", "i9223372036854775808e"},
		{"integer_below_min", "i-9223372036854775809e"},
		{"integer_wraparound", "i18446744073709551616e"},
		{"integer_negative_zero", "i-0e"},
		{"integer_leading_zero", "i03e"},
		{"integer_multiple_zeros", "i00e"},
		{"integer_negative_leading_zero", "i-03e"},
		{"integer_plus", "i+1e"},
		{"integer_missing_digits", "ie"},
		{"integer_sign_only", "i-e"},
		{"integer_fraction", "i1.5e"},
		{"string_without_colon", "4spam"},
		{"integer_trailing_garbage", "i1ejunk"},
		{"string_trailing_garbage", "4:spamjunk"},
		{"list_trailing_garbage", "lejunk"},
		{"dictionary_trailing_garbage", "dejunk"},
		{"trailing_whitespace", "i1e\n"},
		{"second_root_value", "i1ei2e"},
		{"unsorted_dictionary", "d1:bi1e1:ai2ee"},
		{"duplicate_key", "d1:ai1e1:ai2ee"},
		{"duplicate_empty_key", "d0:i1e0:i2ee"},
		{"empty_key_out_of_order", "d1:ai1e0:i2ee"},
		{"nested_duplicate_key", "d1:ad1:xi1e1:xi2eee"},
		{"non_string_key", "di1ei2ee"},
		{"key_without_value", "d1:ae"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer failOnParserPanic(t)
			p, err := NewParser([]byte(tt.input))
			if err != nil {
				return
			}
			value, err := p.Parse()
			if err == nil {
				t.Fatalf("accepted malformed document %q as %#v", tt.input, value)
			}
		})
	}
}

func TestParser_RejectsEveryTruncatedPrefix(t *testing.T) {
	for _, input := range []string{
		"i-123e", "5:\x00\xffabc", "li1e3:abce", "d1:ali1e3:abcee",
	} {
		t.Run(fmt.Sprintf("%q", input), func(t *testing.T) {
			p, err := NewParser([]byte(input))
			if err != nil {
				t.Fatalf("invalid complete fixture: %v", err)
			}
			if _, err := p.Parse(); err != nil {
				t.Fatalf("invalid complete fixture: %v", err)
			}
			for end := 0; end < len(input); end++ {
				t.Run(fmt.Sprintf("prefix_%d", end), func(t *testing.T) {
					defer failOnParserPanic(t)
					p, err := NewParser([]byte(input[:end]))
					if err != nil {
						return
					}
					if _, err := p.Parse(); err == nil {
						t.Fatalf("accepted truncated document %q", input[:end])
					}
				})
			}
		})
	}
}

func TestParser_RawInfoRootScope(t *testing.T) {
	const info = "d1:ai1ee"
	const otherInfo = "d1:bi2ee"
	for _, tt := range []struct {
		name  string
		input string
		want  string
	}{
		{"root_only", "d4:info" + info + "e", info},
		{"siblings", "d8:announce1:x4:info" + info + "1:z1:ye", info},
		{"nested_before_root", "d1:ad4:info" + otherInfo + "e4:info" + info + "e", info},
		{"nested_after_root", "d4:info" + info + "1:zd4:info" + otherInfo + "ee", info},
		{"only_nested", "d1:zd4:info" + otherInfo + "ee", ""},
		{"inside_root_list", "ld4:info" + otherInfo + "ee", ""},
		{"inside_info", "d4:infod4:info" + otherInfo + "ee", "d4:info" + otherInfo + "e"},
		{"binary_info", "d4:infod1:x4:\x00\xff:eee", "d1:x4:\x00\xff:ee"},
		// The generic parser captures the encoded value; the torrent loader
		// is responsible for requiring info to be a dictionary.
		{"string_info_range", "d4:info4:spame", "4:spam"},
		{"integer_info_range", "d4:infoi42ee", "i42e"},
		// Adapted from zeebo's nested RawMessage scenarios; source in
		// parser_reference_test.go. Capture the entire dictionary including
		// empty containers and every closing delimiter.
		{"mixed_containers_in_info", "d4:infod1:aldedei17e2:oke1:zi9eee", "d1:aldedei17e2:oke1:zi9ee"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer failOnParserPanic(t)
			p, err := NewParser([]byte(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			var rawInfo []byte
			p.RawInfo = &rawInfo
			if _, err := p.Parse(); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(rawInfo, []byte(tt.want)) {
				t.Fatalf("captured %q, want exact root info bytes %q", rawInfo, tt.want)
			}
		})
	}
}

func TestParser_NestingLimit(t *testing.T) {
	// Application contract, not a BEP 3 requirement: at most 128
	// simultaneously open containers, counting the root as depth 1.
	const maxDepth = 128
	for _, container := range []struct {
		name string
		open string
	}{
		{"lists", "l"},
		{"dictionaries", "d1:a"},
	} {
		for _, depth := range []int{1, maxDepth, maxDepth + 1} {
			t.Run(fmt.Sprintf("%s/depth_%d", container.name, depth), func(t *testing.T) {
				defer failOnParserPanic(t)
				input := strings.Repeat(container.open, depth) + "i0e" + strings.Repeat("e", depth)
				p, err := NewParser([]byte(input))
				if err == nil {
					_, err = p.Parse()
				}
				if depth <= maxDepth && err != nil {
					t.Fatalf("allowed depth %d rejected: %v", depth, err)
				}
				if depth > maxDepth && err == nil {
					t.Fatalf("accepted depth %d exceeding limit %d", depth, maxDepth)
				}
			})
		}
	}
}

func TestParser_MixedNestingLimit(t *testing.T) {
	for _, depth := range []int{128, 129} {
		t.Run(fmt.Sprintf("depth_%d", depth), func(t *testing.T) {
			var input strings.Builder
			for level := 0; level < depth; level++ {
				if level%2 == 0 {
					input.WriteString("d1:a")
				} else {
					input.WriteByte('l')
				}
			}
			input.WriteString("i0e")
			input.WriteString(strings.Repeat("e", depth))
			p, err := NewParser([]byte(input.String()))
			if err == nil {
				_, err = p.Parse()
			}
			if (err != nil) != (depth > 128) {
				t.Fatalf("depth %d: error = %v, want rejection = %t", depth, err, depth > 128)
			}
		})
	}
}

func TestParser_EmptyDictionaryKey(t *testing.T) {
	p, err := NewParser([]byte("d0:i1e1:ai2ee"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	want := BDict{"": BInt(1), "a": BInt(2)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestParser_RawInfoClearedOnFailureOrAbsence(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"absent", "de", false},
		{"trailing_data", "d4:infodeejunk", true},
		{"incomplete_root", "d4:infode", true},
		{"invalid_later_field", "d4:infode1:zi03ee", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParser([]byte(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			rawInfo := []byte("stale value")
			p.RawInfo = &rawInfo
			_, err = p.Parse()
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, want error = %t", err, tt.wantErr)
			}
			if len(rawInfo) != 0 {
				t.Fatalf("published stale or partially parsed info: %q", rawInfo)
			}
		})
	}
}

func TestParser_UnexpectedEOF(t *testing.T) {
	for _, input := range []string{"", "i", "i12", "4:ab", "li1e", "d1:a"} {
		t.Run(fmt.Sprintf("%q", input), func(t *testing.T) {
			p, err := NewParser([]byte(input))
			if err == nil {
				_, err = p.Parse()
			}
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("got %v, want io.ErrUnexpectedEOF", err)
			}
			if !strings.Contains(err.Error(), "byte ") {
				t.Fatalf("error has no input offset: %v", err)
			}
		})
	}
}

func TestParser_SingleUse(t *testing.T) {
	p, err := NewParser([]byte("4:spam"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Parse(); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Parse(); err == nil {
		t.Fatal("repeated Parse returned an already consumed document")
	}
}
