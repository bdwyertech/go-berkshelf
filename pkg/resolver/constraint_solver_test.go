package resolver

import (
	"context"
	"testing"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

func TestConstraintSolverConflictingConstraints(t *testing.T) {
	// Create mock source
	mockSrc := newMockSource("test", 100)

	// Add cookbooks with conflicting dependencies
	mockSrc.addCookbook("app", "1.0.0", map[string]string{
		"database": "~> 2.0",
	})
	mockSrc.addCookbook("api", "1.0.0", map[string]string{
		"database": "~> 1.0",
	})
	mockSrc.addCookbook("database", "1.0.0", map[string]string{})
	mockSrc.addCookbook("database", "1.5.0", map[string]string{})
	mockSrc.addCookbook("database", "2.0.0", map[string]string{})

	// Create constraint solver
	solver := NewConstraintSolver(createSources(mockSrc))

	// Create requirements that will conflict
	appConstraint, _ := berkshelf.NewConstraint("= 1.0.0")
	apiConstraint, _ := berkshelf.NewConstraint("= 1.0.0")
	requirements := []*Requirement{
		NewRequirement("app", appConstraint),
		NewRequirement("api", apiConstraint),
	}

	// Solve
	ctx := context.Background()
	_, err := solver.Solve(ctx, requirements)

	// Should fail due to conflicting constraints
	if err == nil {
		t.Error("Expected solver to fail due to conflicting constraints")
	} else {
		t.Logf("Solver correctly failed with: %v", err)
	}
}

func TestConstraintSolverBacktracking(t *testing.T) {
	// Create mock source
	mockSrc := newMockSource("test", 100)

	// Create a scenario where the solver needs to backtrack
	// App requires database ~> 2.0 and cache ~> 1.0
	// Cache 2.0 requires database ~> 3.0 (conflicts with app)
	// Cache 1.0 requires database >= 2.0 (compatible with app)

	mockSrc.addCookbook("app", "1.0.0", map[string]string{
		"database": "~> 2.0",
		"cache":    "~> 1.0",
	})

	// Cache 2.0 would be tried first (higher version) but conflicts
	mockSrc.addCookbook("cache", "2.0.0", map[string]string{
		"database": "~> 3.0",
	})

	// Cache 1.0 is compatible
	mockSrc.addCookbook("cache", "1.0.0", map[string]string{
		"database": ">= 2.0",
	})

	mockSrc.addCookbook("database", "2.0.0", map[string]string{})
	mockSrc.addCookbook("database", "3.0.0", map[string]string{})

	// Create constraint solver
	solver := NewConstraintSolver(createSources(mockSrc))

	// Create requirements
	appConstraint, _ := berkshelf.NewConstraint("= 1.0.0")
	requirements := []*Requirement{
		NewRequirement("app", appConstraint),
	}

	// Solve
	ctx := context.Background()
	resolution, err := solver.Solve(ctx, requirements)
	if err != nil {
		t.Fatalf("Solver failed: %v", err)
	}

	// Check solution
	if !resolution.HasCookbook("app") {
		t.Error("Expected app to be resolved")
	}

	if !resolution.HasCookbook("cache") {
		t.Error("Expected cache to be resolved")
	} else {
		cache, _ := resolution.GetCookbook("cache")
		if cache.Version.String() != "1.0.0" {
			t.Errorf("Expected cache 1.0.0, got %s", cache.Version.String())
		} else {
			t.Log("Solver correctly backtracked to cache 1.0.0")
		}
	}

	if !resolution.HasCookbook("database") {
		t.Error("Expected database to be resolved")
	} else {
		db, _ := resolution.GetCookbook("database")
		if db.Version.String() != "2.0.0" {
			t.Errorf("Expected database 2.0.0, got %s", db.Version.String())
		}
	}
}

func TestConstraintSolverComplexDependencies(t *testing.T) {
	// Create mock source
	mockSrc := newMockSource("test", 100)

	// Create a complex dependency graph
	mockSrc.addCookbook("webapp", "1.0.0", map[string]string{
		"framework": "~> 2.0",
		"database":  ">= 1.0",
	})

	mockSrc.addCookbook("framework", "2.0.0", map[string]string{
		"logger":   "~> 1.0",
		"database": "~> 1.5",
	})

	mockSrc.addCookbook("framework", "2.5.0", map[string]string{
		"logger":   "~> 2.0",
		"database": "~> 2.0",
	})

	mockSrc.addCookbook("logger", "1.0.0", map[string]string{})
	mockSrc.addCookbook("logger", "2.0.0", map[string]string{})

	mockSrc.addCookbook("database", "1.0.0", map[string]string{})
	mockSrc.addCookbook("database", "1.5.0", map[string]string{})
	mockSrc.addCookbook("database", "2.0.0", map[string]string{})

	// Create constraint solver
	solver := NewConstraintSolver(createSources(mockSrc))

	// Create requirements
	webappConstraint, _ := berkshelf.NewConstraint("= 1.0.0")
	requirements := []*Requirement{
		NewRequirement("webapp", webappConstraint),
	}

	// Solve
	ctx := context.Background()
	resolution, err := solver.Solve(ctx, requirements)
	if err != nil {
		t.Fatalf("Solver failed: %v", err)
	}

	// Check solution
	expectedCookbooks := map[string]string{
		"webapp":    "1.0.0",
		"framework": "2.0.0", // Should pick 2.0.0 because 2.5.0 requires database ~> 2.0 which conflicts
		"logger":    "1.0.0",
		"database":  "1.5.0", // Satisfies both >= 1.0 and ~> 1.5
	}

	for name, expectedVersion := range expectedCookbooks {
		if !resolution.HasCookbook(name) {
			t.Errorf("Expected %s to be resolved", name)
		} else {
			cb, _ := resolution.GetCookbook(name)
			if cb.Version.String() != expectedVersion {
				t.Errorf("Expected %s version %s, got %s", name, expectedVersion, cb.Version.String())
			}
		}
	}
}
