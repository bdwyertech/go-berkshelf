package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
	"github.com/go-chef/chef"
)

// ChefServerSource implements CookbookSource for Chef Server API.
type ChefServerSource struct {
	baseURL    string
	clientName string
	clientKey  string
	priority   int
	chefClient *chef.Client
}

// NewChefServerSource creates a new Chef Server source.
func NewChefServerSource(baseURL, clientName, clientKey string) (*ChefServerSource, error) {
	// Expand tilde in client key path
	if strings.HasPrefix(clientKey, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home directory: %w", err)
		}
		clientKey = filepath.Join(homeDir, clientKey[2:])
	}

	// Read the private key
	keyData, err := os.ReadFile(clientKey)
	if err != nil {
		return nil, fmt.Errorf("reading client key file %s: %w", clientKey, err)
	}

	// Create Chef client
	chefClient, err := chef.NewClient(&chef.Config{
		Name:    clientName,
		Key:     string(keyData),
		BaseURL: baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating chef client: %w", err)
	}

	return &ChefServerSource{
		baseURL:    baseURL,
		clientName: clientName,
		clientKey:  clientKey,
		priority:   150, // Higher priority than Supermarket
		chefClient: chefClient,
	}, nil
}

// Name returns the name of this source.
func (s *ChefServerSource) Name() string {
	return fmt.Sprintf("chef-server (%s)", s.baseURL)
}

// Priority returns the priority of this source.
func (s *ChefServerSource) Priority() int {
	return s.priority
}

// SetPriority sets the priority of this source.
func (s *ChefServerSource) SetPriority(priority int) {
	s.priority = priority
}

// ListVersions returns all available versions of a cookbook.
func (s *ChefServerSource) ListVersions(ctx context.Context, name string) ([]*berkshelf.Version, error) {
	// Get cookbook list with versions
	cookbooks, err := s.chefClient.Cookbooks.List()
	if err != nil {
		return nil, &ErrSourceUnavailable{Source: s.Name(), Reason: err.Error()}
	}

	cookbookVersions, exists := cookbooks[name]
	if !exists {
		return nil, &ErrCookbookNotFound{Name: name}
	}

	versions := make([]*berkshelf.Version, 0, len(cookbookVersions.Versions))
	for _, versionInfo := range cookbookVersions.Versions {
		v, err := berkshelf.NewVersion(versionInfo.Version)
		if err != nil {
			continue // Skip invalid versions
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// FetchMetadata downloads just the metadata for a cookbook version.
func (s *ChefServerSource) FetchMetadata(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Metadata, error) {
	cookbook, err := s.chefClient.Cookbooks.GetVersion(name, version.String())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, &ErrVersionNotFound{Name: name, Version: version.String()}
		}
		return nil, &ErrSourceUnavailable{Source: s.Name(), Reason: err.Error()}
	}

	// Convert dependencies from Chef Server format to berkshelf constraints
	dependencies := make(map[string]*berkshelf.Constraint)
	if cookbook.Metadata.Depends != nil {
		for depName, depConstraint := range cookbook.Metadata.Depends {
			// Parse the constraint string into a berkshelf.Constraint
			constraint, err := berkshelf.NewConstraint(depConstraint)
			if err != nil {
				// If constraint parsing fails, use a default constraint
				constraint, _ = berkshelf.NewConstraint(">= 0.0.0")
			}
			dependencies[depName] = constraint
		}
	}

	metadata := &berkshelf.Metadata{
		Name:         name,
		Version:      version,
		Dependencies: dependencies,
		Description:  cookbook.Metadata.Description,
		Maintainer:   cookbook.Metadata.Maintainer,
		License:      cookbook.Metadata.License,
	}

	return metadata, nil
}

// FetchCookbook downloads the complete cookbook at the specified version.
func (s *ChefServerSource) FetchCookbook(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Cookbook, error) {
	// First fetch the metadata
	metadata, err := s.FetchMetadata(ctx, name, version)
	if err != nil {
		return nil, err
	}

	// Create the cookbook object
	result := &berkshelf.Cookbook{
		Name:         name,
		Version:      version,
		Metadata:     metadata,
		Dependencies: metadata.Dependencies,
		Source: berkshelf.SourceLocation{
			Type: "chef_server",
			URL:  s.baseURL,
		},
		Path: "", // Will be set when downloaded
	}

	return result, nil
}

// DownloadAndExtractCookbook downloads the cookbook files and extracts them to the specified directory.
func (s *ChefServerSource) DownloadAndExtractCookbook(ctx context.Context, cookbook *berkshelf.Cookbook, targetDir string) error {
	if err := s.chefClient.Cookbooks.DownloadTo(cookbook.Name, cookbook.Version.String(), filepath.Dir(targetDir)); err != nil {
		return fmt.Errorf("downloading cookbook %s version %s: %w", cookbook.Name, cookbook.Version.String(), err)
	}
	if err := os.Remove(targetDir); err != nil {
		return fmt.Errorf("error removing target directory: %w", err)
	}
	if err := os.Rename(targetDir+"-"+cookbook.Version.String(), targetDir); err != nil {
		return fmt.Errorf("error renaming target directory: %w", err)
	}

	// Set the cookbook path
	cookbook.Path = targetDir

	return nil
}

// Search returns cookbooks matching the query.
func (s *ChefServerSource) Search(ctx context.Context, query string) ([]*berkshelf.Cookbook, error) {
	// Chef Server doesn't have a direct search API like Supermarket
	// This is a simplified implementation that lists all cookbooks
	cookbooks, err := s.chefClient.Cookbooks.List()
	if err != nil {
		return nil, &ErrSourceUnavailable{Source: s.Name(), Reason: err.Error()}
	}

	var results []*berkshelf.Cookbook
	for name := range cookbooks {
		if strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
			// Get the latest version
			versions, err := s.ListVersions(ctx, name)
			if err != nil || len(versions) == 0 {
				continue
			}

			// Find the latest version
			latest := versions[0]
			for _, v := range versions[1:] {
				if latest.LessThan(v) {
					latest = v
				}
			}

			cookbook := &berkshelf.Cookbook{
				Name:    name,
				Version: latest,
				Metadata: &berkshelf.Metadata{
					Name:    name,
					Version: latest,
				},
			}
			results = append(results, cookbook)
		}
	}

	return results, nil
}

// GetSourceLocation returns the source location for this chef server source
func (s *ChefServerSource) GetSourceLocation() *berkshelf.SourceLocation {
	return &berkshelf.SourceLocation{
		Type: "chef_server",
		URL:  s.baseURL,
	}
}

// GetSourceType returns the source type
func (s *ChefServerSource) GetSourceType() string {
	return "chef_server"
}

// GetSourceURL returns the source URL
func (s *ChefServerSource) GetSourceURL() string {
	return s.baseURL
}
