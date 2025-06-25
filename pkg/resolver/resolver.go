package resolver

import (
	"context"
	"fmt"
	"sort"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyer/go-berkshelf/pkg/source"
)

// DefaultResolver implements the Resolver interface
type DefaultResolver struct {
	sources       []source.CookbookSource
	cache         *ResolutionCache
	maxCandidates int
	mu            sync.RWMutex
}

// ResolutionCache caches cookbook metadata and available versions
type ResolutionCache struct {
	versions map[string][]*berkshelf.Version // cookbook name -> available versions
	metadata map[string]*berkshelf.Cookbook  // cookbook@version -> metadata
	mu       sync.RWMutex
}

// NewResolver creates a new resolver with the given sources
func NewResolver(sources []source.CookbookSource) *DefaultResolver {
	return &DefaultResolver{
		sources:       sources,
		cache:         NewResolutionCache(),
		maxCandidates: 100, // Maximum versions to consider per cookbook
	}
}

// NewResolutionCache creates a new resolution cache
func NewResolutionCache() *ResolutionCache {
	return &ResolutionCache{
		versions: make(map[string][]*berkshelf.Version),
		metadata: make(map[string]*berkshelf.Cookbook),
	}
}

// Resolve implements the Resolver interface
func (r *DefaultResolver) Resolve(ctx context.Context, requirements []*Requirement) (*Resolution, error) {
	// Create a new resolution
	resolution := NewResolution()

	// Create a work queue for cookbooks to resolve
	queue := make([]*Requirement, len(requirements))
	copy(queue, requirements)

	// Track what we've already processed
	processed := make(map[string]bool)

	// Process the queue until empty
	for len(queue) > 0 {
		// Take the first requirement from the queue
		req := queue[0]
		queue = queue[1:]

		// Skip if already processed
		if processed[req.Name] {
			continue
		}
		processed[req.Name] = true

		// Find the best version that satisfies the constraint
		version, source, err := r.findBestVersion(ctx, req)
		if err != nil {
			resolution.AddError(fmt.Errorf("failed to resolve %s: %w", req.Name, err))
			continue
		}

		// Fetch the cookbook metadata
		cookbook, err := r.fetchCookbook(ctx, req.Name, version, source)
		if err != nil {
			resolution.AddError(fmt.Errorf("failed to fetch %s@%s: %w", req.Name, version.String(), err))
			continue
		}

		// Create source location from the actual source that provided the cookbook
		var sourceLocation *berkshelf.SourceLocation
		if req.Source != nil {
			// Use the requirement's specific source if provided
			sourceLocation = req.Source
		} else {
			// Create source location from the actual source that provided the cookbook
			sourceLocation = createSourceLocationFromSource(source)
		}

		// Add to resolution
		resolved := &ResolvedCookbook{
			Name:         cookbook.Name,
			Version:      version,
			Source:       sourceLocation,
			Dependencies: make(map[string]*berkshelf.Version),
			Cookbook:     cookbook,
		}

		// Add to graph
		node := resolution.Graph.AddCookbook(cookbook)
		node.Resolved = true

		// Process dependencies
		for depName, depConstraint := range cookbook.Dependencies {
			// Add dependency to queue if not already processed
			if !processed[depName] {
				depReq := NewRequirement(depName, depConstraint)
				queue = append(queue, depReq)
			}

			// Add edge in graph
			if depNode, exists := resolution.Graph.GetCookbook(depName); exists {
				resolution.Graph.AddDependency(node, depNode, depConstraint)
			} else {
				// Create placeholder node for dependency
				depCookbook := &berkshelf.Cookbook{
					Name: depName,
				}
				depNode := resolution.Graph.AddCookbook(depCookbook)
				resolution.Graph.AddDependency(node, depNode, depConstraint)
			}
		}

		resolution.AddCookbook(resolved)
	}

	// Check for cycles
	if resolution.Graph.HasCycles() {
		resolution.AddError(fmt.Errorf("circular dependency detected"))
	}

	return resolution, nil
}

// findBestVersion finds the best version that satisfies the constraint
func (r *DefaultResolver) findBestVersion(ctx context.Context, req *Requirement) (*berkshelf.Version, source.CookbookSource, error) {
	var bestVersion *berkshelf.Version
	var bestSource source.CookbookSource

	// Try each source in priority order
	for _, src := range r.sources {
		// Skip if requirement specifies a different source
		if req.Source != nil && !sourceMatches(src, req.Source) {
			continue
		}

		// Get available versions from this source
		versions, err := r.getVersions(ctx, src, req.Name)
		if err != nil {
			continue // Try next source
		}

		// Find the best version that satisfies the constraint
		for _, v := range versions {
			// Skip if doesn't satisfy constraint
			if req.Constraint != nil && !req.Constraint.Check(v) {
				continue
			}

			// Use the highest version that satisfies
			if bestVersion == nil || v.GreaterThan(bestVersion) {
				bestVersion = v
				bestSource = src
			}
		}
	}

	if bestVersion == nil {
		return nil, nil, fmt.Errorf("no version found that satisfies constraint %s", req.Constraint)
	}

	return bestVersion, bestSource, nil
}

