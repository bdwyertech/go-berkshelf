package berksfile

import (
	"fmt"
	"strings"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

// Parser parses tokenized Berksfile content into a structured representation
type Parser struct {
	tokens   []Token
	position int
	errors   []error
}

// NewParser creates a new parser for the given tokens
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens: tokens,
		errors: make([]error, 0),
	}
}

// Parse processes the tokens and returns a parsed Berksfile
func (p *Parser) Parse() (*Berksfile, error) {
	berksfile := &Berksfile{
		Sources:   make([]string, 0),
		Cookbooks: make([]*CookbookDef, 0),
		Groups:    make(map[string][]*CookbookDef),
	}

	for p.position < len(p.tokens) {
		token := p.current()

		switch token.Type {
		case TokenSource:
			if err := p.parseSource(berksfile); err != nil {
				p.errors = append(p.errors, err)
			}
		case TokenCookbook:
			if cookbook, err := p.parseCookbook(); err != nil {
				p.errors = append(p.errors, err)
			} else if cookbook != nil {
				berksfile.Cookbooks = append(berksfile.Cookbooks, cookbook)
			}
		case TokenMetadata:
			berksfile.HasMetadata = true
			p.advance()
		case TokenGroup:
			if err := p.parseGroup(berksfile); err != nil {
				p.errors = append(p.errors, err)
			}
		case TokenComment, TokenNewline:
			p.advance()
		case TokenEOF:
			p.advance()
		default:
			p.errors = append(p.errors, fmt.Errorf("unexpected token %s at line %d", token.Type, token.Line))
			p.advance()
		}
	}

	if len(p.errors) > 0 {
		return berksfile, fmt.Errorf("parser errors: %v", p.errors)
	}

	return berksfile, nil
}

func (p *Parser) parseSource(berksfile *Berksfile) error {
	p.advance() // consume 'source'

	if p.current().Type != TokenString {
		return fmt.Errorf("expected string after 'source', got %s", p.current().Type)
	}

	berksfile.Sources = append(berksfile.Sources, p.current().Value)
	p.advance()

	// Skip any trailing newlines
	for p.current().Type == TokenNewline {
		p.advance()
	}

	return nil
}

func (p *Parser) parseCookbook() (*CookbookDef, error) {
	p.advance() // consume 'cookbook'

	if p.current().Type != TokenString {
		return nil, fmt.Errorf("expected cookbook name, got %s", p.current().Type)
	}

	cookbook := &CookbookDef{
		Name:   p.current().Value,
		Groups: make([]string, 0),
	}
	p.advance()

	// Set default constraint (any version) if none is specified
	defaultConstraint, err := ParseConstraint("")
	if err != nil {
		return nil, fmt.Errorf("failed to create default constraint: %v", err)
	}
	cookbook.Constraint = defaultConstraint

	// Parse optional version constraint and/or options
	if p.current().Type == TokenComma {
		p.advance()

		// Check if this is a version constraint or options
		// Version constraints are strings that start with operators or digits
		if p.current().Type == TokenString {
			constraintStr := p.current().Value
			// Check if this looks like a version constraint
			if isVersionConstraint(constraintStr) {
				constraint, err := ParseConstraint(constraintStr)
				if err != nil {
					return nil, fmt.Errorf("invalid constraint %q: %v", constraintStr, err)
				}
				cookbook.Constraint = constraint
				p.advance()

				// Check for options after version constraint
				if p.current().Type == TokenComma {
					p.advance()
				}
			}
		}

		// Parse options (key: value pairs)
		if p.current().Type == TokenString && p.peek().Type == TokenColon {
			if err := p.parseCookbookOptions(cookbook); err != nil {
				return nil, err
			}
		}
	}

	// Skip any trailing newlines
	for p.current().Type == TokenNewline {
		p.advance()
	}

	return cookbook, nil
}

