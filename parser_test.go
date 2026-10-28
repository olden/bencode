package bencode

import (
	"reflect"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		input    string
		expected BValue
	}{
		// primitive types
		{"i42e", BInt(42)},
		{"i-10e", BInt(-10)},
		{"4:spam", BString([]byte("spam"))},
		{"0:", BString([]byte(""))},
		// list
		{
			"l4:spami42ee",
			BList{BString([]byte("spam")), BInt(42)},
		},
		// dict
		{
			"d3:fooi7e4:spam4:eggse",
			BDict{
				"foo":  BInt(7),
				"spam": BString([]byte("eggs")),
			},
		},
		// nested list in dict
		{
			"d4:spaml5:hello5:worlde3:fooi99ee",
			BDict{
				"spam": BList{BString([]byte("hello")), BString([]byte("world"))},
				"foo":  BInt(99),
			},
		},
	}

	for _, tt := range tests {
		p, err := NewParser([]byte(tt.input))
		if err != nil {
			t.Fatalf("parser init failed: %v", err)
		}
		val, err := p.Parse()
		if err != nil {
			t.Errorf("parse error for input %q: %v", tt.input, err)
			continue
		}
		if !reflect.DeepEqual(val, tt.expected) {
			t.Errorf("unexpected result for %q: got %v, want %v", tt.input, val, tt.expected)
		}
	}
}

func TestParser_Parse_Errors(t *testing.T) {
	cases := []string{
		"i42",             // missing 'e'
		"i4x2e",           // invalid number
		"d3:fooi7e4:spam", // missing final 'e'
		"l4:spam4:egg",    // missing final 'e'
		"4spam",           // missing ':'
	}

	for _, input := range cases {
		p, err := NewParser([]byte(input))
		if err != nil {
			continue // lexer error is fine
		}
		_, err = p.Parse()
		if err == nil {
			t.Errorf("expected error for input %q, got nil", input)
		}
	}
}

func BenchmarkParser_Parse(b *testing.B) {
	input := []byte("d4:spaml5:hello5:worlde3:fooi99ee")
	for i := 0; i < b.N; i++ {
		p, err := NewParser(input)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := p.Parse(); err != nil {
			b.Fatal(err)
		}
	}
}
