package resolver

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyertech/go-berkshelf/pkg/source"
)

// DefaultResolver implements the Resolver interface
type DefaultResolver struct {
	sources       []source.CookbookSource
	cache         *ResolutionCache
	maxCandidates int
	workerCount   int
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
		maxCandidates: 100,                  // Maximum versions to consider per cookbook
		workerCount:   runtime.NumCPU() * 2, // Good for I/O bound operations
	}
}

// NewResolutionCache creates a new resolution cache
func NewResolutionCache() *ResolutionCache {
	return &ResolutionCache{
		versions: make(map[string][]*berkshelf.Version),
		metadata: make(map[string]*berkshelf.Cookbook),
	}
}

// Resolve implements concurrent I/O operations for dependency resolution
func (r *DefaultResolver) Resolve(ctx context.Context, requirements []*Requirement) (*Resolution, error) {
	log.Debugf("Starting concurrent dependency resolution with %d workers...", r.workerCount)

	resolution := NewResolution()

	// Phase 1: Parallel version fetching for all requirements
	versionMap, err := r.fetchAllVersionsConcurrently(ctx, requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}

	// Phase 2: Sequential dependency resolution (must be sequential)
	resolvedCookbooks, err := r.resolveSequentially(ctx, requirements, versionMap, resolution)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Phase 3: Parallel cookbook downloading
	err = r.downloadCookbooksConcurrently(ctx, resolvedCookbooks, resolution)
	if err != nil {
		return nil, fmt.Errorf("failed to download cookbooks: %w", err)
	}

	return resolution, nil
}

// fetchAllVersionsConcurrently fetches versions for all cookbooks in parallel using conc/pool
func (r *DefaultResolver) fetchAllVersionsConcurrently(ctx context.Context, requirements []*Requirement) (map[string]map[source.CookbookSource][]*berkshelf.Version, error) {
	versionMap := make(map[string]map[source.CookbookSource][]*berkshelf.Version)
	var mu sync.Mutex

	// Create a result pool with context support
	p := pool.New().WithContext(ctx).WithMaxGoroutines(r.workerCount)

	// Submit jobs to the pool
	for _, req := range requirements {
		if req.Source != nil {
			// Use specific source
			factory := source.NewFactory()
			specificSource, err := factory.CreateFromLocation(req.Source)
			if err != nil {
				log.Warnf("Failed to create specific source for %s: %v", req.Name, err)
				continue
			}

			// Capture variables for closure
			reqName := req.Name
			src := specificSource

			p.Go(func(ctx context.Context) error {
				versions, err := r.getVersions(ctx, src, reqName)
				if err != nil {
					log.Debugf("Failed to fetch versions for %s from %s: %v", reqName, src.Name(), err)
					return nil // Don't fail the entire operation for individual source failures
				}

				mu.Lock()
				if versionMap[reqName] == nil {
					versionMap[reqName] = make(map[source.CookbookSource][]*berkshelf.Version)
				}
				versionMap[reqName][src] = versions
				mu.Unlock()

				return nil
			})
		} else {
			// Use all global sources
			for _, src := range r.sources {
				// Capture variables for closure
				reqName := req.Name
				currentSrc := src

				p.Go(func(ctx context.Context) error {
					versions, err := r.getVersions(ctx, currentSrc, reqName)
					if err != nil {
						log.Debugf("Failed to fetch versions for %s from %s: %v", reqName, currentSrc.Name(), err)
						return nil // Don't fail the entire operation for individual source failures
					}

					mu.Lock()
					if versionMap[reqName] == nil {
						versionMap[reqName] = make(map[source.CookbookSource][]*berkshelf.Version)
					}
					versionMap[reqName][currentSrc] = versions
					mu.Unlock()

					return nil
				})
			}
		}
	}

	// Wait for all jobs to complete
	if err := p.Wait(); err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}

	return versionMap, nil
}

