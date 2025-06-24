package berksfile

import (
	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

// SourceType represents the type of cookbook source
type SourceType string

const (
	// SourceSupermarket represents a Chef Supermarket source
	SourceSupermarket SourceType = "supermarket"
	// SourceGit represents a Git repository source
	SourceGit SourceType = "git"
	// SourcePath represents a local filesystem path
	SourcePath SourceType = "path"
	// SourceChefServer represents a Chef Server source
	SourceChefServer SourceType = "chef_server"
)

// SourceLocation represents where a cookbook comes from
type SourceLocation struct {
	Type    SourceType
	URI     string
	Options map[string]string // branch, tag, ref, etc.
}

// CookbookDef represents a cookbook definition in a Berksfile
type CookbookDef struct {
	Name       string
	Constraint *berkshelf.Constraint
	Source     SourceLocation
	Groups     []string
}

// Berksfile represents a parsed Berksfile
type Berksfile struct {
	Sources     []string                  // List of default sources
	Cookbooks   []*CookbookDef            // All cookbook definitions
	Groups      map[string][]*CookbookDef // Grouped cookbooks
	HasMetadata bool                      // Whether metadata directive is present
}

// GetCookbooks returns all cookbooks, optionally filtered by groups
func (b *Berksfile) GetCookbooks(groups ...string) []*CookbookDef {
	if len(groups) == 0 {
		return b.Cookbooks
	}

	// Create a map to avoid duplicates
	cookbookMap := make(map[string]*CookbookDef)

	for _, group := range groups {
		if groupCookbooks, ok := b.Groups[group]; ok {
			for _, cookbook := range groupCookbooks {
				cookbookMap[cookbook.Name] = cookbook
			}
		}
	}

	// Convert map to slice
	result := make([]*CookbookDef, 0, len(cookbookMap))
	for _, cookbook := range cookbookMap {
		result = append(result, cookbook)
	}

	return result
}

// GetCookbook returns a specific cookbook by name
func (b *Berksfile) GetCookbook(name string) *CookbookDef {
	for _, cookbook := range b.Cookbooks {
		if cookbook.Name == name {
			return cookbook
		}
	}
	return nil
}

// HasGroup checks if a group exists
func (b *Berksfile) HasGroup(name string) bool {
	_, ok := b.Groups[name]
	return ok
}

// GetGroups returns all group names
func (b *Berksfile) GetGroups() []string {
	groups := make([]string, 0, len(b.Groups))
	for group := range b.Groups {
		groups = append(groups, group)
	}
	return groups
}

// ParseConstraint parses a version constraint string
func ParseConstraint(constraintStr string) (*berkshelf.Constraint, error) {
	return berkshelf.NewConstraint(constraintStr)
}
