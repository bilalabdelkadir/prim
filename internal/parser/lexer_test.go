package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLexer_BasicTokens(t *testing.T) {
	input := `model User { id Int @id }`
	lex := NewLexer(input)
	tokens := lex.All()

	expected := []struct {
		typ TokenType
		val string
	}{
		{TOKEN_MODEL, "model"},
		{TOKEN_IDENT, "User"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_IDENT, "id"},
		{TOKEN_IDENT, "Int"},
		{TOKEN_AT, "@"},
		{TOKEN_IDENT, "id"},
		{TOKEN_RBRACE, "}"},
		{TOKEN_EOF, ""},
	}

	assert.Equal(t, len(expected), len(tokens), "token count mismatch")
	for i, exp := range expected {
		assert.Equal(t, exp.typ, tokens[i].Type, "token %d type", i)
		assert.Equal(t, exp.val, tokens[i].Value, "token %d value", i)
	}
}

func TestLexer_OptionalField(t *testing.T) {
	input := `name String?`
	lex := NewLexer(input)
	tokens := lex.All()

	assert.Equal(t, TOKEN_IDENT, tokens[0].Type)
	assert.Equal(t, "name", tokens[0].Value)
	assert.Equal(t, TOKEN_IDENT, tokens[1].Type)
	assert.Equal(t, "String", tokens[1].Value)
	assert.Equal(t, TOKEN_QUESTION, tokens[2].Type)
	assert.Equal(t, TOKEN_EOF, tokens[3].Type)
}

func TestLexer_StringLiteral(t *testing.T) {
	input := `"postgresql"`
	lex := NewLexer(input)
	tok := lex.Next()

	assert.Equal(t, TOKEN_STRING, tok.Type)
	assert.Equal(t, "postgresql", tok.Value)
}

func TestLexer_Comments(t *testing.T) {
	input := "// this is a comment\nmodel User {}"
	lex := NewLexer(input)
	tokens := lex.All()

	assert.Equal(t, TOKEN_MODEL, tokens[0].Type)
	assert.Equal(t, TOKEN_IDENT, tokens[1].Type)
	assert.Equal(t, "User", tokens[1].Value)
	assert.Equal(t, TOKEN_LBRACE, tokens[2].Type)
	assert.Equal(t, TOKEN_RBRACE, tokens[3].Type)
	assert.Equal(t, TOKEN_EOF, tokens[4].Type)
}

func TestLexer_Parens(t *testing.T) {
	input := `@default(autoincrement())`
	lex := NewLexer(input)
	tokens := lex.All()

	expected := []TokenType{
		TOKEN_AT,
		TOKEN_IDENT,  // default
		TOKEN_LPAREN, // (
		TOKEN_IDENT,  // autoincrement
		TOKEN_LPAREN, // (
		TOKEN_RPAREN, // )
		TOKEN_RPAREN, // )
		TOKEN_EOF,
	}

	assert.Equal(t, len(expected), len(tokens), "token count")
	for i, exp := range expected {
		assert.Equal(t, exp, tokens[i].Type, "token %d type", i)
	}
	assert.Equal(t, "default", tokens[1].Value)
	assert.Equal(t, "autoincrement", tokens[3].Value)
}
