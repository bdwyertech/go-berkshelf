package berkshelf

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// Version wraps semver.Version for berkshelf-specific functionality
type Version struct {
	*semver.Version
}

// NewVersion creates a new Version from a string
func NewVersion(v string) (*Version, error) {
	// Clean up common variations in version strings
	cleaned := cleanVersionString(v)

	sv, err := semver.NewVersion(cleaned)
	if err != nil {
		return nil, fmt.Errorf("invalid version %q: %w", v, err)
	}
	return &Version{Version: sv}, nil
}

// MustVersion creates a new Version from a string and panics on error
func MustVersion(v string) *Version {
	version, err := NewVersion(v)
	if err != nil {
		panic(err)
	}
	return version
}

// String returns the string representation of the version
func (v *Version) String() string {
	if v.Version == nil {
		return ""
	}
	return v.Version.String()
}

// Equal checks if two versions are equal
func (v *Version) Equal(other *Version) bool {
	if v.Version == nil || other.Version == nil {
		return v.Version == other.Version
	}
	return v.Version.Equal(other.Version)
}

// Compare compares two versions
// Returns -1 if v < other, 0 if v == other, 1 if v > other
func (v *Version) Compare(other *Version) int {
	if v.Version == nil && other.Version == nil {
		return 0
	}
	if v.Version == nil {
		return -1
	}
	if other.Version == nil {
		return 1
	}
	return v.Version.Compare(other.Version)
}

// LessThan checks if this version is less than another
func (v *Version) LessThan(other *Version) bool {
	return v.Compare(other) < 0
}

// GreaterThan checks if this version is greater than another
func (v *Version) GreaterThan(other *Version) bool {
	return v.Compare(other) > 0
}

// cleanVersionString normalizes version strings for parsing
func cleanVersionString(v string) string {
	// Remove common prefixes like 'v'
	if len(v) > 1 && v[0] == 'v' {
		return v[1:]
	}
	return v
}
