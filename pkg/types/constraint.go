package types

import (
	"fmt"
	"strings"
)

// ConstraintOperator represents a version constraint operator
type ConstraintOperator string

const (
	// ConstraintEqual requires exact version match
	ConstraintEqual ConstraintOperator = "="
	// ConstraintNotEqual requires version to not match
	ConstraintNotEqual ConstraintOperator = "!="
	// ConstraintGreaterThan requires version to be greater
	ConstraintGreaterThan ConstraintOperator = ">"
	// ConstraintLessThan requires version to be less
	ConstraintLessThan ConstraintOperator = "<"
	// ConstraintGreaterEqual requires version to be greater or equal
	ConstraintGreaterEqual ConstraintOperator = ">="
	// ConstraintLessEqual requires version to be less or equal
	ConstraintLessEqual ConstraintOperator = "<="
	// ConstraintPessimistic allows changes to the last digit (~>)
	ConstraintPessimistic ConstraintOperator = "~>"
)

// Constraint represents a version constraint
type Constraint struct {
	Operator ConstraintOperator
	Version  *Version
}

// String returns the string representation of the constraint
func (c Constraint) String() string {
	return fmt.Sprintf("%s %s", c.Operator, c.Version)
}

// Satisfies checks if a version satisfies this constraint
func (c Constraint) Satisfies(v *Version) bool {
	if c.Version == nil || v == nil {
		return false
	}

	cmp := v.Compare(*c.Version)

	switch c.Operator {
	case ConstraintEqual:
		return cmp == 0
	case ConstraintNotEqual:
		return cmp != 0
	case ConstraintGreaterThan:
		return cmp > 0
	case ConstraintLessThan:
		return cmp < 0
	case ConstraintGreaterEqual:
		return cmp >= 0
	case ConstraintLessEqual:
		return cmp <= 0
	case ConstraintPessimistic:
		// ~> 2.2 means >= 2.2.0 AND < 3.0.0
		// ~> 2.2.0 means >= 2.2.0 AND < 2.3.0
		if cmp < 0 {
			return false // v is less than constraint version
		}

		// Check upper bound
		if c.Version.Patch > 0 {
			// ~> 2.2.3 means >= 2.2.3 AND < 2.3.0
			upperBound := Version{
				Major: c.Version.Major,
				Minor: c.Version.Minor + 1,
				Patch: 0,
			}
			return v.Compare(upperBound) < 0
		} else {
			// ~> 2.2 means >= 2.2.0 AND < 3.0.0
			upperBound := Version{
				Major: c.Version.Major + 1,
				Minor: 0,
				Patch: 0,
			}
			return v.Compare(upperBound) < 0
		}
	default:
		return false
	}
}

// ParseConstraint parses a constraint string into a Constraint
func ParseConstraint(s string) (*Constraint, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty constraint")
	}

	// Find the operator
	var operator ConstraintOperator
	var versionStr string

	// Check for two-character operators first
	for _, op := range []ConstraintOperator{ConstraintPessimistic, ConstraintGreaterEqual, ConstraintLessEqual, ConstraintNotEqual} {
		opStr := string(op)
		if strings.HasPrefix(s, opStr+" ") {
			operator = op
			versionStr = strings.TrimSpace(s[len(opStr):])
			break
		}
	}

	// If no two-character operator found, check single-character operators
	if operator == "" {
		for _, op := range []ConstraintOperator{ConstraintEqual, ConstraintGreaterThan, ConstraintLessThan} {
			opStr := string(op)
			if strings.HasPrefix(s, opStr+" ") {
				operator = op
				versionStr = strings.TrimSpace(s[len(opStr):])
				break
			}
		}
	}

	// If no operator found, assume equals
	if operator == "" {
		operator = ConstraintEqual
		versionStr = s
	}

	// Parse the version
	version, err := ParseVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version in constraint: %w", err)
	}

	return &Constraint{
		Operator: operator,
		Version:  version,
	}, nil
}

// ConstraintSet represents a set of constraints that must all be satisfied
type ConstraintSet struct {
	Constraints []*Constraint
}

// String returns the string representation of the constraint set
func (cs ConstraintSet) String() string {
	parts := make([]string, len(cs.Constraints))
	for i, c := range cs.Constraints {
		parts[i] = c.String()
	}
	return strings.Join(parts, ", ")
}

// Satisfies checks if a version satisfies all constraints in the set
func (cs ConstraintSet) Satisfies(v *Version) bool {
	for _, c := range cs.Constraints {
		if !c.Satisfies(v) {
			return false
		}
	}
	return true
}

// ParseConstraintSet parses a comma-separated list of constraints
func ParseConstraintSet(s string) (*ConstraintSet, error) {
	parts := strings.Split(s, ",")
	constraints := make([]*Constraint, 0, len(parts))

	for _, part := range parts {
		c, err := ParseConstraint(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, c)
	}

	return &ConstraintSet{
		Constraints: constraints,
	}, nil
}
