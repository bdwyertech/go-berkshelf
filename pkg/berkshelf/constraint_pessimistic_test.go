package berkshelf

import (
	"testing"
)

func TestPessimisticConstraintOperator(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		version    string
		shouldPass bool
	}{
		// ~> 2.0 should match >= 2.0, < 3.0
		{"~> 2.0 matches 2.0.0", "~> 2.0", "2.0.0", true},
		{"~> 2.0 matches 2.1.0", "~> 2.0", "2.1.0", true},
		{"~> 2.0 matches 2.9.9", "~> 2.0", "2.9.9", true},
		{"~> 2.0 matches 2.10.0", "~> 2.0", "2.10.0", true},
		{"~> 2.0 does not match 3.0.0", "~> 2.0", "3.0.0", false},
		{"~> 2.0 does not match 1.9.9", "~> 2.0", "1.9.9", false},

		// ~> 2.0.0 should match >= 2.0.0, < 2.1.0
		{"~> 2.0.0 matches 2.0.0", "~> 2.0.0", "2.0.0", true},
		{"~> 2.0.0 matches 2.0.1", "~> 2.0.0", "2.0.1", true},
		{"~> 2.0.0 matches 2.0.99", "~> 2.0.0", "2.0.99", true},
		{"~> 2.0.0 does not match 2.1.0", "~> 2.0.0", "2.1.0", false},
		{"~> 2.0.0 does not match 1.9.9", "~> 2.0.0", "1.9.9", false},

		// ~> 1.2.3 should match >= 1.2.3, < 1.3.0
		{"~> 1.2.3 matches 1.2.3", "~> 1.2.3", "1.2.3", true},
		{"~> 1.2.3 matches 1.2.4", "~> 1.2.3", "1.2.4", true},
		{"~> 1.2.3 matches 1.2.99", "~> 1.2.3", "1.2.99", true},
		{"~> 1.2.3 does not match 1.3.0", "~> 1.2.3", "1.3.0", false},
		{"~> 1.2.3 does not match 1.2.2", "~> 1.2.3", "1.2.2", false},

		// ~> 1 should match >= 1.0.0, < 2.0.0
		{"~> 1 matches 1.0.0", "~> 1", "1.0.0", true},
		{"~> 1 matches 1.9.9", "~> 1", "1.9.9", true},
		{"~> 1 matches 1.99.99", "~> 1", "1.99.99", true},
		{"~> 1 does not match 2.0.0", "~> 1", "2.0.0", false},
		{"~> 1 does not match 0.9.9", "~> 1", "0.9.9", false},

		// Edge cases
		{"~> 0 matches 0.0.0", "~> 0", "0.0.0", true},
		{"~> 0 matches 0.9.9", "~> 0", "0.9.9", true},
		{"~> 0 does not match 1.0.0", "~> 0", "1.0.0", false},

		{"~> 0.0 matches 0.0.0", "~> 0.0", "0.0.0", true},
		{"~> 0.0 matches 0.99.99", "~> 0.0", "0.99.99", true},
		{"~> 0.0 does not match 1.0.0", "~> 0.0", "1.0.0", false},

		{"~> 0.0.0 matches 0.0.0", "~> 0.0.0", "0.0.0", true},
		{"~> 0.0.0 matches 0.0.99", "~> 0.0.0", "0.0.99", true},
		{"~> 0.0.0 does not match 0.1.0", "~> 0.0.0", "0.1.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraint, err := NewConstraint(tt.constraint)
			if err != nil {
				t.Fatalf("Failed to create constraint %s: %v", tt.constraint, err)
			}

			version, err := NewVersion(tt.version)
			if err != nil {
				t.Fatalf("Failed to create version %s: %v", tt.version, err)
			}

			result := constraint.Check(version)
			if result != tt.shouldPass {
				t.Errorf("Constraint %s check for version %s: expected %v, got %v",
					tt.constraint, tt.version, tt.shouldPass, result)
			}
		})
	}
}

func TestPessimisticConstraintConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"~> 1", ">= 1.0.0, < 2.0.0"},
		{"~> 2.0", ">= 2.0.0, < 3.0.0"},
		{"~> 2.0.0", ">= 2.0.0, < 2.1.0"},
		{"~> 1.2.3", ">= 1.2.3, < 1.3.0"},
		{"~> 0", ">= 0.0.0, < 1.0.0"},
		{"~> 0.0", ">= 0.0.0, < 1.0.0"},
		{"~> 0.0.0", ">= 0.0.0, < 0.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertPessimisticConstraint(tt.input[3:]) // Remove "~> " prefix
			if result != tt.expected {
				t.Errorf("convertPessimisticConstraint(%s) = %s, want %s",
					tt.input, result, tt.expected)
			}
		})
	}
}
