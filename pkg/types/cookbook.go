package types

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Cookbook represents a Chef cookbook
type Cookbook struct {
	Name         string
	Version      *Version
	Path         string
	Metadata     *Metadata
	Dependencies map[string]*ConstraintSet
}

// String returns the string representation of the cookbook
func (c Cookbook) String() string {
	return fmt.Sprintf("%s (%s)", c.Name, c.Version)
}

// Metadata represents cookbook metadata
type Metadata struct {
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description"`
	LongDescription string                 `json:"long_description"`
	Maintainer      string                 `json:"maintainer"`
	MaintainerEmail string                 `json:"maintainer_email"`
	License         string                 `json:"license"`
	Platforms       map[string]string      `json:"platforms"`
	Dependencies    map[string]string      `json:"dependencies"`
	Provides        map[string]string      `json:"provides"`
	Recipes         map[string]string      `json:"recipes"`
	Attributes      map[string]interface{} `json:"attributes"`
	ChefVersions    map[string]string      `json:"chef_versions"`
	OhaiVersions    map[string]string      `json:"ohai_versions"`
	Gems            map[string]string      `json:"gems"`
	Issues          string                 `json:"issues_url"`
	Source          string                 `json:"source_url"`
}

// ParseDependencies converts metadata dependencies to ConstraintSets
func (m *Metadata) ParseDependencies() (map[string]*ConstraintSet, error) {
	deps := make(map[string]*ConstraintSet)

	for name, constraint := range m.Dependencies {
		if constraint == "" {
			// No constraint means any version
			deps[name] = &ConstraintSet{
				Constraints: []*Constraint{
					{
						Operator: ConstraintGreaterEqual,
						Version:  &Version{Major: 0, Minor: 0, Patch: 0},
					},
				},
			}
		} else {
			cs, err := ParseConstraintSet(constraint)
			if err != nil {
				return nil, fmt.Errorf("invalid constraint for %s: %w", name, err)
			}
			deps[name] = cs
		}
	}

	return deps, nil
}

// CookbookRef represents a reference to a cookbook with constraints
type CookbookRef struct {
	Name       string
	Constraint *ConstraintSet
	Source     SourceLocation
	Groups     []string
}

// String returns the string representation of the cookbook reference
func (cr CookbookRef) String() string {
	if cr.Constraint != nil {
		return fmt.Sprintf("%s (%s)", cr.Name, cr.Constraint)
	}
	return cr.Name
}

// SourceLocation represents where a cookbook comes from
type SourceLocation struct {
	Type SourceType
	URL  string
	Path string
	Ref  string // Git ref (branch, tag, commit)
}

// SourceType represents the type of cookbook source
type SourceType string

const (
	// SourceTypeSupermarket represents Chef Supermarket
	SourceTypeSupermarket SourceType = "supermarket"
	// SourceTypeGit represents a Git repository
	SourceTypeGit SourceType = "git"
	// SourceTypePath represents a local filesystem path
	SourceTypePath SourceType = "path"
	// SourceTypeChefServer represents a Chef Server
	SourceTypeChefServer SourceType = "chef_server"
)

// LoadMetadataFromFile loads metadata from a JSON file
func LoadMetadataFromFile(path string) (*Metadata, error) {
	// For now, we only support metadata.json
	// TODO: Add support for metadata.rb parsing

	metadataPath := filepath.Join(path, "metadata.json")
	var metadata Metadata

	// Read the file
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata.json: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata.json: %w", err)
	}

	return &metadata, nil
}

// Resolution represents a resolved set of cookbooks
type Resolution struct {
	Cookbooks map[string]*ResolvedCookbook
}

// ResolvedCookbook represents a cookbook with a specific version and source
type ResolvedCookbook struct {
	Name         string
	Version      *Version
	Source       SourceLocation
	Dependencies map[string]*Version
}

// String returns the string representation of the resolved cookbook
func (rc ResolvedCookbook) String() string {
	return fmt.Sprintf("%s (%s) from %s", rc.Name, rc.Version, rc.Source.Type)
}
