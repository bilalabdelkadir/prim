package parser

import (
	"fmt"
	"strings"

	"github.com/bilalabdelkadir/prim/internal/schema"
)

// Parser is a recursive-descent parser for Prisma schema files.
type Parser struct {
	tokens []Token
	pos    int
}

// Parse tokenizes and parses the input string into a Schema AST.
func Parse(input string) (*schema.Schema, error) {
	lex := NewLexer(input)
	tokens := lex.All()
	p := &Parser{tokens: tokens, pos: 0}
	return p.parse()
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TOKEN_EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) next() Token {
	t := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return t
}

func (p *Parser) expect(tt TokenType) (Token, error) {
	t := p.next()
	if t.Type != tt {
		return t, fmt.Errorf("line %d: expected %s, got %s %q",
			t.Line, tt, t.Type, t.Value)
	}
	return t, nil
}

func (p *Parser) parse() (*schema.Schema, error) {
	s := &schema.Schema{
		Models: make([]*Model, 0, 8),
	}

	for p.peek().Type != TOKEN_EOF {
		switch p.peek().Type {
		case TOKEN_DATASOURCE:
			ds, err := p.parseDatasource()
			if err != nil {
				return nil, err
			}
			s.Datasource = ds
		case TOKEN_MODEL:
			m, err := p.parseModel()
			if err != nil {
				return nil, err
			}
			s.Models = append(s.Models, m)
		default:
			// Skip unknown top-level tokens (e.g. enum — not yet implemented).
			p.next()
		}
	}

	return s, nil
}

// parseDatasource parses: datasource <name> { key = value ... }
func (p *Parser) parseDatasource() (*schema.Datasource, error) {
	if _, err := p.expect(TOKEN_DATASOURCE); err != nil {
		return nil, err
	}
	// datasource name (e.g. "db") — consume and discard.
	if t := p.peek(); t.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("line %d: expected datasource name after \"datasource\" keyword, got %s %q",
			t.Line, t.Type, t.Value)
	}
	p.next()
	if t := p.peek(); t.Type != TOKEN_LBRACE {
		return nil, fmt.Errorf("line %d: expected \"{\" to start datasource body, got %s %q",
			t.Line, t.Type, t.Value)
	}
	p.next()

	ds := &schema.Datasource{}

	for p.peek().Type != TOKEN_RBRACE && p.peek().Type != TOKEN_EOF {
		key, err := p.expect(TOKEN_IDENT)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TOKEN_EQUALS); err != nil {
			return nil, err
		}

		var val string
		switch p.peek().Type {
		case TOKEN_STRING:
			tok := p.next()
			val = tok.Value
		case TOKEN_IDENT:
			// Handle env("...") form.
			fnName := p.next()
			if fnName.Value == "env" {
				if _, err := p.expect(TOKEN_LPAREN); err != nil {
					return nil, err
				}
				strTok, err := p.expect(TOKEN_STRING)
				if err != nil {
					return nil, err
				}
				val = strTok.Value
				if _, err := p.expect(TOKEN_RPAREN); err != nil {
					return nil, err
				}
			} else {
				val = fnName.Value
			}
		default:
			tok := p.next()
			val = tok.Value
		}

		switch key.Value {
		case "provider":
			ds.Provider = val
		case "url":
			ds.URL = val
		}
	}

	if _, err := p.expect(TOKEN_RBRACE); err != nil {
		return nil, err
	}
	return ds, nil
}

