package bencode

import (
	"errors"
	"fmt"
	"io"
)

type BValue interface{}
type BInt int64
type BString []byte
type BList []BValue
type BDict map[string]BValue

// MaxDepth limits simultaneously open lists and dictionaries, counting the
// root container as depth 1. Integers are limited to int64 by BInt.
const MaxDepth = 128

type Parser struct {
	lexer     *Lexer
	curr      Token
	currStart int
	input     []byte
	parsed    bool
	rawInfo   []byte
	// RawInfo optionally receives the encoded root dictionary's info value
	// after a successful Parse. The returned bytes alias the input buffer.
	RawInfo *[]byte
}

func NewParser(input []byte) (*Parser, error) {
	lexer := NewLexer(input)
	tok, err := lexer.NextToken()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("byte 0: expected value: %w", io.ErrUnexpectedEOF)
		}
		return nil, err
	}
	return &Parser{lexer: lexer, curr: tok, input: input}, nil
}

func (p *Parser) advance() error {
	start := p.lexer.pos
	tok, err := p.lexer.NextToken()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("byte %d: %w", start, io.ErrUnexpectedEOF)
		}
		return err
	}
	p.curr = tok
	p.currStart = start
	return nil
}

// Parse reads exactly one complete document. A Parser is single-use, and its
// input must not be modified while parsing or while using returned byte slices.
func (p *Parser) Parse() (BValue, error) {
	if p.parsed {
		return nil, fmt.Errorf("parser has already been used")
	}
	p.parsed = true
	if p.RawInfo != nil {
		*p.RawInfo = nil
	}
	value, err := p.parseValue(0)
	if err != nil {
		return nil, err
	}
	if p.lexer.pos != len(p.input) {
		return nil, fmt.Errorf("byte %d: trailing data after root value", p.lexer.pos)
	}
	if p.RawInfo != nil {
		*p.RawInfo = p.rawInfo
	}
	return value, nil
}

// parseValue leaves curr on the value's final token; its caller advances to
// the next sibling. depth is the number of containers already open.
func (p *Parser) parseValue(depth int) (BValue, error) {
	if (p.curr.Type == TokenListStart || p.curr.Type == TokenDictStart) && depth >= MaxDepth {
		return nil, fmt.Errorf("byte %d: nesting exceeds limit %d", p.currStart, MaxDepth)
	}
	switch p.curr.Type {
	case TokenIntStart:
		return p.parseInt()
	case TokenString:
		val := BString(p.curr.Value)
		return val, nil
	case TokenListStart:
		return p.parseList(depth + 1)
	case TokenDictStart:
		return p.parseDict(depth + 1)
	default:
		return nil, fmt.Errorf("byte %d: unexpected token: %v", p.currStart, p.curr.Type)
	}
}

func (p *Parser) parseInt() (BValue, error) {
	if err := p.advance(); err != nil { // skip 'i'
		return nil, err
	}
	if p.curr.Type != TokenNumber {
		return nil, fmt.Errorf("byte %d: expected number after 'i'", p.currStart)
	}
	val, err := parseInt64(p.curr.Value)
	if err != nil {
		return nil, fmt.Errorf("byte %d: invalid integer: %w", p.currStart, err)
	}
	if err := p.advance(); err != nil { // skip number
		return nil, err
	}
	if p.curr.Type != TokenEnd {
		return nil, fmt.Errorf("byte %d: expected 'e' after number", p.currStart)
	}
	return BInt(val), nil
}

func (p *Parser) parseList(depth int) (BValue, error) {
	var list BList
	if err := p.advance(); err != nil { // skip 'l'
		return nil, err
	}
	for p.curr.Type != TokenEnd {
		val, err := p.parseValue(depth)
		if err != nil {
			return nil, err
		}
		list = append(list, val)
		if err := p.advance(); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (p *Parser) parseDict(depth int) (BValue, error) {
	dict := make(BDict)
	var previousKey string
	if err := p.advance(); err != nil { // skip 'd'
		return nil, err
	}
	for p.curr.Type != TokenEnd {
		if p.curr.Type != TokenString {
			return nil, fmt.Errorf("byte %d: expected string key in dict", p.currStart)
		}
		key := string(p.curr.Value)
		if len(dict) > 0 && key <= previousKey {
			return nil, fmt.Errorf("byte %d: dictionary keys must be strictly increasing (duplicate or out of order)", p.currStart)
		}
		previousKey = key
		// The lexer is positioned immediately after the key, before reading
		// any bytes of the value (including its length prefix for strings).
		start := p.lexer.pos
		if err := p.advance(); err != nil {
			return nil, err
		}

		val, err := p.parseValue(depth)
		if err != nil {
			return nil, err
		}
		dict[key] = val
		if depth == 1 && key == "info" && p.RawInfo != nil {
			p.rawInfo = p.input[start:p.lexer.pos]
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
	}
	return dict, nil
}
