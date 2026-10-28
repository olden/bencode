package bencode

import (
	"fmt"
	"io"
	"strconv"
)

func parseInt64(b []byte) (int64, error) {
	digits := b
	negative := len(b) > 0 && b[0] == '-'
	if negative {
		digits = b[1:]
	}
	if len(digits) == 0 {
		return 0, fmt.Errorf("missing digits")
	}
	if digits[0] == '0' && (len(digits) > 1 || negative) {
		return 0, fmt.Errorf("leading zero or negative zero")
	}
	for _, c := range digits {
		if !isDigit(c) {
			return 0, fmt.Errorf("invalid digit: %c", c)
		}
	}
	return strconv.ParseInt(string(b), 10, 64)
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
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
		return Token{}, io.EOF
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
		if isDigit(ch) || ch == '-' {
			start := l.pos - 1
			if ch == '-' {
				if l.pos >= len(l.input) || !isDigit(l.input[l.pos]) {
					return Token{}, fmt.Errorf("byte %d: invalid negative number format", start)
				}
			}

			for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
				l.pos++
			}
			lengthBytes := l.input[start:l.pos]

			if l.pos < len(l.input) && l.input[l.pos] == ':' { // This is a string value
				l.pos++ // skip ':'
				if ch == '-' {
					return Token{}, fmt.Errorf("byte %d: string length cannot be negative", start)
				}
				if len(lengthBytes) > 1 && lengthBytes[0] == '0' {
					return Token{}, fmt.Errorf("byte %d: invalid string length: leading zero", start)
				}
				// Lengths are slice indexes, so parse directly into the native
				// int range instead of narrowing an int64 on 32-bit systems.
				length, err := strconv.Atoi(string(lengthBytes))
				if err != nil {
					return Token{}, fmt.Errorf("byte %d: invalid string length: %w", start, err)
				}
				// Check the remaining bytes before adding the length to pos.
				if length > len(l.input)-l.pos {
					return Token{}, fmt.Errorf("byte %d: truncated string: %w", start, io.ErrUnexpectedEOF)
				}
				str := l.input[l.pos : l.pos+length]
				l.pos += length
				return Token{Type: TokenString, Value: str}, nil
			}

			// Otherwise it's a numeric token (e.g., used in integers)
			return Token{Type: TokenNumber, Value: lengthBytes}, nil
		}
		return Token{}, fmt.Errorf("byte %d: unexpected character: %q", l.pos-1, ch)
	}
}
