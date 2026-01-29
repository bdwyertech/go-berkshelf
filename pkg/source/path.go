package source

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
)

// PathSource implements CookbookSource for local filesystem paths.
type PathSource struct {
	basePath string
	priority int
}

// NewPathSource creates a new path-based cookbook source.
func NewPathSource(path string) (*PathSource, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("path does not exist: %s", absPath)
	}

	return &PathSource{
		basePath: absPath,
		priority: 200, // Highest priority for local paths
	}, nil
}

// Name returns the name of this source.
func (p *PathSource) Name() string {
	return fmt.Sprintf("path (%s)", p.basePath)
}

// Priority returns the priority of this source.
func (p *PathSource) Priority() int {
	return p.priority
}

// findCookbookPath looks for a cookbook in the path source.
func (p *PathSource) findCookbookPath(name string) (string, error) {
	// First check if the base path itself is the cookbook
	if p.isCookbook(p.basePath) {
		// Check if the cookbook name matches
		metadata, err := p.ReadMetadata(p.basePath)
		if err == nil && metadata.Name == name {
			return p.basePath, nil
		}
	}

	// Check subdirectories
	entries, err := os.ReadDir(p.basePath)
	if err != nil {
		return "", fmt.Errorf("reading directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		cookbookPath := filepath.Join(p.basePath, entry.Name())
		if p.isCookbook(cookbookPath) {
			// Check if this is the cookbook we're looking for
			metadata, err := p.ReadMetadata(cookbookPath)
			if err == nil && metadata.Name == name {
				return cookbookPath, nil
			}

			// Also check if directory name matches
			if entry.Name() == name {
				return cookbookPath, nil
			}
		}
	}

	return "", &ErrCookbookNotFound{Name: name}
}

// isCookbook checks if a directory contains a cookbook.
func (p *PathSource) isCookbook(path string) bool {
	// Check for metadata.json or metadata.rb
	metadataJSON := filepath.Join(path, "metadata.json")
	metadataRB := filepath.Join(path, "metadata.rb")

	if _, err := os.Stat(metadataJSON); err == nil {
		return true
	}
	if _, err := os.Stat(metadataRB); err == nil {
		return true
	}

	return false
}

// ReadMetadata reads cookbook metadata from a directory.
func (p *PathSource) ReadMetadata(cookbookPath string) (*berkshelf.Metadata, error) {
	// Try metadata.json first
	metadataPath := filepath.Join(cookbookPath, "metadata.json")
	if _, err := os.Stat(metadataPath); err == nil {
		return p.ReadMetadataJSON(metadataPath)
	}

	// Try metadata.rb
	metadataPath = filepath.Join(cookbookPath, "metadata.rb")
	if _, err := os.Stat(metadataPath); err == nil {
		return p.ReadMetadataRB(metadataPath, cookbookPath)
	}

	return nil, &ErrInvalidMetadata{
		Name:   filepath.Base(cookbookPath),
		Reason: "no metadata.json or metadata.rb found",
	}
}

// metadataJSON represents the structure of metadata.json
type metadataJSON struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Maintainer   string                 `json:"maintainer"`
	License      string                 `json:"license"`
	Dependencies map[string]interface{} `json:"dependencies"`
}

// ReadMetadataJSON parses a metadata.json file.
func (p *PathSource) ReadMetadataJSON(path string) (*berkshelf.Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading metadata.json: %w", err)
	}

	var meta metadataJSON
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, &ErrInvalidMetadata{
			Name:   filepath.Base(filepath.Dir(path)),
			Reason: fmt.Sprintf("invalid JSON: %v", err),
		}
	}

	// Parse version
	version, err := berkshelf.NewVersion(meta.Version)
	if err != nil {
		return nil, &ErrInvalidMetadata{
			Name:   meta.Name,
			Reason: fmt.Sprintf("invalid version: %v", err),
		}
	}

	// Parse dependencies
	dependencies := make(map[string]*berkshelf.Constraint)
	for name, value := range meta.Dependencies {
		constraintStr := ""
		switch v := value.(type) {
		case string:
			constraintStr = v
		case map[string]interface{}:
			// Some metadata formats use objects for dependencies
			if version, ok := v["version"].(string); ok {
				constraintStr = version
			}
		}

		if constraintStr != "" {
			constraint, err := berkshelf.NewConstraint(constraintStr)
			if err == nil {
				dependencies[name] = constraint
			}
		}
	}

	return &berkshelf.Metadata{
		Name:         meta.Name,
		Version:      version,
		Description:  meta.Description,
		Maintainer:   meta.Maintainer,
		License:      meta.License,
		Dependencies: dependencies,
	}, nil
}

