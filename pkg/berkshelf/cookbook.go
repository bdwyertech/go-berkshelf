package berkshelf

import (
	"fmt"
	"path/filepath"
)

// Cookbook represents a Chef cookbook with its metadata
type Cookbook struct {
	Name         string                 `json:"name"`
	Version      *Version               `json:"version"`
	Dependencies map[string]*Constraint `json:"dependencies,omitempty"`
	Metadata     *Metadata              `json:"metadata,omitempty"`
	Source       SourceLocation         `json:"source,omitempty"`
	Path         string                 `json:"path,omitempty"`
	TarballURL   string                 `json:"tarball_url,omitempty"`
}

// Metadata represents cookbook metadata from metadata.rb or metadata.json
type Metadata struct {
	Name            string                 `json:"name"`
	Version         *Version               `json:"version"`
	Description     string                 `json:"description,omitempty"`
	LongDescription string                 `json:"long_description,omitempty"`
	Maintainer      string                 `json:"maintainer,omitempty"`
	MaintainerEmail string                 `json:"maintainer_email,omitempty"`
	License         string                 `json:"license,omitempty"`
	Platforms       map[string]*Constraint `json:"platforms,omitempty"`
	Dependencies    map[string]*Constraint `json:"dependencies,omitempty"`
	Recommendations map[string]*Constraint `json:"recommendations,omitempty"`
	Suggestions     map[string]*Constraint `json:"suggestions,omitempty"`
	Conflicts       map[string]*Constraint `json:"conflicts,omitempty"`
	Provides        map[string]*Constraint `json:"provides,omitempty"`
	Replaces        map[string]*Constraint `json:"replaces,omitempty"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	Recipes         map[string]string      `json:"recipes,omitempty"`
	Issues          string                 `json:"issues_url,omitempty"`
	Source          string                 `json:"source_url,omitempty"`
	ChefVersion     *Constraint            `json:"chef_version,omitempty"`
	OhaiVersion     *Constraint            `json:"ohai_version,omitempty"`
}

// NewCookbook creates a new cookbook instance
func NewCookbook(name string, version *Version) *Cookbook {
	return &Cookbook{
		Name:         name,
		Version:      version,
		Dependencies: make(map[string]*Constraint),
	}
}

// String returns a string representation of the cookbook
func (c *Cookbook) String() string {
	if c.Version != nil {
		return fmt.Sprintf("%s (%s)", c.Name, c.Version.String())
	}
	return c.Name
}

// AddDependency adds a dependency constraint to the cookbook
func (c *Cookbook) AddDependency(name string, constraint *Constraint) {
	if c.Dependencies == nil {
		c.Dependencies = make(map[string]*Constraint)
	}
	c.Dependencies[name] = constraint
}

// HasDependency checks if the cookbook depends on another cookbook
func (c *Cookbook) HasDependency(name string) bool {
	_, exists := c.Dependencies[name]
	return exists
}

// GetDependency returns the constraint for a dependency
func (c *Cookbook) GetDependency(name string) (*Constraint, bool) {
	constraint, exists := c.Dependencies[name]
	return constraint, exists
}

// IsLocal returns true if the cookbook is from a local path
func (c *Cookbook) IsLocal() bool {
	return c.Source.Type == "path" || c.Path != ""
}

// IsGit returns true if the cookbook is from a git repository
func (c *Cookbook) IsGit() bool {
	return c.Source.Type == "git"
}

// IsSupermarket returns true if the cookbook is from a supermarket
func (c *Cookbook) IsSupermarket() bool {
	return c.Source.Type == "supermarket" || c.Source.Type == ""
}

// BaseName returns the cookbook name without any path components
func (c *Cookbook) BaseName() string {
	return filepath.Base(c.Name)
}

// Validate performs basic validation on the cookbook
func (c *Cookbook) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("cookbook name cannot be empty")
	}

	if c.Version == nil {
		return fmt.Errorf("cookbook %s must have a version", c.Name)
	}

	// Validate dependencies
	for depName, constraint := range c.Dependencies {
		if depName == "" {
			return fmt.Errorf("cookbook %s has dependency with empty name", c.Name)
		}
		if constraint == nil {
			return fmt.Errorf("cookbook %s dependency %s has nil constraint", c.Name, depName)
		}
	}

	return nil
}
