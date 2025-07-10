package berksfile

import (
	"fmt"
	"strings"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
)

// Parse parses the input Berksfile DSL and returns a Berksfile struct or error.
func Parse(input string) (*Berksfile, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		// Return empty but valid Berksfile for empty input
		return &Berksfile{
			Sources:     []*berkshelf.SourceLocation{},
			Cookbooks:   []*CookbookDef{},
			Groups:      make(map[string][]*CookbookDef),
			HasMetadata: false,
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
