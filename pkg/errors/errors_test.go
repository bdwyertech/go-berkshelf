package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestBerkshelfError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *BerkshelfError
		contains []string
	}{
		{
			name: "basic error",
			err: &BerkshelfError{
				Type:    ErrorTypeValidation,
				Message: "test error",
			},
			contains: []string{"[validation]", "test error"},
		},
		{
			name: "error with cause",
			err: &BerkshelfError{
				Type:    ErrorTypeNetwork,
				Message: "connection failed",
				Cause:   errors.New("timeout"),
			},
			contains: []string{"[network]", "connection failed", "Caused by: timeout"},
		},
		{
			name: "error with context",
			err: &BerkshelfError{
				Type:    ErrorTypeResolution,
				Message: "resolution failed",
				Context: map[string]interface{}{
					"cookbook": "nginx",
					"version":  "1.2.3",
				},
			},
			contains: []string{"[resolution]", "resolution failed", "Context:", "cookbook=nginx", "version=1.2.3"},
		},
		{
			name: "error with suggestions",
			err: &BerkshelfError{
				Type:    ErrorTypeParsing,
				Message: "syntax error",
				Suggestions: []string{
					"Check your syntax",
					"Verify brackets",
				},
			},
			contains: []string{"[parsing]", "syntax error", "Suggestions:", "Check your syntax", "Verify brackets"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Error() = %v, should contain %v", result, expected)
				}
			}
		})
	}
}

func TestBerkshelfError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := &BerkshelfError{
		Type:    ErrorTypeNetwork,
		Message: "wrapper error",
		Cause:   cause,
	}

	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestBerkshelfError_Is(t *testing.T) {
	err1 := &BerkshelfError{Type: ErrorTypeValidation}
	err2 := &BerkshelfError{Type: ErrorTypeValidation}
	err3 := &BerkshelfError{Type: ErrorTypeNetwork}
	regularErr := errors.New("regular error")

	if !err1.Is(err2) {
		t.Error("Expected err1.Is(err2) to be true")
	}

	if err1.Is(err3) {
		t.Error("Expected err1.Is(err3) to be false")
	}

	if err1.Is(regularErr) {
		t.Error("Expected err1.Is(regularErr) to be false")
	}
}

func TestBerkshelfError_WithContext(t *testing.T) {
	err := NewValidationError("test error", nil)
	err.WithContext("key1", "value1")
	err.WithContext("key2", 42)

	if err.Context["key1"] != "value1" {
		t.Errorf("Expected context key1 = value1, got %v", err.Context["key1"])
	}

	if err.Context["key2"] != 42 {
		t.Errorf("Expected context key2 = 42, got %v", err.Context["key2"])
	}
}

func TestBerkshelfError_WithSuggestion(t *testing.T) {
	err := NewValidationError("test error", nil)
	err.WithSuggestion("Try this")
	err.WithSuggestion("Or this")

	if len(err.Suggestions) < 3 { // 1 default + 2 added
		t.Errorf("Expected at least 3 suggestions, got %d", len(err.Suggestions))
	}

	found := false
	for _, suggestion := range err.Suggestions {
		if suggestion == "Try this" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'Try this' in suggestions")
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name        string
		constructor func(string, error) *BerkshelfError
		expectedType ErrorType
	}{
		{"NewValidationError", NewValidationError, ErrorTypeValidation},
		{"NewNetworkError", NewNetworkError, ErrorTypeNetwork},
		{"NewResolutionError", NewResolutionError, ErrorTypeResolution},
		{"NewParsingError", NewParsingError, ErrorTypeParsing},
		{"NewFileSystemError", NewFileSystemError, ErrorTypeFileSystem},
		{"NewAuthenticationError", NewAuthenticationError, ErrorTypeAuthentication},
		{"NewConfigurationError", NewConfigurationError, ErrorTypeConfiguration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cause := errors.New("test cause")
			err := tt.constructor("test message", cause)

			if err.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, err.Type)
			}

			if err.Message != "test message" {
				t.Errorf("Expected message 'test message', got %v", err.Message)
			}

			if err.Cause != cause {
				t.Errorf("Expected cause %v, got %v", cause, err.Cause)
			}

			if len(err.Suggestions) == 0 {
				t.Error("Expected at least one suggestion")
			}
		})
	}
}

func TestErrorCollector(t *testing.T) {
	collector := NewErrorCollector()

	// Test empty collector
	if collector.HasErrors() {
		t.Error("Expected empty collector to have no errors")
	}

	if collector.Error() != "" {
		t.Error("Expected empty collector error message to be empty")
	}

	// Add errors
	err1 := NewValidationError("validation error", nil)
	err2 := NewNetworkError("network error", nil)
	regularErr := errors.New("regular error")

	collector.Add(err1)
	collector.Add(err2)
	collector.Add(regularErr)
	collector.Add(nil) // Should be ignored

	// Test collector with errors
	if !collector.HasErrors() {
		t.Error("Expected collector to have errors")
	}

	if len(collector.Errors()) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(collector.Errors()))
	}

	errorMsg := collector.Error()
	if !strings.Contains(errorMsg, "Multiple errors occurred (3 total)") {
		t.Errorf("Expected multiple errors message, got: %s", errorMsg)
	}

	// Test summary
	summary := collector.Summary()
	if summary[ErrorTypeValidation] != 1 {
		t.Errorf("Expected 1 validation error, got %d", summary[ErrorTypeValidation])
	}

	if summary[ErrorTypeNetwork] != 1 {
		t.Errorf("Expected 1 network error, got %d", summary[ErrorTypeNetwork])
	}

	if summary["unknown"] != 1 {
		t.Errorf("Expected 1 unknown error, got %d", summary["unknown"])
	}
}

func TestErrorCollector_SingleError(t *testing.T) {
	collector := NewErrorCollector()
	err := NewValidationError("single error", nil)
	collector.Add(err)

	errorMsg := collector.Error()
	if strings.Contains(errorMsg, "Multiple errors occurred") {
		t.Errorf("Expected single error message, got: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "single error") {
		t.Errorf("Expected error message to contain 'single error', got: %s", errorMsg)
	}
}
