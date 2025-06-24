package berksfile

import (
	"strings"
	"testing"
)

func TestLexer_BasicTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:     "empty file",
			input:    "",
			expected: []TokenType{TokenEOF},
		},
		{
			name:     "comment",
			input:    "# This is a comment",
			expected: []TokenType{TokenComment, TokenEOF},
		},
		{
			name:     "source declaration",
			input:    "source 'https://supermarket.chef.io'",
			expected: []TokenType{TokenSource, TokenString, TokenEOF},
		},
		{
			name:     "cookbook with version",
			input:    "cookbook 'nginx', '~> 2.7.6'",
			expected: []TokenType{TokenCookbook, TokenString, TokenComma, TokenString, TokenEOF},
		},
		{
			name:     "metadata directive",
			input:    "metadata",
			expected: []TokenType{TokenMetadata, TokenEOF},
		},
		{
			name:     "group block",
			input:    "group :test do\nend",
			expected: []TokenType{TokenGroup, TokenSymbol, TokenString, TokenNewline, TokenEnd, TokenEOF},
		},
		{
			name:     "cookbook with hash options",
			input:    "cookbook 'private', git: 'git@github.com:user/repo.git'",
			expected: []TokenType{TokenCookbook, TokenString, TokenComma, TokenString, TokenColon, TokenString, TokenEOF},
		},
		{
			name:     "multiple lines",
			input:    "source 'https://supermarket.chef.io'\n\ncookbook 'nginx'",
			expected: []TokenType{TokenSource, TokenString, TokenNewline, TokenNewline, TokenCookbook, TokenString, TokenEOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, expectedType := range tt.expected {
				if tokens[i].Type != expectedType {
					t.Errorf("token %d: expected type %v, got %v", i, expectedType, tokens[i].Type)
				}
			}
		})
	}
}

func TestLexer_TokenValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "string values",
			input: `"double quoted" 'single quoted'`,
			expected: []Token{
				{Type: TokenString, Value: "double quoted", Line: 1, Column: 1},
				{Type: TokenString, Value: "single quoted", Line: 1, Column: 17},
				{Type: TokenEOF, Value: "", Line: 1, Column: 32},
			},
		},
		{
			name:  "symbol values",
			input: `:test :production`,
			expected: []Token{
				{Type: TokenSymbol, Value: ":test", Line: 1, Column: 1},
				{Type: TokenSymbol, Value: ":production", Line: 1, Column: 7},
				{Type: TokenEOF, Value: "", Line: 1, Column: 18},
			},
		},
		{
			name:  "comment value",
			input: "# This is a comment\ncookbook 'test'",
			expected: []Token{
				{Type: TokenComment, Value: " This is a comment", Line: 1, Column: 1},
				{Type: TokenNewline, Value: "\n", Line: 1, Column: 20},
				{Type: TokenCookbook, Value: "cookbook", Line: 2, Column: 1},
				{Type: TokenString, Value: "test", Line: 2, Column: 10},
				{Type: TokenEOF, Value: "", Line: 2, Column: 16},
			},
		},
		{
			name:  "escaped strings",
			input: `"line1\nline2" 'tab\there'`,
			expected: []Token{
				{Type: TokenString, Value: "line1\nline2", Line: 1, Column: 1},
				{Type: TokenString, Value: "tab\there", Line: 1, Column: 16},
				{Type: TokenEOF, Value: "", Line: 1, Column: 27},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, expected := range tt.expected {
				actual := tokens[i]
				if actual.Type != expected.Type {
					t.Errorf("token %d: expected type %v, got %v", i, expected.Type, actual.Type)
				}
				if actual.Value != expected.Value {
					t.Errorf("token %d: expected value %q, got %q", i, expected.Value, actual.Value)
				}
				if actual.Line != expected.Line {
					t.Errorf("token %d: expected line %d, got %d", i, expected.Line, actual.Line)
				}
				if actual.Column != expected.Column {
					t.Errorf("token %d: expected column %d, got %d", i, expected.Column, actual.Column)
				}
			}
		})
	}
}

func TestLexer_ComplexBerksfile(t *testing.T) {
	input := `# Berksfile for myapp
source 'https://supermarket.chef.io'

metadata

cookbook 'nginx', '~> 2.7.6'
cookbook 'mysql', '>= 5.0.0'

group :integration do
  cookbook 'test-kitchen'
end

cookbook 'private', git: 'git@github.com:user/repo.git', branch: 'master'
`

	lexer := NewLexer(strings.NewReader(input))
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that we got a reasonable number of tokens
	if len(tokens) < 20 {
		t.Errorf("expected at least 20 tokens, got %d", len(tokens))
	}

	// Verify specific patterns
	foundSource := false
	foundMetadata := false
	foundGroup := false

	for _, token := range tokens {
		switch token.Type {
		case TokenSource:
			foundSource = true
		case TokenMetadata:
			foundMetadata = true
		case TokenGroup:
			foundGroup = true
		}
	}

	if !foundSource {
		t.Error("expected to find 'source' token")
	}
	if !foundMetadata {
		t.Error("expected to find 'metadata' token")
	}
	if !foundGroup {
		t.Error("expected to find 'group' token")
	}
}

func TestLexer_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "unterminated string",
			input:       `"unterminated`,
			shouldError: true,
		},
		{
			name:        "valid input",
			input:       `cookbook "test"`,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			_, err := lexer.Tokenize()

			if tt.shouldError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
