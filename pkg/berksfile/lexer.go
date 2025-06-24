package berksfile

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// TokenType represents the type of a lexical token
type TokenType int

const (
	// Special tokens
	TokenEOF TokenType = iota
	TokenError
	TokenNewline

	// Literals
	TokenString
	TokenSymbol
	TokenNumber
	TokenComment

	// Keywords
	TokenSource
	TokenCookbook
	TokenMetadata
	TokenGroup
	TokenEnd

	// Operators
	TokenComma
	TokenColon
	TokenArrow // =>
	TokenOpenParen
	TokenCloseParen
	TokenOpenBrace
	TokenCloseBrace
)

var tokenTypeNames = map[TokenType]string{
	TokenEOF:        "EOF",
	TokenError:      "ERROR",
	TokenNewline:    "NEWLINE",
	TokenString:     "STRING",
	TokenSymbol:     "SYMBOL",
	TokenNumber:     "NUMBER",
	TokenComment:    "COMMENT",
	TokenSource:     "SOURCE",
	TokenCookbook:   "COOKBOOK",
	TokenMetadata:   "METADATA",
	TokenGroup:      "GROUP",
	TokenEnd:        "END",
	TokenComma:      "COMMA",
	TokenColon:      "COLON",
	TokenArrow:      "ARROW",
	TokenOpenParen:  "OPEN_PAREN",
	TokenCloseParen: "CLOSE_PAREN",
	TokenOpenBrace:  "OPEN_BRACE",
	TokenCloseBrace: "CLOSE_BRACE",
}

func (t TokenType) String() string {
	if name, ok := tokenTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", t)
}

// Token represents a lexical token
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q) at %d:%d", t.Type, t.Value, t.Line, t.Column)
}

// Lexer tokenizes Berksfile content
type Lexer struct {
	reader *bufio.Reader
	line   int
	column int
	tokens []Token
	errors []error
}

// NewLexer creates a new lexer for the given input
func NewLexer(input io.Reader) *Lexer {
	return &Lexer{
		reader: bufio.NewReader(input),
		line:   1,
		column: 1,
		tokens: make([]Token, 0),
		errors: make([]error, 0),
	}
}

// Tokenize processes the entire input and returns all tokens
func (l *Lexer) Tokenize() ([]Token, error) {
	for {
		token := l.nextToken()
		l.tokens = append(l.tokens, token)
		if token.Type == TokenEOF || token.Type == TokenError {
			break
		}
	}

	if len(l.errors) > 0 {
		return l.tokens, fmt.Errorf("lexer errors: %v", l.errors)
	}

	return l.tokens, nil
}

func (l *Lexer) nextToken() Token {
	l.skipWhitespace()

	ch := l.peek()
	if ch == 0 {
		return Token{Type: TokenEOF, Value: "", Line: l.line, Column: l.column}
	}

	startLine := l.line
	startColumn := l.column

	switch ch {
	case '\n':
		l.read()
		return Token{Type: TokenNewline, Value: "\n", Line: startLine, Column: startColumn}
	case '#':
		return l.readComment()
	case '"', '\'':
		return l.readString()
	case ':':
		l.read()
		if l.peek() == ':' {
			// Handle :: scope operator if needed
			l.read()
			return Token{Type: TokenSymbol, Value: "::", Line: startLine, Column: startColumn}
		}
		// Check if this is a symbol
		if unicode.IsLetter(rune(l.peek())) || l.peek() == '_' {
			return l.readSymbol()
		}
		return Token{Type: TokenColon, Value: ":", Line: startLine, Column: startColumn}
	case ',':
		l.read()
		return Token{Type: TokenComma, Value: ",", Line: startLine, Column: startColumn}
	case '(':
		l.read()
		return Token{Type: TokenOpenParen, Value: "(", Line: startLine, Column: startColumn}
	case ')':
		l.read()
		return Token{Type: TokenCloseParen, Value: ")", Line: startLine, Column: startColumn}
	case '{':
		l.read()
		return Token{Type: TokenOpenBrace, Value: "{", Line: startLine, Column: startColumn}
	case '}':
		l.read()
		return Token{Type: TokenCloseBrace, Value: "}", Line: startLine, Column: startColumn}
	case '=':
		l.read()
		if l.peek() == '>' {
			l.read()
			return Token{Type: TokenArrow, Value: "=>", Line: startLine, Column: startColumn}
		}
		return Token{Type: TokenError, Value: "=", Line: startLine, Column: startColumn}
	default:
		if unicode.IsLetter(rune(ch)) || ch == '_' {
			return l.readIdentifier()
		}
		if unicode.IsDigit(rune(ch)) {
			return l.readNumber()
		}
		l.read()
		return Token{Type: TokenError, Value: string(ch), Line: startLine, Column: startColumn}
	}
}

