package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	// Get the cookbook version details from Chef server
	chefCookbook, err := s.chefClient.Cookbooks.GetVersion(cookbook.Name, cookbook.Version.String())
	if err != nil {
		return fmt.Errorf("fetching cookbook details: %w", err)
	}

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	// Download all cookbook files
	fileTypes := map[string][]chef.CookbookItem{
		"recipes":     chefCookbook.Recipes,
		"attributes":  chefCookbook.Attributes,
		"libraries":   chefCookbook.Libraries,
		"templates":   chefCookbook.Templates,
		"files":       chefCookbook.Files,
		"resources":   chefCookbook.Resources,
		"providers":   chefCookbook.Providers,
		"definitions": chefCookbook.Definitions,
	}

	for fileType, items := range fileTypes {
		for _, item := range items {
			if err := s.downloadFile(ctx, item, targetDir, fileType); err != nil {
				return fmt.Errorf("downloading %s file %s: %w", fileType, item.Name, err)
			}
		}
	}

	// Download root files
	for _, item := range chefCookbook.RootFiles {
		if err := s.downloadFile(ctx, item, targetDir, ""); err != nil {
			return fmt.Errorf("downloading root file %s: %w", item.Name, err)
		}
	}

	// Create metadata.json
	metadataPath := filepath.Join(targetDir, "metadata.json")
	metadataData := map[string]interface{}{
		"name":         cookbook.Name,
		"version":      cookbook.Version.String(),
		"description":  cookbook.Metadata.Description,
		"maintainer":   cookbook.Metadata.Maintainer,
		"license":      cookbook.Metadata.License,
		"dependencies": make(map[string]string),
	}

	// Add dependencies
	deps := metadataData["dependencies"].(map[string]string)
	for name, constraint := range cookbook.Dependencies {
		deps[name] = constraint.String()
	}

	metadataJSON, err := json.MarshalIndent(metadataData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return fmt.Errorf("writing metadata.json: %w", err)
	}

	// Set the cookbook path
	cookbook.Path = targetDir

	return nil
}

// downloadFile downloads a single file from the Chef server.
func (s *ChefServerSource) downloadFile(ctx context.Context, item chef.CookbookItem, targetDir, fileType string) error {
	// Construct the target path
	var targetPath string
	if fileType == "" {
		// Root file
		targetPath = filepath.Join(targetDir, item.Name)
	} else {
		targetPath = filepath.Join(targetDir, fileType, item.Name)
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Download the file - note the field is Url, not URL
	req, err := http.NewRequestWithContext(ctx, "GET", item.Url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Create a custom HTTP client with authentication
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// For simplicity, we'll make an unauthenticated request first
	// In a production implementation, you'd need to sign the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	// Create the file
	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", targetPath, err)
	}
	defer outFile.Close()

	// Copy the content
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("writing file %s: %w", targetPath, err)
	}

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
