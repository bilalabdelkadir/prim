package parser

// TokenType identifies the kind of lexical token.
type TokenType int

const (
	TOKEN_EOF TokenType = iota
	TOKEN_MODEL
	TOKEN_DATASOURCE
	TOKEN_ENUM
	TOKEN_IDENT
	TOKEN_AT
	TOKEN_LBRACE
	TOKEN_RBRACE
	TOKEN_QUESTION
	TOKEN_BRACKETL
	TOKEN_BRACKETR
	TOKEN_EQUALS
	TOKEN_STRING
	TOKEN_LPAREN
	TOKEN_RPAREN
	TOKEN_COMMA
)

// Token is a single lexical unit produced by the Lexer.
type Token struct {
	Type  TokenType
	Value string
	Line  int
	Col   int
}

// Lexer performs a single-pass tokenization of a Prisma schema string.
type Lexer struct {
	input []byte
	pos   int
	line  int
	col   int
}

// NewLexer creates a Lexer for the given input string.
func NewLexer(input string) *Lexer {
	return &Lexer{
		input: []byte(input),
		pos:   0,
		line:  1,
		col:   1,
	}
}

// Next returns the next token from the input.
func (l *Lexer) Next() Token {
	l.skipWhitespaceAndComments()

	if l.pos >= len(l.input) {
		return Token{Type: TOKEN_EOF, Line: l.line, Col: l.col}
	}

	ch := l.input[l.pos]
	line, col := l.line, l.col

	switch ch {
	case '{':
		l.advance()
		return Token{Type: TOKEN_LBRACE, Value: "{", Line: line, Col: col}
	case '}':
		l.advance()
		return Token{Type: TOKEN_RBRACE, Value: "}", Line: line, Col: col}
	case '@':
		l.advance()
		return Token{Type: TOKEN_AT, Value: "@", Line: line, Col: col}
	case '?':
		l.advance()
		return Token{Type: TOKEN_QUESTION, Value: "?", Line: line, Col: col}
	case '[':
		l.advance()
		return Token{Type: TOKEN_BRACKETL, Value: "[", Line: line, Col: col}
	case ']':
		l.advance()
		return Token{Type: TOKEN_BRACKETR, Value: "]", Line: line, Col: col}
	case '=':
		l.advance()
		return Token{Type: TOKEN_EQUALS, Value: "=", Line: line, Col: col}
	case '(':
		l.advance()
		return Token{Type: TOKEN_LPAREN, Value: "(", Line: line, Col: col}
	case ')':
		l.advance()
		return Token{Type: TOKEN_RPAREN, Value: ")", Line: line, Col: col}
	case ',':
		l.advance()
		return Token{Type: TOKEN_COMMA, Value: ",", Line: line, Col: col}
	case '"':
		return l.readString(line, col)
	default:
		if isIdentStart(ch) {
			return l.readIdent(line, col)
		}
		// Unknown character — emit as IDENT with single char, advance past it.
		l.advance()
		return Token{Type: TOKEN_IDENT, Value: string(ch), Line: line, Col: col}
	}
}

// All returns every token up to and including EOF.
func (l *Lexer) All() []Token {
	tokens := make([]Token, 0, 64)
	for {
		t := l.Next()
		tokens = append(tokens, t)
		if t.Type == TOKEN_EOF {
			break
		}
	}
	return tokens
}

// advance moves the position forward by one byte, updating line/col.
func (l *Lexer) advance() {
	if l.pos < len(l.input) {
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

// skipWhitespaceAndComments skips spaces, tabs, newlines, and // line comments.
func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.advance()
			continue
		}
		// Line comment: // until end of line.
		if ch == '/' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '/' {
			for l.pos < len(l.input) && l.input[l.pos] != '\n' {
				l.advance()
			}
			continue
		}
		break
	}
}

// readString reads a double-quoted string literal (the opening quote is at l.pos).
func (l *Lexer) readString(line, col int) Token {
	l.advance() // skip opening "
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		l.advance()
	}
	val := string(l.input[start:l.pos])
	if l.pos < len(l.input) {
		l.advance() // skip closing "
	}
	return Token{Type: TOKEN_STRING, Value: val, Line: line, Col: col}
}

// readIdent reads an identifier or keyword.
func (l *Lexer) readIdent(line, col int) Token {
	start := l.pos
	for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
		l.advance()
	}
	val := string(l.input[start:l.pos])
	tt := TOKEN_IDENT
	switch val {
	case "model":
		tt = TOKEN_MODEL
	case "datasource":
		tt = TOKEN_DATASOURCE
	case "enum":
		tt = TOKEN_ENUM
	}
	return Token{Type: tt, Value: val, Line: line, Col: col}
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}