func (l *Lexer) readComment() Token {
	startLine := l.line
	startColumn := l.column

	var buf bytes.Buffer
	l.read() // consume #

	for {
		ch := l.peek()
		if ch == 0 || ch == '\n' {
			break
		}
		buf.WriteByte(l.read())
	}

	return Token{Type: TokenComment, Value: buf.String(), Line: startLine, Column: startColumn}
}

func (l *Lexer) readString() Token {
	startLine := l.line
	startColumn := l.column

	quote := l.read() // consume opening quote
	var buf bytes.Buffer

	for {
		ch := l.peek()
		if ch == 0 {
			l.errors = append(l.errors, fmt.Errorf("unterminated string at line %d", startLine))
			return Token{Type: TokenError, Value: buf.String(), Line: startLine, Column: startColumn}
		}
		if ch == quote {
			l.read() // consume closing quote
			break
		}
		if ch == '\\' {
			l.read()
			escaped := l.read()
			switch escaped {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case 'r':
				buf.WriteByte('\r')
			case '\\':
				buf.WriteByte('\\')
			case '"', '\'':
				buf.WriteByte(escaped)
			default:
				buf.WriteByte(escaped)
			}
		} else {
			buf.WriteByte(l.read())
		}
	}

	return Token{Type: TokenString, Value: buf.String(), Line: startLine, Column: startColumn}
}

func (l *Lexer) readSymbol() Token {
	startLine := l.line
	startColumn := l.column - 1 // account for already consumed ':'

	var buf bytes.Buffer
	buf.WriteByte(':')

	for {
		ch := l.peek()
		if unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' {
			buf.WriteByte(l.read())
		} else {
			break
		}
	}

	return Token{Type: TokenSymbol, Value: buf.String(), Line: startLine, Column: startColumn}
}

func (l *Lexer) readIdentifier() Token {
	startLine := l.line
	startColumn := l.column

	var buf bytes.Buffer

	for {
		ch := l.peek()
		if unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' {
			buf.WriteByte(l.read())
		} else {
			break
		}
	}

	value := buf.String()
	tokenType := TokenString

	// Check for keywords
	switch value {
	case "source":
		tokenType = TokenSource
	case "cookbook":
		tokenType = TokenCookbook
	case "metadata":
		tokenType = TokenMetadata
	case "group":
		tokenType = TokenGroup
	case "end":
		tokenType = TokenEnd
	}

	return Token{Type: tokenType, Value: value, Line: startLine, Column: startColumn}
}

func (l *Lexer) readNumber() Token {
	startLine := l.line
	startColumn := l.column

	var buf bytes.Buffer

	// Read integer part
	for {
		ch := l.peek()
		if unicode.IsDigit(rune(ch)) {
			buf.WriteByte(l.read())
		} else {
			break
		}
	}

	// Check for decimal
	if l.peek() == '.' {
		nextCh := l.peekAhead(1)
		if unicode.IsDigit(rune(nextCh)) {
			buf.WriteByte(l.read()) // consume '.'
			// Read decimal part
			for {
				ch := l.peek()
				if unicode.IsDigit(rune(ch)) {
					buf.WriteByte(l.read())
				} else {
					break
				}
			}
		}
	}

	return Token{Type: TokenNumber, Value: buf.String(), Line: startLine, Column: startColumn}
}

func (l *Lexer) skipWhitespace() {
	for {
		ch := l.peek()
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.read()
		} else {
			break
		}
	}
}

func (l *Lexer) peek() byte {
	b, err := l.reader.Peek(1)
	if err != nil {
		return 0
	}
	return b[0]
}

func (l *Lexer) peekAhead(n int) byte {
	b, err := l.reader.Peek(n + 1)
	if err != nil || len(b) <= n {
		return 0
	}
	return b[n]
}

func (l *Lexer) read() byte {
	b, err := l.reader.ReadByte()
	if err != nil {
		return 0
	}

	if b == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}

	return b
}

// TokenizeString is a convenience function to tokenize a string
func TokenizeString(input string) ([]Token, error) {
	lexer := NewLexer(strings.NewReader(input))
	return lexer.Tokenize()
}
