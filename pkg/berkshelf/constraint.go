package berkshelf

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Constraint represents a version constraint
type Constraint struct {
	raw        string
	constraint *semver.Constraints
}

// NewConstraint creates a constraint from a string
func NewConstraint(c string) (*Constraint, error) {
	// If no constraint is provided, use "any version" (>= 0.0.0)
	if c == "" {
		constraint, err := semver.NewConstraint(">= 0.0.0")
		if err != nil {
			return nil, fmt.Errorf("failed to create default constraint: %w", err)
		}
		return &Constraint{
			raw:        "",
			constraint: constraint,
		}, nil
	}

	// Convert Ruby-style constraints to semver format
	converted := convertRubyConstraint(c)

	constraint, err := semver.NewConstraint(converted)
	if err != nil {
		return nil, fmt.Errorf("invalid constraint %q: %w", c, err)
	}

	return &Constraint{
		raw:        c,
		constraint: constraint,
	}, nil
}

// MustConstraint creates a constraint and panics on error
func MustConstraint(c string) *Constraint {
	constraint, err := NewConstraint(c)
	if err != nil {
		panic(err)
	}
	return constraint
}

// Check verifies if a version satisfies the constraint
func (c *Constraint) Check(v *Version) bool {
	if c.constraint == nil || v.Version == nil {
		return false
	}
	return c.constraint.Check(v.Version)
}

// String returns the original constraint string
func (c *Constraint) String() string {
	if c.raw == "" {
		return ">= 0.0.0"
	}
	return c.raw
}

// pessimisticRegex matches Ruby's pessimistic version operator (~>)
var pessimisticRegex = regexp.MustCompile(`^~>\s*(\d+(?:\.\d+)*)$`)

// convertRubyConstraint converts Ruby-style constraints to semver format
func convertRubyConstraint(c string) string {
	// Trim whitespace
	c = strings.TrimSpace(c)

	// Handle pessimistic operator ~> (Ruby-style)
	if match := pessimisticRegex.FindStringSubmatch(c); match != nil {
		version := match[1]
		return convertPessimisticConstraint(version)
	}

	// Handle other Ruby-style operators that might need conversion
	c = strings.ReplaceAll(c, "==", "=")

	return c
}

// convertPessimisticConstraint converts Ruby's pessimistic constraint operator
// ~> 2.0 becomes >= 2.0, < 3.0 (allows 2.x.y)
// ~> 2.0.0 becomes >= 2.0.0, < 2.1.0 (allows 2.0.x only)
func convertPessimisticConstraint(version string) string {
	parts := strings.Split(version, ".")

	if len(parts) == 1 {
		// ~> 1 becomes >= 1.0.0, < 2.0.0
		major := mustParseInt(parts[0])
		return fmt.Sprintf(">= %s.0.0, < %d.0.0", version, major+1)
	}

	if len(parts) == 2 {
		// ~> 1.2 becomes >= 1.2.0, < 2.0.0 (allows any 1.x where x >= 2)
		major := mustParseInt(parts[0])
		return fmt.Sprintf(">= %s.0, < %d.0.0", version, major+1)
	}

	if len(parts) >= 3 {
		// ~> 1.2.3 becomes >= 1.2.3, < 1.3.0 (allows 1.2.x where x >= 3)
		major := mustParseInt(parts[0])
		minor := mustParseInt(parts[1])
		return fmt.Sprintf(">= %s, < %d.%d.0", version, major, minor+1)
	}

	return version
}

// mustParseInt parses an integer and panics on error (for internal use)
func mustParseInt(s string) int {
	var result int
	for _, r := range s {
		if r < '0' || r > '9' {
			panic(fmt.Sprintf("invalid integer: %s", s))
		}
		result = result*10 + int(r-'0')
	}
	return result
}
