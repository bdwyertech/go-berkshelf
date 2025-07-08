package cache

import (
	"context"
	"fmt"

	"github.com/schollz/progressbar/v3"
	"github.com/sourcegraph/conc/pool"

	"github.com/bdwyer/go-berkshelf/internal/config"
	"github.com/bdwyer/go-berkshelf/pkg/resolver"
	"github.com/bdwyer/go-berkshelf/pkg/source"
)

// Installer handles cookbook caching during install operations
type Installer struct {
	cache         *Cache
	sourceManager *source.Manager
	config        *config.Config
}

// NewInstaller creates a new cache installer
func NewInstaller(cache *Cache, sourceManager *source.Manager, config *config.Config) *Installer {
	return &Installer{
		cache:         cache,
		sourceManager: sourceManager,
		config:        config,
	}
}

// CacheCheckResult contains the result of cache checking
type CacheCheckResult struct {
	CachedCookbooks      []*resolver.ResolvedCookbook
	UncachedRequirements []*resolver.Requirement
	CacheHitCount        int
	CacheMissCount       int
}

// CheckCacheForRequirements checks which requirements are already cached
func (i *Installer) CheckCacheForRequirements(ctx context.Context, requirements []*resolver.Requirement) (*CacheCheckResult, error) {
	result := &CacheCheckResult{
		CachedCookbooks:      make([]*resolver.ResolvedCookbook, 0),
		UncachedRequirements: make([]*resolver.Requirement, 0),
	}

	for _, req := range requirements {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Check if we have this cookbook cached
		// For requirements, we need to check if we have any version that satisfies the constraint
		cached := i.findCachedCookbookForRequirement(req)
		if cached != nil {
			result.CachedCookbooks = append(result.CachedCookbooks, cached)
			result.CacheHitCount++
		} else {
			result.UncachedRequirements = append(result.UncachedRequirements, req)
			result.CacheMissCount++
		}
	}

	return result, nil
}

// DownloadAndCache downloads and caches resolved cookbooks with progress reporting
func (i *Installer) DownloadAndCache(ctx context.Context, resolution *resolver.Resolution) error {
	cookbooks := resolution.AllCookbooks()
	if len(cookbooks) == 0 {
		return nil
	}

	// Create progress bar
	bar := progressbar.NewOptions(len(cookbooks),
		progressbar.OptionSetDescription("Downloading and caching cookbooks"),
		progressbar.OptionSetWidth(50),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	// Use worker pool for concurrent downloads
	concurrency := i.config.GetConcurrency()
	if concurrency <= 0 {
		concurrency = 5 // fallback default
	}

	p := pool.New().WithMaxGoroutines(concurrency)

	// Process each cookbook
	for _, cookbook := range cookbooks {
		cookbook := cookbook // capture loop variable
		p.Go(func() {
			defer bar.Add(1)

			select {
			case <-ctx.Done():
				return
			default:
			}

			if err := i.downloadAndCacheCookbook(ctx, cookbook); err != nil {
				// Log error but continue with other cookbooks
				fmt.Printf("\nWarning: failed to cache cookbook %s@%s: %v\n",
					cookbook.Name, cookbook.Version.String(), err)
			}
		})
	}

	// Wait for all downloads to complete
	p.Wait()
	bar.Finish()
	fmt.Println() // Add newline after progress bar

	return nil
}

// downloadAndCacheCookbook downloads and caches a single cookbook
func (i *Installer) downloadAndCacheCookbook(ctx context.Context, cookbook *resolver.ResolvedCookbook) error {
	// Check if already cached
	key := i.cache.getCookbookKey(cookbook.Name, cookbook.Version.String())
	if _, exists := i.cache.Get(key); exists {
		return nil // Already cached
	}

	// Use the source reference from the resolved cookbook
	if cookbook.SourceRef == nil {
		return fmt.Errorf("no source reference for cookbook %s", cookbook.Name)
	}

	// Download cookbook data
	data, err := cookbook.SourceRef.FetchCookbook(ctx, cookbook.Name, cookbook.Version)
	if err != nil {
		return fmt.Errorf("failed to fetch cookbook %s@%s: %w", cookbook.Name, cookbook.Version.String(), err)
	}

	// Cache the cookbook data
	if err := i.cache.Put(key, data); err != nil {
		return fmt.Errorf("failed to cache cookbook %s@%s: %w", cookbook.Name, cookbook.Version.String(), err)
	}

	return nil
}

// findCachedCookbookForRequirement finds a cached cookbook that satisfies the requirement
func (i *Installer) findCachedCookbookForRequirement(req *resolver.Requirement) *resolver.ResolvedCookbook {
	// For now, we'll implement a simple approach - check if we have the exact version
	// In a more sophisticated implementation, we could check all cached versions
	// and find the best match for the constraint

	// Try to find sources that might have this cookbook
	sources := i.sourceManager.GetSources()
	for _, src := range sources {
		versions, err := src.ListVersions(context.Background(), req.Name)
		if err != nil {
			continue
		}

		// Find the best version that satisfies the constraint and is cached
		for _, version := range versions {
			if req.Constraint.Check(version) {
				// Check if this version is cached
				key := i.cache.getCookbookKey(req.Name, version.String())
				if _, exists := i.cache.Get(key); exists {
					// Create resolved cookbook object
					resolvedCookbook := &resolver.ResolvedCookbook{
						Name:      req.Name,
						Version:   version,
						Source:    src.GetSourceLocation(),
						SourceRef: src,
					}
					return resolvedCookbook
				}
			}
		}
	}

	return nil
}

// GetCacheStats returns cache statistics
func (i *Installer) GetCacheStats() *CacheStats {
	return i.cache.Stats()
}

// CleanupCache performs cache cleanup
func (i *Installer) CleanupCache(ctx context.Context) error {
	return i.cache.Cleanup(ctx)
}
