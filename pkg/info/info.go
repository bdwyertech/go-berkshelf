package info

import (
	"context"
	"fmt"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyer/go-berkshelf/pkg/source"
)

// CookbookInfo contains detailed information about a cookbook
type CookbookInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version,omitempty"`
	Description  string            `json:"description,omitempty"`
	Maintainer   string            `json:"maintainer,omitempty"`
	License      string            `json:"license,omitempty"`
	Source       string            `json:"source"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
	Versions     []string          `json:"available_versions,omitempty"`
}

// Provider provides cookbook information
type Provider struct {
	sourceManager *source.Manager
}

// New creates a new info provider
func New(sourceManager *source.Manager) *Provider {
	return &Provider{
		sourceManager: sourceManager,
	}
}

// GetInfo retrieves information about a cookbook
// If requestedVersion is empty, it returns info for the latest version
func (p *Provider) GetInfo(ctx context.Context, cookbookName string, requestedVersion string) (*CookbookInfo, error) {
	info := &CookbookInfo{
		Name: cookbookName,
	}

	var sourceUsed string

	// Try to get information from each source
	for _, src := range p.sourceManager.GetSources() {
		// Get available versions
		versions, err := src.ListVersions(ctx, cookbookName)
		if err != nil {
			continue // Try next source
		}

		if len(versions) == 0 {
			continue // No versions found in this source
		}

		sourceUsed = src.Name()
		info.Source = sourceUsed

		// Convert versions to strings
		for _, version := range versions {
			info.Versions = append(info.Versions, version.String())
		}

		// Determine which version to get details for
		var targetVersion string
		if requestedVersion != "" {
			targetVersion = requestedVersion
		} else if len(versions) > 0 {
			// Use latest version
			targetVersion = versions[0].String()
		}

		if targetVersion != "" {
			info.Version = targetVersion

			// Get cookbook metadata
			targetVer, err := berkshelf.NewVersion(targetVersion)
			if err != nil {
				continue // Try next source
			}

			cookbook, err := src.FetchCookbook(ctx, cookbookName, targetVer)
			if err != nil {
				continue // Try next source
			}

			// Fill in metadata
			if cookbook.Metadata != nil {
				info.Description = cookbook.Metadata.Description
				info.Maintainer = cookbook.Metadata.Maintainer
				info.License = cookbook.Metadata.License

				// Convert dependencies
				if len(cookbook.Metadata.Dependencies) > 0 {
					info.Dependencies = make(map[string]string)
					for depName, constraint := range cookbook.Metadata.Dependencies {
						if constraint != nil {
							info.Dependencies[depName] = constraint.String()
						} else {
							info.Dependencies[depName] = ">= 0.0.0"
						}
					}
				}
			}
		}

		// Successfully got info from this source
		return info, nil
	}

	return nil, fmt.Errorf("cookbook %s not found in any source", cookbookName)
}

// GetVersions retrieves just the available versions for a cookbook
func (p *Provider) GetVersions(ctx context.Context, cookbookName string) ([]string, error) {
	for _, src := range p.sourceManager.GetSources() {
		versions, err := src.ListVersions(ctx, cookbookName)
		if err != nil {
			continue // Try next source
		}

		if len(versions) > 0 {
			// Convert to strings
			versionStrings := make([]string, len(versions))
			for i, v := range versions {
				versionStrings[i] = v.String()
			}
			return versionStrings, nil
		}
	}

	return nil, fmt.Errorf("cookbook %s not found in any source", cookbookName)
}