// resolveSequentially performs dependency resolution using pre-fetched version data
func (r *DefaultResolver) resolveSequentially(ctx context.Context, requirements []*Requirement, versionMap map[string]map[source.CookbookSource][]*berkshelf.Version, resolution *Resolution) ([]*ResolvedCookbook, error) {
	var resolvedCookbooks []*ResolvedCookbook
	queue := make([]*Requirement, len(requirements))
	copy(queue, requirements)
	processed := make(map[string]bool)
	resolving := make(map[string]bool)   // Track cookbooks currently being resolved to detect cycles
	dependencyChain := make([]string, 0) // Track current dependency chain for cycle detection

	for len(queue) > 0 {
		req := queue[0]
		queue = queue[1:]

		if processed[req.Name] {
			continue
		}

		// Check for circular dependency in current resolution chain
		if resolving[req.Name] {
			cycleError := fmt.Errorf("circular dependency detected involving cookbook '%s' in chain: %v -> %s",
				req.Name, dependencyChain, req.Name)
			resolution.AddError(cycleError)
			log.Warnf("Circular dependency detected: %s in chain %v", req.Name, dependencyChain)
			continue
		}

		resolving[req.Name] = true
		dependencyChain = append(dependencyChain, req.Name)

		// Find best version using pre-fetched data
		version, cookbookSource, err := r.findBestVersionFromCache(req, versionMap)
		if err != nil {
			// Try to fetch versions for this cookbook if not in cache
			// Use first available source as fallback
			if len(r.sources) == 0 {
				resolution.AddError(fmt.Errorf("failed to resolve %s: no sources available", req.Name))
				resolving[req.Name] = false
				dependencyChain = dependencyChain[:len(dependencyChain)-1]
				continue
			}

			newVersions, fetchErr := r.getVersions(ctx, r.sources[0], req.Name)
			if fetchErr != nil {
				resolution.AddError(fmt.Errorf("failed to resolve %s: %w", req.Name, err))
				resolving[req.Name] = false
				dependencyChain = dependencyChain[:len(dependencyChain)-1]
				continue
			}

			// Add to version map
			if versionMap[req.Name] == nil {
				versionMap[req.Name] = make(map[source.CookbookSource][]*berkshelf.Version)
			}
			versionMap[req.Name][r.sources[0]] = newVersions

			// Try again
			version, cookbookSource, err = r.findBestVersionFromCache(req, versionMap)
			if err != nil {
				resolution.AddError(fmt.Errorf("failed to resolve %s: %w", req.Name, err))
				resolving[req.Name] = false
				dependencyChain = dependencyChain[:len(dependencyChain)-1]
				continue
			}
		}

		log.Infof("Using %s (%s) from %s", req.Name, version.String(), cookbookSource.Name())

		// Fetch cookbook metadata to get dependencies
		cookbook, err := r.fetchCookbook(ctx, req.Name, version, cookbookSource)
		if err != nil {
			resolution.AddError(fmt.Errorf("failed to fetch cookbook %s@%s: %w", req.Name, version.String(), err))
			resolving[req.Name] = false
			dependencyChain = dependencyChain[:len(dependencyChain)-1]
			continue
		}

		// Create resolved cookbook
		resolved := &ResolvedCookbook{
			Name:         req.Name,
			Version:      version,
			Source:       cookbookSource.GetSourceLocation(),
			SourceRef:    cookbookSource,
			Dependencies: make(map[string]*berkshelf.Version),
			Cookbook:     cookbook,
		}

		resolvedCookbooks = append(resolvedCookbooks, resolved)

		// Add to graph
		node := resolution.Graph.AddCookbook(cookbook)
		node.Resolved = true

		// Add dependencies to queue and build dependency graph
		if cookbook.Metadata != nil && cookbook.Metadata.Dependencies != nil {
			for depName, constraint := range cookbook.Metadata.Dependencies {
				// Add dependency to queue if not processed
				if !processed[depName] {
					depReq := &Requirement{
						Name:       depName,
						Constraint: constraint,
					}
					queue = append(queue, depReq)
					resolved.Dependencies[depName] = nil // Will be filled later
				}

				// Create or get dependency node for graph building
				var depNode *CookbookNode
				if existingNode, exists := resolution.Graph.GetCookbook(depName); exists {
					depNode = existingNode
				} else {
					// Create a placeholder cookbook for the dependency
					placeholderCookbook := &berkshelf.Cookbook{
						Name:    depName,
						Version: nil, // Will be filled when resolved
					}
					depNode = resolution.Graph.AddCookbook(placeholderCookbook)
				}

				// Add dependency edge to graph
				resolution.Graph.AddDependency(node, depNode, constraint)

				// Check for cycles after adding each dependency
				if resolution.Graph.HasCycles() {
					cycleError := fmt.Errorf("circular dependency detected: %s depends on %s, creating a cycle", req.Name, depName)
					resolution.AddError(cycleError)
					log.Warnf("Circular dependency detected: %s -> %s creates cycle", req.Name, depName)
				}
			}
		}

		processed[req.Name] = true
		resolving[req.Name] = false
		dependencyChain = dependencyChain[:len(dependencyChain)-1]
	}

	// Final check for cycles in the complete graph
	if resolution.Graph.HasCycles() {
		if !resolution.HasErrors() {
			// Only add this error if we haven't already detected cycles
			cycleError := fmt.Errorf("circular dependencies detected in final cookbook dependency graph")
			resolution.AddError(cycleError)
			log.Warnf("Circular dependencies detected in final dependency graph")
		}
	}

	return resolvedCookbooks, nil
}

