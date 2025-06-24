package cache

import (
	"context"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

// Cache defines the interface for cookbook caching
type Cache interface {
	// Get retrieves a cookbook from the cache
	Get(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Cookbook, error)

	// Put stores a cookbook in the cache
	Put(ctx context.Context, cookbook *berkshelf.Cookbook) error

	// Has checks if a cookbook exists in the cache
	Has(ctx context.Context, name string, version *berkshelf.Version) bool

	// Delete removes a cookbook from the cache
	Delete(ctx context.Context, name string, version *berkshelf.Version) error

	// Clear removes all cookbooks from the cache
	Clear(ctx context.Context) error

	// List returns all cached cookbooks
	List(ctx context.Context) ([]*CacheEntry, error)

	// Path returns the cache directory path
	Path() string

	// Size returns the cache size in bytes
	Size(ctx context.Context) (int64, error)
}

// CacheEntry represents a cached cookbook entry
type CacheEntry struct {
	Name     string
	Version  *berkshelf.Version
	Path     string
	Size     int64
	ModTime  int64
	Cookbook *berkshelf.Cookbook
}

// String returns a string representation of the cache entry
func (e *CacheEntry) String() string {
	if e.Version != nil {
		return e.Name + " (" + e.Version.String() + ")"
	}
	return e.Name
}