// ReadMetadataRB parses a metadata.rb file (simplified).
func (p *PathSource) ReadMetadataRB(path string, cookbookPath string) (*berkshelf.Metadata, error) {
	// For now, we'll do a very simple parsing of metadata.rb
	// In a full implementation, we would need a Ruby parser

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading metadata.rb: %w", err)
	}

	content := string(data)
	metadata := &berkshelf.Metadata{
		Dependencies: make(map[string]*berkshelf.Constraint),
	}

	// Extract name
	if matches := extractRubyString(content, "name"); len(matches) > 0 {
		metadata.Name = matches[0]
	} else {
		// Use directory name as fallback
		metadata.Name = filepath.Base(cookbookPath)
	}

	// Extract version
	if matches := extractRubyString(content, "version"); len(matches) > 0 {
		if v, err := berkshelf.NewVersion(matches[0]); err == nil {
			metadata.Version = v
		}
	}
	if metadata.Version == nil {
		// Default version
		metadata.Version, _ = berkshelf.NewVersion("0.0.0")
	}

	// Extract description
	if matches := extractRubyString(content, "description"); len(matches) > 0 {
		metadata.Description = matches[0]
	}

	// Extract maintainer
	if matches := extractRubyString(content, "maintainer"); len(matches) > 0 {
		metadata.Maintainer = matches[0]
	}

	// Extract license
	if matches := extractRubyString(content, "license"); len(matches) > 0 {
		metadata.License = matches[0]
	}

	// Extract dependencies (simplified)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "depends") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := strings.Trim(parts[1], `"',`)
				constraintStr := ">= 0.0.0"
				if len(parts) >= 3 {
					constraintStr = strings.Trim(strings.Join(parts[2:], " "), `"',`)
				}

				if constraint, err := berkshelf.NewConstraint(constraintStr); err == nil {
					metadata.Dependencies[name] = constraint
				}
			}
		}
	}

	return metadata, nil
}

// extractRubyString extracts string values from Ruby code (simplified).
func extractRubyString(content, key string) []string {
	var matches []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key) && strings.Contains(line, " ") {
			// Extract the value after the key
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				// Remove quotes
				value = strings.Trim(value, `"'`)
				matches = append(matches, value)
			}
		}
	}

	return matches
}

// ListVersions returns the versions available in the path source.
func (p *PathSource) ListVersions(ctx context.Context, name string) ([]*berkshelf.Version, error) {
	cookbookPath, err := p.findCookbookPath(name)
	if err != nil {
		return nil, err
	}

	metadata, err := p.ReadMetadata(cookbookPath)
	if err != nil {
		return nil, err
	}

	// Path sources only have one version
	return []*berkshelf.Version{metadata.Version}, nil
}

// FetchMetadata returns the metadata for a cookbook.
func (p *PathSource) FetchMetadata(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Metadata, error) {
	cookbookPath, err := p.findCookbookPath(name)
	if err != nil {
		return nil, err
	}

	metadata, err := p.ReadMetadata(cookbookPath)
	if err != nil {
		return nil, err
	}

	// Check version matches
	if version != nil && metadata.Version.String() != version.String() {
		return nil, &ErrVersionNotFound{
			Name:    name,
			Version: version.String(),
		}
	}

	return metadata, nil
}

// FetchCookbook returns the cookbook from the path.
func (p *PathSource) FetchCookbook(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Cookbook, error) {
	cookbookPath, err := p.findCookbookPath(name)
	if err != nil {
		return nil, err
	}

	metadata, err := p.ReadMetadata(cookbookPath)
	if err != nil {
		return nil, err
	}

	// Check version matches
	if version != nil && metadata.Version.String() != version.String() {
		return nil, &ErrVersionNotFound{
			Name:    name,
			Version: version.String(),
		}
	}

	return &berkshelf.Cookbook{
		Name:     name,
		Version:  metadata.Version,
		Metadata: metadata,
		Path:     cookbookPath,
	}, nil
}

// DownloadAndExtractCookbook copies the cookbook files from the local path to the target directory.
func (p *PathSource) DownloadAndExtractCookbook(ctx context.Context, cookbook *berkshelf.Cookbook, targetDir string) error {
	sourceDir := cookbook.Path
	if sourceDir == "" {
		// Find the cookbook path if not already set
		cookbookPath, err := p.findCookbookPath(cookbook.Name)
		if err != nil {
			return fmt.Errorf("finding cookbook path: %w", err)
		}
		sourceDir = cookbookPath
	}

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	// Copy all files from source to target
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Copy file
		return copyFile(path, targetPath, info.Mode())
	})

	if err != nil {
		return fmt.Errorf("copying cookbook files: %w", err)
	}

	// Update cookbook path
	cookbook.Path = targetDir

	return nil
}

// Search is not implemented for path sources.
func (p *PathSource) Search(ctx context.Context, query string) ([]*berkshelf.Cookbook, error) {
	return nil, ErrNotImplemented
}

// GetSourceLocation returns the source location for this path source
func (p *PathSource) GetSourceLocation() *berkshelf.SourceLocation {
	return &berkshelf.SourceLocation{
		Type: "path",
		Path: p.basePath,
	}
}

// GetSourceType returns the source type
func (p *PathSource) GetSourceType() string {
	return "path"
}

// GetSourceURL returns the source URL (empty for path sources)
func (p *PathSource) GetSourceURL() string {
	return ""
}
