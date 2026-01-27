package errors

import (
	"fmt"
	"strings"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeNetwork        ErrorType = "network"
	ErrorTypeResolution     ErrorType = "resolution"
	ErrorTypeParsing        ErrorType = "parsing"
	ErrorTypeFileSystem     ErrorType = "filesystem"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeConfiguration  ErrorType = "configuration"
)

// BerkshelfError represents a structured error with context
type BerkshelfError struct {
	Type        ErrorType
	Message     string
	Cause       error
	Context     map[string]interface{}
	Suggestions []string
}

// Error implements the error interface
func (e *BerkshelfError) Error() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("[%s] %s", e.Type, e.Message))

	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("Caused by: %s", e.Cause.Error()))
	}

	if len(e.Context) > 0 {
		var contextParts []string
		for k, v := range e.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", k, v))
		}
		parts = append(parts, fmt.Sprintf("Context: %s", strings.Join(contextParts, ", ")))
	}

	if len(e.Suggestions) > 0 {
		parts = append(parts, fmt.Sprintf("Suggestions: %s", strings.Join(e.Suggestions, "; ")))
	}

	return strings.Join(parts, "\n")
}

// Unwrap returns the underlying cause
func (e *BerkshelfError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches a target error type
func (e *BerkshelfError) Is(target error) bool {
	if t, ok := target.(*BerkshelfError); ok {
		return e.Type == t.Type
	}
	return false
}

// NewValidationError creates a validation error
func NewValidationError(message string, cause error) *BerkshelfError {
	return &BerkshelfError{
		Type:    ErrorTypeValidation,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
		Suggestions: []string{
			"Check your Berksfile syntax",
			"Verify cookbook names and version constraints",
		},
	}
}

// NewNetworkError creates a network error
func NewNetworkError(message string, cause error) *BerkshelfError {
	return &BerkshelfError{
		Type:    ErrorTypeNetwork,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
		Suggestions: []string{
			"Check your internet connection",
			"Verify source URLs are accessible",
			"Check if you need authentication credentials",
		},
	}
}

// NewResolutionError creates a dependency resolution error
func NewResolutionError(message string, cause error) *BerkshelfError {
	return &BerkshelfError{
		Type:    ErrorTypeResolution,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
		Suggestions: []string{
			"Check version constraints for conflicts",
			"Verify all cookbooks are available in configured sources",
			"Consider updating version constraints to be less restrictive",
		},
	}
}

// NewParsingError creates a parsing error
func NewParsingError(message string, cause error) *BerkshelfError {
	return &BerkshelfError{
		Type:    ErrorTypeParsing,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
		Suggestions: []string{
			"Check Berksfile syntax",
			"Verify all quotes and brackets are properly closed",
			"Check for unsupported Ruby syntax",
		},
	}
}

// NewFileSystemError creates a filesystem error
func NewFileSystemError(message string, cause error) *BerkshelfError {
	return &BerkshelfError{
		Type:    ErrorTypeFileSystem,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
		Suggestions: []string{
			"Check file and directory permissions",
			"Verify paths exist and are accessible",
			"Ensure sufficient disk space",
		},
	}
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(message string, cause error) *BerkshelfError {
	return &BerkshelfError{
		Type:    ErrorTypeAuthentication,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
		Suggestions: []string{
			"Check your credentials are correct",
			"Verify API keys or certificates are valid",
			"Ensure you have proper permissions",
		},
	}
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(message string, cause error) *BerkshelfError {
	return &BerkshelfError{
		Type:    ErrorTypeConfiguration,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
		Suggestions: []string{
			"Check your configuration file syntax",
			"Verify all required settings are present",
			"Check environment variables",
		},
	}
}

// WithContext adds context to an error
func (e *BerkshelfError) WithContext(key string, value interface{}) *BerkshelfError {
	e.Context[key] = value
	return e
}

// WithSuggestion adds a suggestion to an error
func (e *BerkshelfError) WithSuggestion(suggestion string) *BerkshelfError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// ErrorCollector collects multiple errors and provides summary
type ErrorCollector struct {
	errors []error
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

// Add adds an error to the collection
func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// Error returns a combined error message
func (ec *ErrorCollector) Error() string {
	if len(ec.errors) == 0 {
		return ""
	}

	if len(ec.errors) == 1 {
		return ec.errors[0].Error()
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("Multiple errors occurred (%d total):", len(ec.errors)))

	for i, err := range ec.errors {
		parts = append(parts, fmt.Sprintf("  %d. %s", i+1, err.Error()))
	}

	return strings.Join(parts, "\n")
}

// Summary returns a summary of errors by type
func (ec *ErrorCollector) Summary() map[ErrorType]int {
	summary := make(map[ErrorType]int)

	for _, err := range ec.errors {
		if berkshelfErr, ok := err.(*BerkshelfError); ok {
			summary[berkshelfErr.Type]++
		} else {
			summary["unknown"]++
		}
	}

	return summary
}
