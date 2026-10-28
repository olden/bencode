package bencode_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/olden/bencode"
	"github.com/olden/bencode/decoder"
	"github.com/olden/bencode/encoder"
)

func TestParseEncodeRoundTrip(t *testing.T) {
	for _, input := range []string{
		"i0e", "i-9223372036854775808e", "i9223372036854775807e",
		"0:", "4:\x00\xff:e", "le", "de", "li7e3:cate",
		"d0:i1e1:ali2e3:doge1:bd1:x0:ee",
	} {
		t.Run(input, func(t *testing.T) {
			p, err := bencode.NewParser([]byte(input))
			if err != nil {
				t.Fatal(err)
			}
			first, err := p.Parse()
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := encoder.Encode(first)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encoded, []byte(input)) {
				t.Fatalf("encoded %q, want %q", encoded, input)
			}
			p, err = bencode.NewParser(encoded)
			if err != nil {
				t.Fatal(err)
			}
			second, err := p.Parse()
			if err != nil || !reflect.DeepEqual(first, second) {
				t.Fatalf("round trip changed value: %#v -> %#v (error %v)", first, second, err)
			}
		})
	}
}

func FuzzUnmarshalRoundTrip(f *testing.F) {
	for _, input := range []string{"i-7e", "3:cat", "le", "de", "d1:ali2e2:oke1:z3:\x00\xffxe"} {
		f.Add([]byte(input))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 64*1024 {
			t.Skip()
		}
		p, err := bencode.NewParser(input)
		if err != nil {
			return
		}
		value, err := p.Parse()
		if err != nil {
			return
		}
		// Confirm that the extra struct wrapper also fits the parser's depth
		// budget before requiring the decoder to accept it.
		wrapped := append([]byte("d5:value"), input...)
		wrapped = append(wrapped, 'e')
		p, err = bencode.NewParser(wrapped)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Parse(); err != nil {
			return
		}
		var out struct {
			Value any `bencode:"value"`
		}
		if err := decoder.UnmarshalBDict(bencode.BDict{"value": value}, &out); err != nil {
			t.Fatal(err)
		}
		output, err := encoder.Encode(out.Value)
		if err != nil || !bytes.Equal(input, output) {
			t.Fatalf("unmarshal changed canonical bytes: %q -> %q (%v)", input, output, err)
		}
	})
}

func FuzzParseEncodeRoundTrip(f *testing.F) {
	for _, input := range []string{"i7e", "0:", "le", "de", "d1:ali1e2:oke1:z3:\x00\xffxe"} {
		f.Add([]byte(input))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 64*1024 {
			t.Skip()
		}
		p, err := bencode.NewParser(input)
		if err != nil {
			return
		}
		value, err := p.Parse()
		if err != nil {
			return
		}
		output, err := encoder.Encode(value)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(input, output) {
			t.Fatalf("canonical bytes changed: %q -> %q", input, output)
		}
	})
}