// findBestVersionFromCache finds the best version using cached version data
func (r *DefaultResolver) findBestVersionFromCache(req *Requirement, versionMap map[string]map[source.CookbookSource][]*berkshelf.Version) (*berkshelf.Version, source.CookbookSource, error) {
	sourceVersions, exists := versionMap[req.Name]
	if !exists {
		return nil, nil, fmt.Errorf("no versions found for cookbook %s", req.Name)
	}

	var bestVersion *berkshelf.Version
	var bestSource source.CookbookSource

	for src, versions := range sourceVersions {
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

// SetMaxWorkers configures the number of concurrent workers for I/O operations
func (r *DefaultResolver) SetMaxWorkers(workers int) {
	if workers > 0 {
		r.workerCount = workers
		log.Debugf("Set resolver worker count to %d", workers)
	}
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

// downloadCookbooksConcurrently downloads cookbook metadata in parallel using conc/pool
func (r *DefaultResolver) downloadCookbooksConcurrently(ctx context.Context, resolvedCookbooks []*ResolvedCookbook, resolution *Resolution) error {
	var mu sync.Mutex

	// Create a result pool with context support
	p := pool.New().WithContext(ctx).WithMaxGoroutines(r.workerCount)

	// Submit jobs to the pool
	for _, resolved := range resolvedCookbooks {
		// Use the stored source reference
		if resolved.SourceRef == nil {
			log.Warnf("No source reference for %s@%s", resolved.Name, resolved.Version.String())
			continue
		}

		// Capture variables for closure
		name := resolved.Name
		version := resolved.Version
		sourceRef := resolved.SourceRef

		p.Go(func(ctx context.Context) error {
			cookbook, err := r.fetchCookbook(ctx, name, version, sourceRef)
			if err != nil {
				mu.Lock()
				resolution.AddError(fmt.Errorf("failed to fetch %s@%s: %w", name, version.String(), err))
				mu.Unlock()
				return nil // Don't fail the entire operation for individual cookbook failures
			}

			// Find the resolved cookbook and update it
			mu.Lock()
			for _, res := range resolvedCookbooks {
				if res.Name == name && res.Version.Equal(version) {
					res.Cookbook = cookbook
					resolution.AddCookbook(res)
					break
				}
			}
			mu.Unlock()

			return nil
		})
	}

	// Wait for all jobs to complete
	if err := p.Wait(); err != nil {
		return fmt.Errorf("failed to download cookbooks: %w", err)
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
