package bencode

import (
	"bytes"
	"strings"
	"testing"
)

func FuzzParser_NoPanic(f *testing.F) {
	for _, seed := range []string{
		"", "i0e", "i-9223372036854775808e", "i9223372036854775808e",
		"i03e", "i-0e", "0:", "5:\x00\xff\x80:e", "-1:x",
		"9223372036854775807:x", "18446744073709551616:",
		"le", "de", "li1e3:abce", "d1:ai1ee", "d1:ai1e1:ai2ee",
		"i1ejunk", "d4:infod1:ai1ee1:zd4:infod1:bi2eeee",
		strings.Repeat("l", 129) + "i0e" + strings.Repeat("e", 129),
	} {
		f.Add([]byte(seed))
	}
	// Exercise the upstream-inspired corpus during both seed runs and mutation.
	for _, tt := range referenceParserCases {
		f.Add([]byte(tt.input))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		// Bound the fuzz harness cost; this is not a production document limit.
		if len(input) > 64*1024 {
			t.Skip()
		}
		defer failOnParserPanic(t)
		original := bytes.Clone(input)
		p, err := NewParser(input)
		if err == nil {
			var rawInfo []byte
			p.RawInfo = &rawInfo
			_, _ = p.Parse() // Malformed inputs may return errors, never panic.
		}
		if !bytes.Equal(input, original) {
			t.Fatal("parser modified its input buffer")
		}
	})
}
