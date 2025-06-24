package source

import (
	"errors"
	"fmt"
)

// Common errors
var (
	// ErrNotImplemented is returned when a source doesn't implement an optional method.
	ErrNotImplemented = errors.New("method not implemented")

	// ErrInvalidSource is returned when a source configuration is invalid.
	ErrInvalidSource = errors.New("invalid source configuration")

	// ErrAuthenticationRequired is returned when authentication is needed but not provided.
	ErrAuthenticationRequired = errors.New("authentication required")
)

// ErrCookbookNotFound is returned when a cookbook cannot be found.
type ErrCookbookNotFound struct {
	Name    string
	Version string
}

func (e *ErrCookbookNotFound) Error() string {
	if e.Version != "" {
		return fmt.Sprintf("cookbook %s version %s not found", e.Name, e.Version)
	}
	return fmt.Sprintf("cookbook %s not found", e.Name)
}

// ErrVersionNotFound is returned when a specific version cannot be found.
type ErrVersionNotFound struct {
	Name    string
	Version string
}

func (e *ErrVersionNotFound) Error() string {
	return fmt.Sprintf("version %s of cookbook %s not found", e.Version, e.Name)
}

// ErrInvalidMetadata is returned when cookbook metadata is invalid or corrupt.
type ErrInvalidMetadata struct {
	Name   string
	Reason string
}

func (e *ErrInvalidMetadata) Error() string {
	return fmt.Sprintf("invalid metadata for cookbook %s: %s", e.Name, e.Reason)
}

// ErrSourceUnavailable is returned when a source is temporarily unavailable.
type ErrSourceUnavailable struct {
	Source string
	Reason string
}

func (e *ErrSourceUnavailable) Error() string {
	return fmt.Sprintf("source %s unavailable: %s", e.Source, e.Reason)
}
