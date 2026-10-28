package bencode

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"testing"
)

func TestLexer_AllTokenTypes(t *testing.T) {
	inputs := map[string][]Token{
		"i42e":   {{Type: TokenIntStart}, {Type: TokenNumber, Value: []byte("42")}, {Type: TokenEnd}},
		"i0e":    {{Type: TokenIntStart}, {Type: TokenNumber, Value: []byte("0")}, {Type: TokenEnd}},
		"i-10e":  {{Type: TokenIntStart}, {Type: TokenNumber, Value: []byte("-10")}, {Type: TokenEnd}}, // parser will handle signed int
		"4:spam": {{Type: TokenString, Value: []byte("spam")}},
		"0:":     {{Type: TokenString, Value: []byte("")}},
		"li1ei2ei3ee": {
			{Type: TokenListStart},
			{Type: TokenIntStart}, {Type: TokenNumber, Value: []byte("1")}, {Type: TokenEnd},
			{Type: TokenIntStart}, {Type: TokenNumber, Value: []byte("2")}, {Type: TokenEnd},
			{Type: TokenIntStart}, {Type: TokenNumber, Value: []byte("3")}, {Type: TokenEnd},
			{Type: TokenEnd},
		},
		"d3:fooi123e4:spam4:eggsl4:testi1ee": {
			{Type: TokenDictStart},
			{Type: TokenString, Value: []byte("foo")},
			{Type: TokenIntStart},
			{Type: TokenNumber, Value: []byte("123")},
			{Type: TokenEnd},
			{Type: TokenString, Value: []byte("spam")},
			{Type: TokenString, Value: []byte("eggs")},
			{Type: TokenListStart},
			{Type: TokenString, Value: []byte("test")},
			{Type: TokenIntStart},
			{Type: TokenNumber, Value: []byte("1")},
			{Type: TokenEnd},
			{Type: TokenEnd}, // end of outer dict
		},
	}

	for inputStr, expectedTokens := range inputs {
		t.Run("extra case: "+inputStr, func(t *testing.T) {
			lexer := NewLexer([]byte(inputStr))
			for i, exp := range expectedTokens {
				tok, err := lexer.NextToken()
				if err != nil {
					t.Fatalf("unexpected error at token %d: %v", i, err)
				}
				if tok.Type != exp.Type || !bytes.Equal(tok.Value, exp.Value) {
					t.Errorf("token %d mismatch: got %+v, want %+v", i, tok, exp)
				}
			}
		})
	}
}

func TestLexer_StringLengthBounds(t *testing.T) {
	maxInt := uint64(^uint(0) >> 1)
	aboveInt32Error := error(io.ErrUnexpectedEOF)
	if strconv.IntSize == 32 {
		aboveInt32Error = strconv.ErrRange
	}
	for _, tt := range []struct {
		name   string
		length string
		want   error
	}{
		{"native_max", strconv.FormatUint(maxInt, 10), io.ErrUnexpectedEOF},
		{"native_overflow", strconv.FormatUint(maxInt+1, 10), strconv.ErrRange},
		{"int32_max", "2147483647", io.ErrUnexpectedEOF},
		{"int32_overflow", "2147483648", aboveInt32Error},
		{"int32_wraparound", "4294967296", aboveInt32Error},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// A nonzero position also exercises the end-index addition check.
			lexer := NewLexer([]byte("1:y" + tt.length + ":x"))
			if _, err := lexer.NextToken(); err != nil {
				t.Fatal(err)
			}
			token, err := lexer.NextToken()
			if !errors.Is(err, tt.want) {
				t.Fatalf("got token %#v and error %v, want %v", token, err, tt.want)
			}
		})
	}
}

func BenchmarkLexer(b *testing.B) {
	var testData = []byte("d4:spam4:eggsi123e3:fooi-42ee")
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(testData)
		for {
			_, err := lexer.NextToken()
			if err != nil {
				break
			}
		}
	}
}