// parseModel parses: model <Name> { field... }
func (p *Parser) parseModel() (*schema.Model, error) {
	if _, err := p.expect(TOKEN_MODEL); err != nil {
		return nil, err
	}
	if t := p.peek(); t.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("line %d: expected model name after \"model\" keyword, got %s %q",
			t.Line, t.Type, t.Value)
	}
	nameTok := p.next()

	if t := p.peek(); t.Type != TOKEN_LBRACE {
		return nil, fmt.Errorf("line %d: expected \"{\" to start model body, got %s %q",
			t.Line, t.Type, t.Value)
	}
	p.next()

	m := &schema.Model{
		Name:   nameTok.Value,
		Fields: make([]*Field, 0, 8),
	}

	for p.peek().Type != TOKEN_RBRACE && p.peek().Type != TOKEN_EOF {
		f, err := p.parseField()
		if err != nil {
			return nil, err
		}
		m.Fields = append(m.Fields, f)
	}

	if _, err := p.expect(TOKEN_RBRACE); err != nil {
		return nil, err
	}
	return m, nil
}

// parseField parses: <name> <Type>[?|[]] [@attr(...)]...
func (p *Parser) parseField() (*schema.Field, error) {
	nameTok, err := p.expect(TOKEN_IDENT)
	if err != nil {
		return nil, err
	}
	if t := p.peek(); t.Type != TOKEN_IDENT {
		return nil, fmt.Errorf("line %d: expected field type after field name %q, got %s %q",
			t.Line, nameTok.Value, t.Type, t.Value)
	}
	typeTok := p.next()

	f := &schema.Field{
		Name:       nameTok.Value,
		Type:       schema.FieldType(typeTok.Value),
		Attributes: make([]*Attribute, 0, 4),
	}

	// Optional marker.
	if p.peek().Type == TOKEN_QUESTION {
		p.next()
		f.IsOptional = true
	}

	// Array marker.
	if p.peek().Type == TOKEN_BRACKETL {
		p.next()
		if _, err := p.expect(TOKEN_BRACKETR); err != nil {
			return nil, err
		}
		f.IsArray = true
	}

	// Attributes.
	for p.peek().Type == TOKEN_AT {
		attr, err := p.parseAttribute()
		if err != nil {
			return nil, err
		}
		f.Attributes = append(f.Attributes, attr)
	}

	return f, nil
}

// parseAttribute parses: @<name> [ ( args ) ]
func (p *Parser) parseAttribute() (*schema.Attribute, error) {
	if _, err := p.expect(TOKEN_AT); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(TOKEN_IDENT)
	if err != nil {
		return nil, err
	}

	attr := &schema.Attribute{
		Name: nameTok.Value,
	}

	// Optional argument list.
	if p.peek().Type == TOKEN_LPAREN {
		p.next() // consume (
		args, err := p.parseAttributeArgs()
		if err != nil {
			return nil, err
		}
		attr.Args = args
		if _, err := p.expect(TOKEN_RPAREN); err != nil {
			return nil, err
		}
	}

	return attr, nil
}

// parseAttributeArgs collects arguments between parens as raw strings,
// splitting by top-level commas and handling nested parens.
func (p *Parser) parseAttributeArgs() ([]string, error) {
	args := make([]string, 0, 4)
	if p.peek().Type == TOKEN_RPAREN {
		return args, nil
	}

	var buf strings.Builder
	depth := 0

	for {
		t := p.peek()
		if t.Type == TOKEN_EOF {
			return nil, fmt.Errorf("line %d col %d: unexpected EOF in attribute args", t.Line, t.Col)
		}

		if t.Type == TOKEN_RPAREN && depth == 0 {
			// End of top-level args — don't consume the closing paren.
			break
		}

		if t.Type == TOKEN_COMMA && depth == 0 {
			p.next() // consume comma
			args = append(args, strings.TrimSpace(buf.String()))
			buf.Reset()
			continue
		}

		if t.Type == TOKEN_LPAREN {
			depth++
		} else if t.Type == TOKEN_RPAREN {
			depth--
		}

		p.next()
		buf.WriteString(t.Value)
	}

	if buf.Len() > 0 {
		args = append(args, strings.TrimSpace(buf.String()))
	}

	return args, nil
}

// type aliases so tests in this package can reference schema types directly.
type Model = schema.Model
type Field = schema.Field
type Attribute = schema.Attribute
