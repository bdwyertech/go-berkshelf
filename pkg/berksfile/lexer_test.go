package berksfile

import (
	"testing"
)

// TokenInfo represents a token with its type and value for testing
type TokenInfo struct {
	Type  int
	Value string
}

// Helper function to collect all tokens from a lexer
func collectTokens(lexer *Lexer) []TokenInfo {
	var tokens []TokenInfo
	var lval yySymType

	for {
		tok := lexer.Lex(&lval)
		if tok == 0 { // EOF
			break
		}
		tokens = append(tokens, TokenInfo{
			Type:  tok,
			Value: lval.str,
		})
	}

	return tokens
}

func TestLexer_BasicTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenInfo
	}{
		{
			name:     "empty file",
			input:    "",
			expected: []TokenInfo{},
		},
		{
			name:  "source declaration",
			input: "source 'https://supermarket.chef.io'",
			expected: []TokenInfo{
				{Type: SOURCE, Value: ""},
				{Type: STRING, Value: "'https://supermarket.chef.io'"},
			},
		},
		{
			name:  "cookbook with version",
			input: "cookbook 'nginx', '~> 2.7.6'",
			expected: []TokenInfo{
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: "'nginx'"},
				{Type: COMMA, Value: ","},
				{Type: STRING, Value: "'~> 2.7.6'"},
			},
		},
		{
			name:  "metadata directive",
			input: "metadata",
			expected: []TokenInfo{
				{Type: METADATA, Value: ""},
			},
		},
		{
			name:  "group block",
			input: "group :test do\nend",
			expected: []TokenInfo{
				{Type: GROUP, Value: ""},
				{Type: COLON, Value: ":"},
				{Type: IDENT, Value: "test"},
				{Type: DO, Value: ""},
				{Type: NEWLINE, Value: "\n"},
				{Type: END, Value: ""},
			},
		},
		{
			name:  "cookbook with git source",
			input: "cookbook 'private', git: 'git@github.com:user/repo.git'",
			expected: []TokenInfo{
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: "'private'"},
				{Type: COMMA, Value: ","},
				{Type: IDENT, Value: "git"},
				{Type: COLON, Value: ":"},
				{Type: STRING, Value: "'git@github.com:user/repo.git'"},
			},
		},
		{
			name:  "cookbook with hash options",
			input: "cookbook 'test', { git: 'repo.git', branch: 'master' }",
			expected: []TokenInfo{
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: "'test'"},
				{Type: COMMA, Value: ","},
				{Type: LBRACE, Value: "{"},
				{Type: IDENT, Value: "git"},
				{Type: COLON, Value: ":"},
				{Type: STRING, Value: "'repo.git'"},
				{Type: COMMA, Value: ","},
				{Type: IDENT, Value: "branch"},
				{Type: COLON, Value: ":"},
				{Type: STRING, Value: "'master'"},
				{Type: RBRACE, Value: "}"},
			},
		},
		{
			name:  "hashrocket syntax",
			input: "cookbook 'test', :git => 'repo.git'",
			expected: []TokenInfo{
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: "'test'"},
				{Type: COMMA, Value: ","},
				{Type: COLON, Value: ":"},
				{Type: IDENT, Value: "git"},
				{Type: HASHROCKET, Value: "=>"},
				{Type: STRING, Value: "'repo.git'"},
			},
		},
		{
			name:  "multiple lines with newlines",
			input: "source 'https://supermarket.chef.io'\n\ncookbook 'nginx'",
			expected: []TokenInfo{
				{Type: SOURCE, Value: ""},
				{Type: STRING, Value: "'https://supermarket.chef.io'"},
				{Type: NEWLINE, Value: "\n"},
				{Type: NEWLINE, Value: "\n"},
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: "'nginx'"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens := collectTokens(lexer)

			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, expected := range tt.expected {
				actual := tokens[i]
				if actual.Type != expected.Type {
					t.Errorf("token %d: expected type %d, got %d", i, expected.Type, actual.Type)
				}
				if expected.Value != "" && actual.Value != expected.Value {
					t.Errorf("token %d: expected value %q, got %q", i, expected.Value, actual.Value)
				}
			}
		})
	}
}

func TestLexer_StringHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenInfo
	}{
		{
			name:  "double quoted strings",
			input: `cookbook "nginx"`,
			expected: []TokenInfo{
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: `"nginx"`},
			},
		},
		{
			name:  "single quoted strings",
			input: `cookbook 'nginx'`,
			expected: []TokenInfo{
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: `'nginx'`},
			},
		},
		{
			name:  "mixed quotes",
			input: `cookbook "nginx", '~> 2.7'`,
			expected: []TokenInfo{
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: `"nginx"`},
				{Type: COMMA, Value: ","},
				{Type: STRING, Value: `'~> 2.7'`},
			},
		},
		{
			name:  "strings with spaces",
			input: `cookbook 'my cookbook name'`,
			expected: []TokenInfo{
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: `'my cookbook name'`},
			},
		},
		{
			name:  "escaped strings",
			input: `cookbook 'test\'s cookbook'`,
			expected: []TokenInfo{
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: `'test\'s cookbook'`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens := collectTokens(lexer)

			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, expected := range tt.expected {
				actual := tokens[i]
				if actual.Type != expected.Type {
					t.Errorf("token %d: expected type %d, got %d", i, expected.Type, actual.Type)
				}
				if actual.Value != expected.Value {
					t.Errorf("token %d: expected value %q, got %q", i, expected.Value, actual.Value)
				}
			}
		})
	}
}

