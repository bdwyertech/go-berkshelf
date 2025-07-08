package resolver

import (
	"context"
	"fmt"
	"sort"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyertech/go-berkshelf/pkg/source"
)

// ConstraintSolver implements a more sophisticated dependency resolution algorithm
// that can handle conflicting constraints and backtracking
type ConstraintSolver struct {
	sources []source.CookbookSource
	cache   *ResolutionCache
}

// SolverState represents the current state of the resolution process
type SolverState struct {
	resolved     map[string]*berkshelf.Version
	constraints  map[string][]*berkshelf.Constraint
	dependencies map[string][]string
	queue        []string
}

// NewConstraintSolver creates a new constraint solver
func NewConstraintSolver(sources []source.CookbookSource) *ConstraintSolver {
	return &ConstraintSolver{
		sources: sources,
		cache:   NewResolutionCache(),
	}
}

// Solve attempts to find a solution that satisfies all constraints
func (cs *ConstraintSolver) Solve(ctx context.Context, requirements []*Requirement) (*Resolution, error) {
	state := &SolverState{
		resolved:     make(map[string]*berkshelf.Version),
		constraints:  make(map[string][]*berkshelf.Constraint),
		dependencies: make(map[string][]string),
		queue:        make([]string, 0),
	}

	// Initialize with requirements
	for _, req := range requirements {
		state.queue = append(state.queue, req.Name)
		if req.Constraint != nil {
			state.constraints[req.Name] = append(state.constraints[req.Name], req.Constraint)
		}
	}

	// Try to solve
	solution, err := cs.solve(ctx, state)
	if err != nil {
		return nil, err
	}

	// Build resolution from solution
	return cs.buildResolution(ctx, solution)
}

// solve performs the actual constraint solving with backtracking
func (cs *ConstraintSolver) solve(ctx context.Context, state *SolverState) (map[string]*berkshelf.Version, error) {
	// Base case: queue is empty, we have a solution
	if len(state.queue) == 0 {
		return state.resolved, nil
	}

	// Take next cookbook from queue
	cookbookName := state.queue[0]
	state.queue = state.queue[1:]

	// Skip if already resolved
	if _, exists := state.resolved[cookbookName]; exists {
		return cs.solve(ctx, state)
	}

	// Get all constraints for this cookbook
	constraints := state.constraints[cookbookName]

	// Get all available versions
	var allVersions []*berkshelf.Version
	for _, src := range cs.sources {
		versions, err := src.ListVersions(ctx, cookbookName)
		if err != nil {
			continue
		}
		allVersions = append(allVersions, versions...)
	}

	// Sort versions in descending order (newest first)
	sort.Slice(allVersions, func(i, j int) bool {
		return allVersions[i].GreaterThan(allVersions[j])
	})

	// Try each version
	for _, version := range allVersions {
		// Check if version satisfies all constraints
		if !cs.satisfiesAllConstraints(version, constraints) {
			continue
		}

		// Save current state for backtracking
		savedQueue := make([]string, len(state.queue))
		copy(savedQueue, state.queue)
		savedConstraints := cs.copyConstraints(state.constraints)

		// Try this version
		state.resolved[cookbookName] = version

		// Fetch cookbook to get dependencies
		cookbook, err := cs.fetchCookbook(ctx, cookbookName, version)
		if err != nil {
			continue
		}

		// Add dependencies to queue and constraints
		for depName, depConstraint := range cookbook.Dependencies {
			if _, exists := state.resolved[depName]; !exists {
				state.queue = append(state.queue, depName)
			}
			state.constraints[depName] = append(state.constraints[depName], depConstraint)
			state.dependencies[cookbookName] = append(state.dependencies[cookbookName], depName)
		}

		// Try to solve with this choice
		solution, err := cs.solve(ctx, state)
		if err == nil {
			return solution, nil
		}

		// Backtrack: restore state
		delete(state.resolved, cookbookName)
		state.queue = savedQueue
		state.constraints = savedConstraints
	}

	return nil, fmt.Errorf("no solution found for %s with constraints %v", cookbookName, constraints)
}

// satisfiesAllConstraints checks if a version satisfies all given constraints
func (cs *ConstraintSolver) satisfiesAllConstraints(version *berkshelf.Version, constraints []*berkshelf.Constraint) bool {
	for _, constraint := range constraints {
		if !constraint.Check(version) {
			return false
		}
	}
	return true
}

// copyConstraints creates a deep copy of the constraints map
func (cs *ConstraintSolver) copyConstraints(constraints map[string][]*berkshelf.Constraint) map[string][]*berkshelf.Constraint {
	copy := make(map[string][]*berkshelf.Constraint)
	for k, v := range constraints {
		copy[k] = make([]*berkshelf.Constraint, len(v))
		for i, c := range v {
			copy[k][i] = c
		}
	}
	return copy
}

// fetchCookbook fetches cookbook metadata from any available source
func (cs *ConstraintSolver) fetchCookbook(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Cookbook, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s@%s", name, version.String())
	if cookbook := cs.cache.GetMetadata(cacheKey); cookbook != nil {
		return cookbook, nil
	}

	// Try each source
	for _, src := range cs.sources {
		cookbook, err := src.FetchCookbook(ctx, name, version)
		if err == nil {
			cs.cache.SetMetadata(cacheKey, cookbook)
			return cookbook, nil
		}
	}

	return nil, fmt.Errorf("cookbook %s@%s not found in any source", name, version.String())
}

// buildResolution builds a Resolution object from the solution
func (cs *ConstraintSolver) buildResolution(ctx context.Context, solution map[string]*berkshelf.Version) (*Resolution, error) {
	resolution := NewResolution()

	// Fetch all cookbooks and build resolution
	for name, version := range solution {
		cookbook, err := cs.fetchCookbook(ctx, name, version)
		if err != nil {
			resolution.AddError(err)
			continue
		}

		// Add to resolution
		resolved := &ResolvedCookbook{
			Name:         name,
			Version:      version,
			Source:       nil, // TODO: track source
			Dependencies: make(map[string]*berkshelf.Version),
			Cookbook:     cookbook,
		}

		// Add resolved dependencies
		for depName, depConstraint := range cookbook.Dependencies {
			if depVersion, exists := solution[depName]; exists {
				resolved.Dependencies[depName] = depVersion
			} else {
				resolution.AddError(fmt.Errorf("dependency %s %s of %s not resolved", depName, depConstraint, name))
			}
		}

		resolution.AddCookbook(resolved)

		// Add to graph
		node := resolution.Graph.AddCookbook(cookbook)
		node.Resolved = true
	}

	// Build graph edges
	for _, cookbook := range resolution.AllCookbooks() {
		node, _ := resolution.Graph.GetCookbook(cookbook.Name)
		for depName := range cookbook.Dependencies {
			if depNode, exists := resolution.Graph.GetCookbook(depName); exists {
				resolution.Graph.AddDependency(node, depNode, nil)
			}
		}
	}

	return resolution, nil
}
