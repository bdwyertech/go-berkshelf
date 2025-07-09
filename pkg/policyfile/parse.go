package policyfile

import (
	"fmt"
	"strings"
	
	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
)

// Global variable to store parse errors
var lastParseError error

// ParsePolicyfile parses the input Policyfile.rb DSL and returns a Policyfile struct or error.
// Only parses Berkshelf-equivalent directives: default_source and cookbook
func ParsePolicyfile(input string) (*Policyfile, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		// Return empty but valid Policyfile for empty input
		return &Policyfile{
			DefaultSources: []*berkshelf.SourceLocation{},
			Cookbooks:      []*CookbookDef{},
		}, nil
	}

	var parsePanic any
	defer func() {
		if r := recover(); r != nil {
			parsePanic = r
		}
	}()

	lastParseError = nil
	lexer := NewLexer(input)
	lexer.sourceText = input // Store source text for error reporting
	Result = nil
	yyParse(lexer)

	if parsePanic != nil {
		return nil, fmt.Errorf("panic during parse: %v", parsePanic)
	}
	if lastParseError != nil {
		return nil, lastParseError
	}
	if Result == nil {
		return nil, fmt.Errorf("parse error - Result is nil")
	}

	return Result, nil
}

// Error handling for the lexer
func (l *Lexer) Error(s string) {
	lastParseError = fmt.Errorf("parse error: %s at position %v", s, l.s.Pos())
}
