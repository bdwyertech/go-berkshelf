//go:generate goyacc -o parser.go policyfile.y

package policyfile

import (
	"fmt"
	"strings"
	"text/scanner"
	"unicode"
)

var keywords = map[string]int{
	"default_source": DEFAULT_SOURCE,
	"cookbook":       COOKBOOK,
}

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
	l.sourceText = src
	return &l
}

func (l *Lexer) Lex(lval *yySymType) int {
	// Use buffered token if any
	if l.buf.n != 0 {
		l.buf.n = 0
		lval.str = l.buf.lit
		return l.buf.tok
	}

	tok := l.s.Scan()
	lit := l.s.TokenText()

	// Log token for debugging
	l.tokenLog = append(l.tokenLog, fmt.Sprintf("%s:%s", scanner.TokenString(tok), lit))

	switch tok {
	case scanner.EOF:
		return 0
	case scanner.Ident:
		// Check for keywords
		if keywordTok, ok := keywords[lit]; ok {
			lval.str = lit
			return keywordTok
		}
		lval.str = lit
		return IDENTIFIER
	case scanner.String, scanner.RawString:
		lval.str = lit
		return STRING
	case '\n':
		return NEWLINE
	case ',':
		return COMMA
	case ':':
		// Handle symbols like :supermarket, :chef_server, etc.
		nextTok := l.s.Scan()
		if nextTok == scanner.Ident {
			symbolName := l.s.TokenText()
			lval.str = ":" + symbolName
			return SYMBOL
		} else {
			// Buffer the next token and return standalone colon
			if nextTok == scanner.String || nextTok == scanner.RawString {
				l.buf.tok = STRING
			} else if nextTok == scanner.Ident {
				l.buf.tok = IDENTIFIER
			} else {
				l.buf.tok = int(nextTok)
			}
			l.buf.lit = l.s.TokenText()
			l.buf.n = 1
			lval.str = lit
			return COLON
		}
	case '#':
		// Skip comments - read until end of line
		for {
			ch := l.s.Next()
			if ch == '\n' || ch == scanner.EOF {
				if ch == '\n' {
					return NEWLINE
				}
				return 0
			}
		}
	default:
		if unicode.IsSpace(rune(tok)) {
			// Skip other whitespace and continue
			return l.Lex(lval)
		}
		lval.str = lit
		return int(tok)
	}
}
