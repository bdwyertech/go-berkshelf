package berksfileparser

import (
	"fmt"
	"strings"
)

// ParseBerksfile parses the input Berksfile DSL and returns a Berksfile struct or error.
func ParseBerksfile(input string) (*Berksfile, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, fmt.Errorf("empty input")
	}

	var parsePanic any
	defer func() {
		if r := recover(); r != nil {
			parsePanic = r
		}
	}()

	lastParseError = nil
	lexer := NewLexer(input)
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
	// Consider empty Berksfile as error (no sources, cookbooks, or groups)
	if len(Result.Sources) == 0 && len(Result.Cookbooks) == 0 && len(Result.Groups) == 0 {
		return nil, fmt.Errorf("no valid content found")
	}
	return Result, nil
}
