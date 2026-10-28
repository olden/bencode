package bencode

import (
	"fmt"
	"unicode"
)

func parseInt64(b []byte) (int64, error) {
	var n int64
	neg := false
	if len(b) > 0 && b[0] == '-' {
		neg = true
		b = b[1:]
	}
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid digit: %c", c)
		}
		n = n*10 + int64(c-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}

// TokenType represents the type of token in the bencode format
type TokenType int

const (
	TokenIntStart  TokenType = iota // 'i'
	TokenListStart                  // 'l'
	TokenDictStart                  // 'd'
	TokenEnd                        // 'e'
	TokenString                     // "4:spam"
	TokenNumber                     // numbers in int or length of string
	TokenColon                      // ':'
)

// Token represents a single token in the bencode stream
type Token struct {
	Type  TokenType
	Value []byte // Only used for numbers and strings
}

// Lexer reads bytes and produces a stream of tokens
type Lexer struct {
	input []byte
	pos   int
}

// NewLexer returns a new instance of Lexer
func NewLexer(input []byte) *Lexer {
	return &Lexer{input: input}
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() (Token, error) {
	if l.pos >= len(l.input) {
		return Token{}, fmt.Errorf("end of input")
	}

	ch := l.input[l.pos]
	l.pos++

	switch ch {
	case 'i':
		return Token{Type: TokenIntStart}, nil
	case 'l':
		return Token{Type: TokenListStart}, nil
	case 'd':
		return Token{Type: TokenDictStart}, nil
	case 'e':
		return Token{Type: TokenEnd}, nil
	case ':':
		return Token{Type: TokenColon}, nil
	default:
		if unicode.IsDigit(rune(ch)) || ch == '-' {
			start := l.pos - 1
			if ch == '-' {
				if l.pos >= len(l.input) || !unicode.IsDigit(rune(l.input[l.pos])) {
					return Token{}, fmt.Errorf("invalid negative number format")
				}
			}

			for l.pos < len(l.input) && unicode.IsDigit(rune(l.input[l.pos])) {
				l.pos++
			}
			lengthBytes := l.input[start:l.pos]

			if l.pos < len(l.input) && l.input[l.pos] == ':' { // This is a string value
				l.pos++ // skip ':'
				var length int64
				length, err := parseInt64(lengthBytes)
				if err != nil {
					return Token{}, fmt.Errorf("invalid string length: %v", err)
				}
				if l.pos+int(length) > len(l.input) {
					return Token{}, fmt.Errorf("unexpected end of input while reading string")
				}
				str := l.input[l.pos : l.pos+int(length)]
				l.pos += int(length)
				return Token{Type: TokenString, Value: str}, nil
			}

			// Otherwise it's a numeric token (e.g., used in integers)
			return Token{Type: TokenNumber, Value: lengthBytes}, nil
		}
		return Token{}, fmt.Errorf("unexpected character: %q", ch)
	}
}
