package bencode

import (
	"bytes"
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