func (p *Parser) parseCookbookOptions(cookbook *CookbookDef) error {
	source := &berkshelf.SourceLocation{
		Options: make(map[string]any),
	}

	for p.position < len(p.tokens) {
		token := p.current()

		// Check for end of options
		if token.Type == TokenNewline || token.Type == TokenEOF {
			break
		}

		// Parse key: value pairs
		if token.Type == TokenString {
			key := token.Value
			p.advance()

			if p.current().Type != TokenColon {
				return fmt.Errorf("expected ':' after option key %q", key)
			}
			p.advance()

			if p.current().Type != TokenString {
				return fmt.Errorf("expected value for option %q", key)
			}
			value := p.current().Value
			p.advance()

			// Handle specific source types
			switch key {
			case "git":
				source.Type = "git"
				source.URL = value
			case "path":
				source.Type = "path"
				source.Path = value
			case "github":
				source.Type = "git"
				source.URL = fmt.Sprintf("https://github.com/%s.git", value)
			case "ref":
				source.Ref = value
			case "branch":
				source.Options["branch"] = value
			case "tag":
				source.Options["tag"] = value
			default:
				source.Options[key] = value
			}

			// Check for comma separator
			if p.current().Type == TokenComma {
				p.advance()
			}
		} else {
			break
		}
	}

	if source.Type != "" {
		cookbook.Source = source
	}

	return nil
}

func (p *Parser) parseGroup(berksfile *Berksfile) error {
	p.advance() // consume 'group'

	// Parse group name(s)
	groupNames := make([]string, 0)

	for {
		token := p.current()
		if token.Type == TokenSymbol {
			groupNames = append(groupNames, strings.TrimPrefix(token.Value, ":"))
			p.advance()

			if p.current().Type == TokenComma {
				p.advance()
			} else {
				break
			}
		} else {
			break
		}
	}

	if len(groupNames) == 0 {
		return fmt.Errorf("expected group name after 'group'")
	}

	// Skip 'do' if present
	if p.current().Type == TokenString && p.current().Value == "do" {
		p.advance()
	}

	// Skip newlines
	for p.current().Type == TokenNewline {
		p.advance()
	}

	// Parse cookbooks in group
	for p.position < len(p.tokens) {
		token := p.current()

		if token.Type == TokenEnd {
			p.advance()
			break
		}

		if token.Type == TokenCookbook {
			cookbook, err := p.parseCookbook()
			if err != nil {
				p.errors = append(p.errors, err)
			} else if cookbook != nil {
				// Add to each group
				for _, groupName := range groupNames {
					cookbook.Groups = append(cookbook.Groups, groupName)
					if berksfile.Groups[groupName] == nil {
						berksfile.Groups[groupName] = make([]*CookbookDef, 0)
					}
					berksfile.Groups[groupName] = append(berksfile.Groups[groupName], cookbook)
				}
				berksfile.Cookbooks = append(berksfile.Cookbooks, cookbook)
			}
		} else if token.Type == TokenComment || token.Type == TokenNewline {
			p.advance()
		} else {
			p.errors = append(p.errors, fmt.Errorf("unexpected token %s in group", token.Type))
			p.advance()
		}
	}

	// Skip any trailing newlines
	for p.current().Type == TokenNewline {
		p.advance()
	}

	return nil
}

func (p *Parser) current() Token {
	if p.position >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.position]
}

func (p *Parser) advance() {
	if p.position < len(p.tokens) {
		p.position++
	}
}

func (p *Parser) peek() Token {
	if p.position+1 >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.position+1]
}

// ParseString is a convenience function to parse a Berksfile string
func ParseString(input string) (*Berksfile, error) {
	tokens, err := TokenizeString(input)
	if err != nil {
		return nil, err
	}

	parser := NewParser(tokens)
	return parser.Parse()
}

// isVersionConstraint checks if a string looks like a version constraint
func isVersionConstraint(s string) bool {
	// Version constraints typically start with operators or digits
	if len(s) == 0 {
		return false
	}

	// Check for common version constraint patterns
	// Operators: =, !=, >, <, >=, <=, ~>
	if strings.HasPrefix(s, "=") || strings.HasPrefix(s, "!") ||
		strings.HasPrefix(s, ">") || strings.HasPrefix(s, "<") ||
		strings.HasPrefix(s, "~>") {
		return true
	}

	// Check if it starts with a digit (plain version number)
	if s[0] >= '0' && s[0] <= '9' {
		return true
	}

	return false
}
