package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

// SupermarketSource implements CookbookSource for Chef Supermarket API.
type SupermarketSource struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	priority   int
}

// NewSupermarketSource creates a new Supermarket source.
func NewSupermarketSource(baseURL string) *SupermarketSource {
	if baseURL == "" {
		baseURL = "https://supermarket.chef.io"
	}

	return &SupermarketSource{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		priority: 100, // Default priority
	}
}

// SetAPIKey sets the API key for authenticated requests.
func (s *SupermarketSource) SetAPIKey(key string) {
	s.apiKey = key
}

// Name returns the name of this source.
func (s *SupermarketSource) Name() string {
	return fmt.Sprintf("supermarket (%s)", s.baseURL)
}

// Priority returns the priority of this source.
func (s *SupermarketSource) Priority() int {
	return s.priority
}

// SetPriority sets the priority of this source.
func (s *SupermarketSource) SetPriority(priority int) {
	s.priority = priority
}

// cookbookResponse represents the API response for a cookbook.
type cookbookResponse struct {
	Name            string                 `json:"name"`
	Maintainer      string                 `json:"maintainer"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	LatestVersion   string                 `json:"latest_version"`
	ExternalURL     string                 `json:"external_url"`
	SourceURL       string                 `json:"source_url"`
	IssuesURL       string                 `json:"issues_url"`
	Deprecated      bool                   `json:"deprecated"`
	Versions        []string               `json:"versions"`
	VersionsDetails map[string]versionInfo `json:"versions_details"`
}

// versionInfo contains information about a specific version.
type versionInfo struct {
	Version      string            `json:"version"`
	TarballURL   string            `json:"tarball_file_url"`
	Dependencies map[string]string `json:"dependencies"`
}

// ListVersions returns all available versions of a cookbook.
func (s *SupermarketSource) ListVersions(ctx context.Context, name string) ([]*berkshelf.Version, error) {
	endpoint := fmt.Sprintf("%s/api/v1/cookbooks/%s", s.baseURL, url.PathEscape(name))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if s.apiKey != "" {
		req.Header.Set("X-Ops-Userid", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, &ErrSourceUnavailable{Source: s.Name(), Reason: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &ErrCookbookNotFound{Name: name}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supermarket API error: %d %s", resp.StatusCode, string(body))
	}

	var cookbook cookbookResponse
	if err := json.NewDecoder(resp.Body).Decode(&cookbook); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	versions := make([]*berkshelf.Version, 0, len(cookbook.Versions))
	for _, versionStr := range cookbook.Versions {
		v, err := berkshelf.NewVersion(versionStr)
		if err != nil {
			continue // Skip invalid versions
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// cookbookVersionResponse represents the API response for a specific cookbook version.
type cookbookVersionResponse struct {
	Version      string            `json:"version"`
	TarballURL   string            `json:"tarball_file_url"`
	Dependencies map[string]string `json:"dependencies"`
	Attributes   []string          `json:"attributes"`
	Recipes      []recipeInfo      `json:"recipes"`
	Resources    []string          `json:"resources"`
	Providers    []string          `json:"providers"`
	RootFiles    []fileInfo        `json:"root_files"`
}

type recipeInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type fileInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
}

// FetchMetadata downloads just the metadata for a cookbook version.
func (s *SupermarketSource) FetchMetadata(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Metadata, error) {
	endpoint := fmt.Sprintf("%s/api/v1/cookbooks/%s/versions/%s",
		s.baseURL, url.PathEscape(name), url.PathEscape(version.String()))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if s.apiKey != "" {
		req.Header.Set("X-Ops-Userid", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, &ErrSourceUnavailable{Source: s.Name(), Reason: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &ErrVersionNotFound{Name: name, Version: version.String()}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supermarket API error: %d %s", resp.StatusCode, string(body))
	}

	var versionResp cookbookVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&versionResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Convert dependencies
	dependencies := make(map[string]*berkshelf.Constraint)
	for depName, constraintStr := range versionResp.Dependencies {
		constraint, err := berkshelf.NewConstraint(constraintStr)
		if err != nil {
			continue // Skip invalid constraints
		}
		dependencies[depName] = constraint
	}

	metadata := &berkshelf.Metadata{
		Name:         name,
		Version:      version,
		Dependencies: dependencies,
		// Additional fields can be populated from the API response
	}

	return metadata, nil
}

// FetchCookbook downloads the complete cookbook at the specified version.
func (s *SupermarketSource) FetchCookbook(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Cookbook, error) {
	// First fetch the metadata
	metadata, err := s.FetchMetadata(ctx, name, version)
	if err != nil {
		return nil, err
	}

	// For now, just return a cookbook with the metadata
	// In a full implementation, we would download the tarball and extract it
	cookbook := &berkshelf.Cookbook{
		Name:     name,
		Version:  version,
		Metadata: metadata,
		Path:     "", // Would be set after downloading
	}

	return cookbook, nil
}

// Search returns cookbooks matching the query.
func (s *SupermarketSource) Search(ctx context.Context, query string) ([]*berkshelf.Cookbook, error) {
	endpoint := fmt.Sprintf("%s/api/v1/search?q=%s", s.baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if s.apiKey != "" {
		req.Header.Set("X-Ops-Userid", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, &ErrSourceUnavailable{Source: s.Name(), Reason: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supermarket API error: %d %s", resp.StatusCode, string(body))
	}

	// Parse search results
	var results struct {
		Items []cookbookResponse `json:"items"`
		Total int                `json:"total"`
		Start int                `json:"start"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	cookbooks := make([]*berkshelf.Cookbook, 0, len(results.Items))
	for _, item := range results.Items {
		// Create a minimal cookbook from search results
		v, _ := berkshelf.NewVersion(item.LatestVersion)
		cookbook := &berkshelf.Cookbook{
			Name:    item.Name,
			Version: v,
			Metadata: &berkshelf.Metadata{
				Name:        item.Name,
				Version:     v,
				Description: item.Description,
			},
		}
		cookbooks = append(cookbooks, cookbook)
	}

	return cookbooks, nil
}
