package policyfile

import (
	"testing"
)

func TestLexer_Keywords(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"default_source", []int{DEFAULT_SOURCE}},
		{"cookbook", []int{COOKBOOK}},
		{"default_source cookbook", []int{DEFAULT_SOURCE, COOKBOOK}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			var lval yySymType

			for i, expectedToken := range tt.expected {
				token := lexer.Lex(&lval)
				if token != expectedToken {
					t.Errorf("Token %d: expected %d, got %d", i, expectedToken, token)
				}
			}
		})
	}
}

func TestLexer_Symbols(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{":supermarket", ":supermarket"},
		{":chef_server", ":chef_server"},
		{":chef_repo", ":chef_repo"},
		{":artifactory", ":artifactory"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			var lval yySymType

			token := lexer.Lex(&lval)
			if token != SYMBOL {
				t.Errorf("Expected SYMBOL token, got %d", token)
			}

			if lval.str != tt.expected {
				t.Errorf("Expected symbol %s, got %s", tt.expected, lval.str)
			}
		})
	}
}

func TestLexer_Strings(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"nginx"`, `"nginx"`},
		{`"~> 2.7"`, `"~> 2.7"`},
		{`"https://supermarket.example.com"`, `"https://supermarket.example.com"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			var lval yySymType

			token := lexer.Lex(&lval)
			if token != STRING {
				t.Errorf("Expected STRING token, got %d", token)
			}

			if lval.str != tt.expected {
				t.Errorf("Expected string %s, got %s", tt.expected, lval.str)
			}
		})
	}
}

func TestLexer_Comments(t *testing.T) {
	input := `# This is a comment
default_source :supermarket`

	lexer := NewLexer(input)
	var lval yySymType

	// Should get NEWLINE, DEFAULT_SOURCE, SYMBOL, EOF
	expectedTokens := []int{NEWLINE, DEFAULT_SOURCE, SYMBOL}

	for i, expectedToken := range expectedTokens {
		token := lexer.Lex(&lval)
		if token != expectedToken {
			t.Errorf("Token %d: expected %d, got %d", i, expectedToken, token)
		}
	}
	
	// Check for EOF
	token := lexer.Lex(&lval)
	if token != 0 {
		t.Errorf("Expected EOF (0), got %d", token)
	}
}

func TestLexer_ComplexInput(t *testing.T) {
	input := `default_source :supermarket, "https://private.supermarket.com"
cookbook "nginx", "~> 2.7"`

	lexer := NewLexer(input)
	var lval yySymType

	// Expected sequence: DEFAULT_SOURCE, SYMBOL, COMMA, STRING, NEWLINE, COOKBOOK, STRING, COMMA, STRING, EOF
	expectedTokens := []int{
		DEFAULT_SOURCE, SYMBOL, COMMA, STRING, NEWLINE,
		COOKBOOK, STRING, COMMA, STRING, 0,
	}

	for i, expectedToken := range expectedTokens {
		token := lexer.Lex(&lval)
		if token != expectedToken {
			t.Errorf("Token %d: expected %d, got %d (value: %s)", i, expectedToken, token, lval.str)
		}
	}
}

func TestLexer_Whitespace(t *testing.T) {
	input := `   default_source    :supermarket   
   cookbook   "nginx"   `

	lexer := NewLexer(input)
	var lval yySymType

	// Whitespace should be ignored except for newlines
	expectedTokens := []int{DEFAULT_SOURCE, SYMBOL, NEWLINE, COOKBOOK, STRING, 0}

	for i, expectedToken := range expectedTokens {
		token := lexer.Lex(&lval)
		if token != expectedToken {
			t.Errorf("Token %d: expected %d, got %d", i, expectedToken, token)
		}
	}
}