func TestLexer_Keywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"source keyword", "source", SOURCE},
		{"metadata keyword", "metadata", METADATA},
		{"cookbook keyword", "cookbook", COOKBOOK},
		{"group keyword", "group", GROUP},
		{"do keyword", "do", DO},
		{"end keyword", "end", END},
		{"Source uppercase", "Source", SOURCE}, // Should be case insensitive
		{"COOKBOOK uppercase", "COOKBOOK", COOKBOOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			var lval yySymType
			tok := lexer.Lex(&lval)

			if tok != tt.expected {
				t.Errorf("expected token type %d, got %d", tt.expected, tok)
			}
		})
	}
}

func TestLexer_Identifiers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple identifier", "nginx", "nginx"},
		{"identifier with numbers", "nginx2", "nginx2"},
		{"identifier with underscores", "my_cookbook", "my_cookbook"},
		// Note: dashes are not part of identifiers in this lexer
		// They would be tokenized separately
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			var lval yySymType
			tok := lexer.Lex(&lval)

			if tok != IDENT {
				t.Errorf("expected IDENT token, got %d", tok)
			}
			if lval.str != tt.expected {
				t.Errorf("expected identifier %q, got %q", tt.expected, lval.str)
			}
		})
	}
}

func TestLexer_Comments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenInfo
	}{
		{
			name:  "line comment",
			input: "# This is a comment\ncookbook 'test'",
			expected: []TokenInfo{
				{Type: NEWLINE, Value: "\n"},
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: "'test'"},
			},
		},
		{
			name:  "comment at end of line",
			input: "cookbook 'test' # inline comment",
			expected: []TokenInfo{
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: "'test'"},
			},
		},
		{
			name:  "multiple comments",
			input: "# Comment 1\n# Comment 2\ncookbook 'test'",
			expected: []TokenInfo{
				{Type: NEWLINE, Value: "\n"},
				{Type: NEWLINE, Value: "\n"},
				{Type: COOKBOOK, Value: ""},
				{Type: STRING, Value: "'test'"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens := collectTokens(lexer)

			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, expected := range tt.expected {
				actual := tokens[i]
				if actual.Type != expected.Type {
					t.Errorf("token %d: expected type %d, got %d", i, expected.Type, actual.Type)
				}
				if expected.Value != "" && actual.Value != expected.Value {
					t.Errorf("token %d: expected value %q, got %q", i, expected.Value, actual.Value)
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

	lexer := NewLexer(input)
	tokens := collectTokens(lexer)

	// Check that we got a reasonable number of tokens
	if len(tokens) < 20 {
		t.Errorf("expected at least 20 tokens, got %d", len(tokens))
	}

	// Verify specific patterns
	foundSource := false
	foundMetadata := false
	foundGroup := false
	foundGit := false

	for _, token := range tokens {
		switch token.Type {
		case SOURCE:
			foundSource = true
		case METADATA:
			foundMetadata = true
		case GROUP:
			foundGroup = true
		case IDENT:
			if token.Value == "git" {
				foundGit = true
			}
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
	if !foundGit {
		t.Error("expected to find 'git' identifier")
	}
}

func TestLexer_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "valid cookbook",
			input:       `cookbook "test"`,
			shouldError: false,
		},
		{
			name:        "valid source",
			input:       `source "https://supermarket.chef.io"`,
			shouldError: false,
		},
		{
			name:        "valid group",
			input:       "group :test do\nend",
			shouldError: false,
		},
		{
			name:        "incomplete group",
			input:       `group :test`,
			shouldError: true,
		},
		{
			name:        "unterminated group",
			input:       "group :test do\ncookbook 'test'",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any previous errors
			ClearLastError()

			_, err := Parse(tt.input)

			if tt.shouldError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLexer_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenInfo
	}{
		{
			name:  "colon",
			input: ":",
			expected: []TokenInfo{
				{Type: COLON, Value: ":"},
			},
		},
		{
			name:  "comma",
			input: ",",
			expected: []TokenInfo{
				{Type: COMMA, Value: ","},
			},
		},
		{
			name:  "braces",
			input: "{}",
			expected: []TokenInfo{
				{Type: LBRACE, Value: "{"},
				{Type: RBRACE, Value: "}"},
			},
		},
		{
			name:  "hashrocket",
			input: "=>",
			expected: []TokenInfo{
				{Type: HASHROCKET, Value: "=>"},
			},
		},
		{
			name:  "newline",
			input: "\n",
			expected: []TokenInfo{
				{Type: NEWLINE, Value: "\n"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens := collectTokens(lexer)

			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, expected := range tt.expected {
				actual := tokens[i]
				if actual.Type != expected.Type {
					t.Errorf("token %d: expected type %d, got %d", i, expected.Type, actual.Type)
				}
				if actual.Value != expected.Value {
					t.Errorf("token %d: expected value %q, got %q", i, expected.Value, actual.Value)
				}
			}
		})
	}
}
