//go:generate goyacc -o parser.go berksfile.y

package berksfileparser

import (
	"fmt"
	"strings"
	"text/scanner"
	"unicode"
)

var keywords = map[string]int{
	"source":   SOURCE,
	"metadata": METADATA,
	"cookbook": COOKBOOK,
	"group":    GROUP,
	"do":       DO,
	"end":      END,
}

// Global variable to store parse errors
var lastParseError error

type Lexer struct {
	s   scanner.Scanner
	buf struct {
		tok int
		lit string
		n   int
	}
	sourceText string
	tokenLog   []string
}

func NewLexer(src string) *Lexer {
	var l Lexer
	l.s.Init(strings.NewReader(src))
	l.s.Whitespace ^= 1 << '\n' // Don't skip newlines
	l.s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanRawStrings | scanner.ScanComments
	return &l
}

func (l *Lexer) Lex(lval *yySymType) int {
	// Use buffered token if any
	if l.buf.n != 0 {
		l.buf.n = 0
		lval.str = l.buf.lit
		return l.buf.tok
	}

	l.tokenLog = append(l.tokenLog, l.s.TokenText())
	if len(l.tokenLog) > 5 {
		l.tokenLog = l.tokenLog[1:]
	}

	for {
		r := l.s.Scan()
		switch r {
		case scanner.EOF:
			return 0
		case scanner.String, scanner.RawString:
			lval.str = l.s.TokenText()
			return STRING
		case scanner.Ident:
			ident := l.s.TokenText()
			lower := strings.ToLower(ident)
			if tok, isKeyword := keywords[lower]; isKeyword {
				return tok
			}
			lval.str = ident
			return IDENT
		case scanner.Comment:
			// Skip comments and continue lexing
			continue
		case ':':
			// Always return COLON token - let the parser handle symbol syntax
			lval.str = ":"
			return COLON
		case ',':
			lval.str = ","
			return COMMA
		case '{':
			lval.str = "{"
			return LBRACE
		case '}':
			lval.str = "}"
			return RBRACE
		case '=':
			next := l.s.Peek()
			if next == '>' {
				_ = l.s.Next() // consume '>'
				lval.str = "=>"
				return HASHROCKET
			}
			// If just '=', ignore it or handle as needed
			continue
		case '\n':
			lval.str = "\n"
			return NEWLINE
		case ';':
			// ignore semicolons
			continue
		case '#':
			// Handle comments manually - scan until end of line
			for {
				peek := l.s.Peek()
				if peek == '\n' || peek == scanner.EOF {
					break
				}
				_ = l.s.Next()
			}
			continue
		case '\'':
			// Handle single-quoted strings manually
			var str strings.Builder
			str.WriteRune('\'') // Include the opening quote

			for {
				next := l.s.Next()
				if next == scanner.EOF {
					// Unterminated string
					break
				}
				str.WriteRune(next)
				if next == '\'' {
					// Found closing quote
					break
				}
				if next == '\\' {
					// Handle escape sequences
					escaped := l.s.Next()
					if escaped != scanner.EOF {
						str.WriteRune(escaped)
					}
				}
			}
			lval.str = str.String()
			return STRING
		default:
			if unicode.IsSpace(r) {
				// ignore other spaces
				continue
			} else {
				// Unexpected char - for debugging, you might want to continue instead of panic
				fmt.Printf("Warning: unexpected char: %q at %s\n", r, l.s.Pos())
				continue
			}
		}
	}
}

func (l *Lexer) Error(msg string) {
	pos := l.s.Pos()
	offset := pos.Offset

	lineStart := max(offset-pos.Column+1, 0)
	line := ""
	if offset < len(l.sourceText) {
		end := strings.IndexByte(l.sourceText[offset:], '\n')
		if end != -1 {
			line = l.sourceText[lineStart : offset+end]
		}
	}
	fmt.Println(l.sourceText)

	lastParseError = fmt.Errorf(
		"parse error at line %d, column %d: %s\n%s\n%s^",
		pos.Line,
		pos.Column,
		msg,
		line,
		strings.Repeat(" ", pos.Column-1),
	)
}

// GetLastError returns the last parse error
func GetLastError() error {
	return lastParseError
}

// ClearLastError clears the last parse error
func ClearLastError() {
	lastParseError = nil
}
