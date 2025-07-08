package resolver

import (
	"context"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyertech/go-berkshelf/pkg/source"
)

// Resolver defines the interface for dependency resolution
type Resolver interface {
	Resolve(ctx context.Context, requirements []*Requirement) (*Resolution, error)
}

// Requirement represents a cookbook requirement to be resolved
type Requirement struct {
	Name       string
	Constraint *berkshelf.Constraint
	Source     *berkshelf.SourceLocation
}

// Resolution represents a resolved dependency graph
type Resolution struct {
	Graph     *DependencyGraph
	Cookbooks map[string]*ResolvedCookbook
	Errors    []error
}

// ResolvedCookbook represents a cookbook that has been resolved
type ResolvedCookbook struct {
	Name         string
	Version      *berkshelf.Version
	Source       *berkshelf.SourceLocation
	SourceRef    source.CookbookSource // Reference to the actual source object
	Dependencies map[string]*berkshelf.Version
	Cookbook     *berkshelf.Cookbook
}

// NewRequirement creates a new requirement
func NewRequirement(name string, constraint *berkshelf.Constraint) *Requirement {
	return &Requirement{
		Name:       name,
		Constraint: constraint,
	}
}

// NewRequirementWithSource creates a new requirement with a specific source
func NewRequirementWithSource(name string, constraint *berkshelf.Constraint, source *berkshelf.SourceLocation) *Requirement {
	return &Requirement{
		Name:       name,
		Constraint: constraint,
		Source:     source,
	}
}

// String returns a string representation of the requirement
func (r *Requirement) String() string {
	if r.Constraint != nil {
		return r.Name + " " + r.Constraint.String()
	}
	return r.Name
}

// NewResolution creates a new resolution
func NewResolution() *Resolution {
	return &Resolution{
		Graph:     NewDependencyGraph(),
		Cookbooks: make(map[string]*ResolvedCookbook),
		Errors:    make([]error, 0),
	}
}

// AddCookbook adds a resolved cookbook to the resolution
func (r *Resolution) AddCookbook(cookbook *ResolvedCookbook) {
	r.Cookbooks[cookbook.Name] = cookbook
}

// GetCookbook retrieves a resolved cookbook by name
func (r *Resolution) GetCookbook(name string) (*ResolvedCookbook, bool) {
	cookbook, exists := r.Cookbooks[name]
	return cookbook, exists
}

// HasCookbook checks if a cookbook is in the resolution
func (r *Resolution) HasCookbook(name string) bool {
	_, exists := r.Cookbooks[name]
	return exists
}

// AddError adds an error to the resolution
func (r *Resolution) AddError(err error) {
	r.Errors = append(r.Errors, err)
}

// HasErrors returns true if the resolution has any errors
func (r *Resolution) HasErrors() bool {
	return len(r.Errors) > 0
}

// CookbookCount returns the number of resolved cookbooks
func (r *Resolution) CookbookCount() int {
	return len(r.Cookbooks)
}

// AllCookbooks returns all resolved cookbooks
func (r *Resolution) AllCookbooks() []*ResolvedCookbook {
	cookbooks := make([]*ResolvedCookbook, 0, len(r.Cookbooks))
	for _, cookbook := range r.Cookbooks {
		cookbooks = append(cookbooks, cookbook)
	}
	return cookbooks
}
