// Package source provides interfaces and implementations for cookbook sources.
package source

import (
	"context"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

// CookbookSource defines the interface for fetching cookbooks from various sources.
type CookbookSource interface {
	// Name returns the human-readable name of this source.
	Name() string

	// Priority returns the priority of this source (higher = preferred).
	Priority() int

	// ListVersions returns all available versions of a cookbook.
	ListVersions(ctx context.Context, name string) ([]*berkshelf.Version, error)

	// FetchCookbook downloads the complete cookbook at the specified version.
	FetchCookbook(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Cookbook, error)

	// FetchMetadata downloads just the metadata for a cookbook version.
	FetchMetadata(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Metadata, error)

	// DownloadAndExtractCookbook downloads the cookbook files and extracts them to the specified directory.
	DownloadAndExtractCookbook(ctx context.Context, cookbook *berkshelf.Cookbook, targetDir string) error

	// Search returns cookbooks matching the query (optional, may return ErrNotImplemented).
	Search(ctx context.Context, query string) ([]*berkshelf.Cookbook, error)

	// GetSourceLocation returns the source location for this source
	GetSourceLocation() *berkshelf.SourceLocation

	// GetSourceType returns the source type
	GetSourceType() string

	// GetSourceURL returns the source URL
	GetSourceURL() string
}

// SourceFactory creates a CookbookSource from a SourceLocation.
type SourceFactory interface {
	CreateSource(location *berkshelf.SourceLocation) (CookbookSource, error)
}

// Manager coordinates multiple sources.
type Manager struct {
	sources []CookbookSource
}

// NewManager creates a new source manager.
func NewManager() *Manager {
	return &Manager{
		sources: make([]CookbookSource, 0),
	}
}

// AddSource adds a cookbook source to the manager.
func (m *Manager) AddSource(source CookbookSource) {
	m.sources = append(m.sources, source)
}

// GetSources returns all sources in the manager
func (m *Manager) GetSources() []CookbookSource {
	return m.sources
}

// ListVersions queries all sources for available versions.
func (m *Manager) ListVersions(ctx context.Context, name string) ([]*berkshelf.Version, error) {
	versionMap := make(map[string]*berkshelf.Version)

	for _, source := range m.sources {
		versions, err := source.ListVersions(ctx, name)
		if err != nil {
			continue // Try next source
		}

		for _, v := range versions {
			versionMap[v.String()] = v
		}
	}

	// Convert map to slice
	result := make([]*berkshelf.Version, 0, len(versionMap))
	for _, v := range versionMap {
		result = append(result, v)
	}

	return result, nil
}

// FetchCookbook tries to fetch a cookbook from sources in priority order.
func (m *Manager) FetchCookbook(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Cookbook, error) {
	// Sort sources by priority (higher first)
	// TODO: implement sorting

	for _, source := range m.sources {
		cookbook, err := source.FetchCookbook(ctx, name, version)
		if err == nil {
			return cookbook, nil
		}
	}

	return nil, &ErrCookbookNotFound{Name: name, Version: version.String()}
}
