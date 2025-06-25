package resolver

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
			// Use the requirement's specific source if provided, but enhance it with actual source info
			sourceLocation = req.Source
			// Ensure the source location has all the necessary information from the actual source
			if sourceLocation.Type == "git" {
				// The original source location from Berksfile should already have all options preserved
				log.Debugf("Using original source location with Git options: %+v", sourceLocation.Options)
			}
		} else {
			// Create source location from the actual source that provided the cookbook
			sourceLocation = source.GetSourceLocation()
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

	// If requirement specifies a specific source, create and use only that source
	if req.Source != nil {
		log.Debugf("Creating cookbook-specific source for %s: %s %s", req.Name, req.Source.Type, req.Source.URL)
		factory := source.NewFactory()
		specificSource, err := factory.CreateFromLocation(req.Source)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create specific source for %s: %w", req.Name, err)
		}

		// Get available versions from the specific source
		versions, err := r.getVersions(ctx, specificSource, req.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get versions from specific source: %w", err)
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
				bestSource = specificSource
			}
		}

		if bestVersion == nil {
			return nil, nil, fmt.Errorf("no version found in specific source that satisfies constraint %s", req.Constraint)
		}

		return bestVersion, bestSource, nil
	}

	// Use global sources for cookbooks without specific sources
	log.Debugf("Using global sources for %s", req.Name)
	for _, src := range r.sources {
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
	if src == nil || loc == nil {
		fmt.Printf("DEBUG: sourceMatches: src or loc is nil (src: %v, loc: %v)\n", src != nil, loc != nil)
		return false
	}

	source := strings.Fields(src.Name())

	if source[0] != loc.Type {
		return false
	}
	if source[1][1:len(source[1])-1] != loc.URL {
		return false
	}
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

