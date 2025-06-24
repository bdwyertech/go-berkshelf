package types

import (
	"testing"
)

func TestParseConstraint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Constraint
		wantErr bool
	}{
		{
			name:  "equals constraint",
			input: "= 1.2.3",
			want: &Constraint{
				Operator: ConstraintEqual,
				Version:  &Version{Major: 1, Minor: 2, Patch: 3},
			},
			wantErr: false,
		},
		{
			name:  "greater than constraint",
			input: "> 1.2.3",
			want: &Constraint{
				Operator: ConstraintGreaterThan,
				Version:  &Version{Major: 1, Minor: 2, Patch: 3},
			},
			wantErr: false,
		},
		{
			name:  "pessimistic constraint",
			input: "~> 1.2.3",
			want: &Constraint{
				Operator: ConstraintPessimistic,
				Version:  &Version{Major: 1, Minor: 2, Patch: 3},
			},
			wantErr: false,
		},
		{
			name:  "no operator (implied equals)",
			input: "1.2.3",
			want: &Constraint{
				Operator: ConstraintEqual,
				Version:  &Version{Major: 1, Minor: 2, Patch: 3},
			},
			wantErr: false,
		},
		{
			name:  "greater equal constraint",
			input: ">= 1.2.3",
			want: &Constraint{
				Operator: ConstraintGreaterEqual,
				Version:  &Version{Major: 1, Minor: 2, Patch: 3},
			},
			wantErr: false,
		},
		{
			name:    "empty constraint",
			input:   "",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid version",
			input:   "> invalid",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConstraint(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConstraint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !constraintsEqual(got, tt.want) {
				t.Errorf("ParseConstraint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConstraintSatisfies(t *testing.T) {
	tests := []struct {
		name       string
		constraint *Constraint
		version    *Version
		want       bool
	}{
		{
			name: "equals - exact match",
			constraint: &Constraint{
				Operator: ConstraintEqual,
				Version:  &Version{Major: 1, Minor: 2, Patch: 3},
			},
			version: &Version{Major: 1, Minor: 2, Patch: 3},
			want:    true,
		},
		{
			name: "equals - no match",
			constraint: &Constraint{
				Operator: ConstraintEqual,
				Version:  &Version{Major: 1, Minor: 2, Patch: 3},
			},
			version: &Version{Major: 1, Minor: 2, Patch: 4},
			want:    false,
		},
		{
			name: "greater than - satisfied",
			constraint: &Constraint{
				Operator: ConstraintGreaterThan,
				Version:  &Version{Major: 1, Minor: 2, Patch: 3},
			},
			version: &Version{Major: 1, Minor: 2, Patch: 4},
			want:    true,
		},
		{
			name: "greater than - not satisfied",
			constraint: &Constraint{
				Operator: ConstraintGreaterThan,
				Version:  &Version{Major: 1, Minor: 2, Patch: 3},
			},
			version: &Version{Major: 1, Minor: 2, Patch: 3},
			want:    false,
		},
		{
			name: "pessimistic - patch level",
			constraint: &Constraint{
				Operator: ConstraintPessimistic,
				Version:  &Version{Major: 1, Minor: 2, Patch: 3},
			},
			version: &Version{Major: 1, Minor: 2, Patch: 5},
			want:    true,
		},
		{
			name: "pessimistic - minor level blocked",
			constraint: &Constraint{
				Operator: ConstraintPessimistic,
				Version:  &Version{Major: 1, Minor: 2, Patch: 3},
			},
			version: &Version{Major: 1, Minor: 3, Patch: 0},
			want:    false,
		},
		{
			name: "pessimistic - minor level",
			constraint: &Constraint{
				Operator: ConstraintPessimistic,
				Version:  &Version{Major: 1, Minor: 2, Patch: 0},
			},
			version: &Version{Major: 1, Minor: 3, Patch: 0},
			want:    true,
		},
		{
			name: "pessimistic - major level blocked",
			constraint: &Constraint{
				Operator: ConstraintPessimistic,
				Version:  &Version{Major: 1, Minor: 2, Patch: 0},
			},
			version: &Version{Major: 2, Minor: 0, Patch: 0},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.constraint.Satisfies(tt.version)
			if got != tt.want {
				t.Errorf("Constraint.Satisfies() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseConstraintSet(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int // number of constraints
		wantErr bool
	}{
		{
			name:    "single constraint",
			input:   ">= 1.2.3",
			want:    1,
			wantErr: false,
		},
		{
			name:    "multiple constraints",
			input:   ">= 1.2.3, < 2.0.0",
			want:    2,
			wantErr: false,
		},
		{
			name:    "complex constraints",
			input:   ">= 1.2.3, < 2.0.0, != 1.5.0",
			want:    3,
			wantErr: false,
		},
		{
			name:    "invalid constraint in set",
			input:   ">= 1.2.3, invalid",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConstraintSet(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConstraintSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got.Constraints) != tt.want {
				t.Errorf("ParseConstraintSet() returned %d constraints, want %d", len(got.Constraints), tt.want)
			}
		})
	}
}

// Helper function to compare constraints
func constraintsEqual(c1, c2 *Constraint) bool {
	if c1 == nil && c2 == nil {
		return true
	}
	if c1 == nil || c2 == nil {
		return false
	}
	return c1.Operator == c2.Operator && versionsEqual(c1.Version, c2.Version)
}
