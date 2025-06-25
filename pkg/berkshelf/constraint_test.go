package berkshelf

import (
	"testing"
)

func TestNewConstraint(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		wantErr    bool
	}{
		{
			name:       "exact version",
			constraint: "= 1.0.0",
			wantErr:    false,
		},
		{
			name:       "greater than",
			constraint: "> 1.0.0",
			wantErr:    false,
		},
		{
			name:       "greater than or equal",
			constraint: ">= 1.0.0",
			wantErr:    false,
		},
		{
			name:       "less than",
			constraint: "< 2.0.0",
			wantErr:    false,
		},
		{
			name:       "less than or equal",
			constraint: "<= 2.0.0",
			wantErr:    false,
		},
		{
			name:       "pessimistic operator - patch",
			constraint: "~> 1.2.3",
			wantErr:    false,
		},
		{
			name:       "pessimistic operator - minor",
			constraint: "~> 1.2",
			wantErr:    false,
		},
		{
			name:       "pessimistic operator - major",
			constraint: "~> 1",
			wantErr:    false,
		},
		{
			name:       "Ruby-style equality",
			constraint: "== 1.0.0",
			wantErr:    false,
		},
		{
			name:       "empty constraint",
			constraint: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewConstraint(tt.constraint)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConstraintCheck(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		version    string
		want       bool
	}{
		{
			name:       "exact match",
			constraint: "= 1.0.0",
			version:    "1.0.0",
			want:       true,
		},
		{
			name:       "exact mismatch",
			constraint: "= 1.0.0",
			version:    "1.0.1",
			want:       false,
		},
		{
			name:       "greater than - satisfied",
			constraint: "> 1.0.0",
			version:    "1.0.1",
			want:       true,
		},
		{
			name:       "greater than - not satisfied",
			constraint: "> 1.0.0",
			version:    "1.0.0",
			want:       false,
		},
		{
			name:       "greater than or equal - equal",
			constraint: ">= 1.0.0",
			version:    "1.0.0",
			want:       true,
		},
		{
			name:       "greater than or equal - greater",
			constraint: ">= 1.0.0",
			version:    "1.0.1",
			want:       true,
		},
		{
			name:       "less than - satisfied",
			constraint: "< 2.0.0",
			version:    "1.9.9",
			want:       true,
		},
		{
			name:       "less than - not satisfied",
			constraint: "< 2.0.0",
			version:    "2.0.0",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraint, err := NewConstraint(tt.constraint)
			if err != nil {
				t.Fatalf("NewConstraint() failed: %v", err)
			}

			version, err := NewVersion(tt.version)
			if err != nil {
				t.Fatalf("NewVersion() failed: %v", err)
			}

			got := constraint.Check(version)
			if got != tt.want {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPessimisticConstraint(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		version    string
		want       bool
	}{
		// ~> 1.2.3 should allow >= 1.2.3, < 1.3.0
		{
			name:       "pessimistic patch - exact match",
			constraint: "~> 1.2.3",
			version:    "1.2.3",
			want:       true,
		},
		{
			name:       "pessimistic patch - patch increment",
			constraint: "~> 1.2.3",
			version:    "1.2.4",
			want:       true,
		},
		{
			name:       "pessimistic patch - minor increment",
			constraint: "~> 1.2.3",
			version:    "1.3.0",
			want:       false,
		},
		{
			name:       "pessimistic patch - major increment",
			constraint: "~> 1.2.3",
			version:    "2.0.0",
			want:       false,
		},
		{
			name:       "pessimistic patch - below minimum",
			constraint: "~> 1.2.3",
			version:    "1.2.2",
			want:       false,
		},
		// ~> 1.2 should allow >= 1.2.0, < 2.0.0
		{
			name:       "pessimistic minor - exact match",
			constraint: "~> 1.2",
			version:    "1.2.0",
			want:       true,
		},
		{
			name:       "pessimistic minor - patch increment",
			constraint: "~> 1.2",
			version:    "1.2.5",
			want:       true,
		},
		{
			name:       "pessimistic minor - minor increment",
			constraint: "~> 1.2",
			version:    "1.3.0",
			want:       true, // Changed from false - ~> 1.2 allows 1.x where x >= 2
		},
		{
			name:       "pessimistic minor - major increment",
			constraint: "~> 1.2",
			version:    "2.0.0",
			want:       false,
		},
		// ~> 1 should allow >= 1.0.0, < 2.0.0
		{
			name:       "pessimistic major - exact match",
			constraint: "~> 1",
			version:    "1.0.0",
			want:       true,
		},
		{
			name:       "pessimistic major - minor increment",
			constraint: "~> 1",
			version:    "1.5.0",
			want:       true,
		},
		{
			name:       "pessimistic major - major increment",
			constraint: "~> 1",
			version:    "2.0.0",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraint, err := NewConstraint(tt.constraint)
			if err != nil {
				t.Fatalf("NewConstraint() failed: %v", err)
			}

			version, err := NewVersion(tt.version)
			if err != nil {
				t.Fatalf("NewVersion() failed: %v", err)
			}

			got := constraint.Check(version)
			if got != tt.want {
				t.Errorf("Check() = %v, want %v for constraint %s and version %s", got, tt.want, tt.constraint, tt.version)
			}
		})
	}
}

func TestConstraintConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "pessimistic patch level",
			input:    "~> 1.2.3",
			expected: ">= 1.2.3, < 1.3.0",
		},
		{
			name:     "pessimistic minor level",
			input:    "~> 1.2",
			expected: ">= 1.2.0, < 2.0.0",
		},
		{
			name:     "pessimistic major level",
			input:    "~> 1",
			expected: ">= 1.0.0, < 2.0.0",
		},
		{
			name:     "Ruby equality operator",
			input:    "== 1.0.0",
			expected: "= 1.0.0",
		},
		{
			name:     "standard constraint unchanged",
			input:    ">= 1.0.0",
			expected: ">= 1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converted := convertRubyConstraint(tt.input)
			if converted != tt.expected {
				t.Errorf("convertRubyConstraint() = %v, want %v", converted, tt.expected)
			}
		})
	}
}

func TestMustConstraintPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustConstraint should panic on invalid constraint")
		}
	}()

	// Use an actually invalid constraint that will cause panic
	MustConstraint("invalid constraint syntax !!!")
}

func TestConstraintString(t *testing.T) {
	constraint := MustConstraint("~> 1.2.3")
	if constraint.String() != "~> 1.2.3" {
		t.Errorf("String() = %v, want %v", constraint.String(), "~> 1.2.3")
	}
}
