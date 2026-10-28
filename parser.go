package bencode

import (
	"fmt"
)

type BValue interface{}
type BInt int64
type BString []byte
type BList []BValue
type BDict map[string]BValue

type Parser struct {
	lexer   *Lexer
	curr    Token
	input   []byte
	RawInfo *[]byte
}

func NewParser(input []byte) (*Parser, error) {
	lexer := NewLexer(input)
	tok, err := lexer.NextToken()
	if err != nil {
		return nil, err
	}
	return &Parser{lexer: lexer, curr: tok, input: input}, nil
}

func (p *Parser) advance() error {
	tok, err := p.lexer.NextToken()
	if err != nil {
		return err
	}
	p.curr = tok
	return nil
}

func (p *Parser) Parse() (BValue, error) {
	switch p.curr.Type {
	case TokenIntStart:
		return p.parseInt()
	case TokenString:
		val := BString(p.curr.Value)
		return val, nil
	case TokenListStart:
		return p.parseList()
	case TokenDictStart:
		return p.parseDict()
	default:
		return nil, fmt.Errorf("unexpected token: %v", p.curr.Type)
	}
}

func (p *Parser) parseInt() (BValue, error) {
	if err := p.advance(); err != nil { // skip 'i'
		return nil, err
	}
	if p.curr.Type != TokenNumber {
		return nil, fmt.Errorf("expected number after 'i'")
	}
	val, err := parseInt64(p.curr.Value)
	if err != nil {
		return nil, err
	}
	if err := p.advance(); err != nil { // skip number
		return nil, err
	}
	if p.curr.Type != TokenEnd {
		return nil, fmt.Errorf("expected 'e' after number")
	}
	return BInt(val), nil
}

func (p *Parser) parseList() (BValue, error) {
	var list BList
	if err := p.advance(); err != nil { // skip 'l'
		return nil, err
	}
	for p.curr.Type != TokenEnd {
		val, err := p.Parse()
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

func (p *Parser) parseDict() (BValue, error) {
	dict := make(BDict)
	if err := p.advance(); err != nil { // skip 'd'
		return nil, err
	}
	for p.curr.Type != TokenEnd {
		if p.curr.Type != TokenString {
			return nil, fmt.Errorf("expected string key in dict")
		}
		key := string(p.curr.Value)
		if err := p.advance(); err != nil {
			return nil, err
		}

		if key == "info" && p.RawInfo != nil {
			start := p.lexer.pos - 1
			val, err := p.Parse()
			if err != nil {
				return nil, err
			}
			end := p.lexer.pos
			*p.RawInfo = p.input[start:end]
			dict[key] = val
		} else {
			val, err := p.Parse()
			if err != nil {
				return nil, err
			}
			dict[key] = val
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
	}
	return dict, nil
}