// getVersions gets available versions from cache or source
func (r *DefaultResolver) getVersions(ctx context.Context, src source.CookbookSource, name string) ([]*berkshelf.Version, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", src.Name(), name)
	if versions := r.cache.GetVersions(cacheKey); versions != nil {
		return versions, nil
	}

	// Fetch from source
	versions, err := src.ListVersions(ctx, name)
	if err != nil {
		return nil, err
	}

	// Sort versions in descending order
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].GreaterThan(versions[j])
	})

	// Limit the number of versions to consider
	if len(versions) > r.maxCandidates {
		versions = versions[:r.maxCandidates]
	}

	// Cache the result
	r.cache.SetVersions(cacheKey, versions)

	return versions, nil
}

// fetchCookbook fetches cookbook metadata from cache or source
func (r *DefaultResolver) fetchCookbook(ctx context.Context, name string, version *berkshelf.Version, src source.CookbookSource) (*berkshelf.Cookbook, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s@%s", name, version.String())
	if cookbook := r.cache.GetMetadata(cacheKey); cookbook != nil {
		return cookbook, nil
	}

	// Fetch from source
	cookbook, err := src.FetchCookbook(ctx, name, version)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cache.SetMetadata(cacheKey, cookbook)

	return cookbook, nil
}

// sourceMatches checks if a source matches the required source location
func sourceMatches(src source.CookbookSource, loc *berkshelf.SourceLocation) bool {
	// Since sources don't expose their location, we'll need to implement this differently
	// For now, we'll just return true to allow all sources
	// TODO: Implement proper source matching
	return true
}

// Cache methods

// GetVersions retrieves versions from cache
func (c *ResolutionCache) GetVersions(key string) []*berkshelf.Version {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if versions, exists := c.versions[key]; exists {
		// Return a copy to prevent modification
		result := make([]*berkshelf.Version, len(versions))
		copy(result, versions)
		return result
	}

	return nil
}

// SetVersions stores versions in cache
func (c *ResolutionCache) SetVersions(key string, versions []*berkshelf.Version) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store a copy to prevent modification
	c.versions[key] = make([]*berkshelf.Version, len(versions))
	copy(c.versions[key], versions)
}

// GetMetadata retrieves cookbook metadata from cache
func (c *ResolutionCache) GetMetadata(key string) *berkshelf.Cookbook {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.metadata[key]
}

// SetMetadata stores cookbook metadata in cache
func (c *ResolutionCache) SetMetadata(key string, cookbook *berkshelf.Cookbook) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metadata[key] = cookbook
}

// Clear clears the cache
func (c *ResolutionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.versions = make(map[string][]*berkshelf.Version)
	c.metadata = make(map[string]*berkshelf.Cookbook)
}

// createSourceLocationFromSource creates a SourceLocation from a CookbookSource
func createSourceLocationFromSource(src source.CookbookSource) *berkshelf.SourceLocation {
	if src == nil {
		return nil
	}

	// Extract URL from source name - this is a bit hacky but works for our current sources
	name := src.Name()

	log.Debugf("Creating source location from source: %s", name)

	// ChefServerSource names are like "chef-server (https://chef.example.com/organizations/myorg)"
	if len(name) > 12 && name[:12] == "chef-server " {
		url := name[13 : len(name)-1] // Remove "chef-server (" and ")"
		log.Debugf("Chef server URL: %s", url)
		return &berkshelf.SourceLocation{
			Type: "chef_server",
			URL:  url,
		}
	}

	// SupermarketSource names are like "supermarket (https://example.com)"
	if len(name) > 12 && name[:12] == "supermarket " {
		url := name[13 : len(name)-1] // Remove "supermarket (" and ")"
		log.Debugf("Supermarket URL: %s", url)
		return &berkshelf.SourceLocation{
			Type: "supermarket",
			URL:  url,
		}
	}

	// GitSource names are like "git (https://github.com/...)"
	if len(name) > 4 && name[:4] == "git " {
		url := name[5 : len(name)-1] // Remove "git (" and ")"
		log.Debugf("Git URL: %s", url)
		return &berkshelf.SourceLocation{
			Type: "git",
			URL:  url,
		}
	}

	// PathSource names are like "path (/local/path)"
	if len(name) > 5 && name[:5] == "path " {
		path := name[6 : len(name)-1] // Remove "path (" and ")"
		log.Debugf("Path: %s", path)
		return &berkshelf.SourceLocation{
			Type: "path",
			Path: path,
		}
	}

	// Default fallback
	log.Debugf("Unknown source type, defaulting to supermarket: %s", name)
	return &berkshelf.SourceLocation{
		Type: "supermarket",
		URL:  "https://supermarket.chef.io",
	}
}
