package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

// FilesystemCache implements Cache using the local filesystem
type FilesystemCache struct {
	basePath string
}

// NewFilesystemCache creates a new filesystem-based cache
func NewFilesystemCache(basePath string) *FilesystemCache {
	return &FilesystemCache{
		basePath: basePath,
	}
}

// Get retrieves a cookbook from the cache
func (c *FilesystemCache) Get(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Cookbook, error) {
	cookbookPath := c.getCookbookPath(name, version)
	metadataPath := filepath.Join(cookbookPath, "metadata.json")

	// Check if metadata file exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cookbook %s (%s) not found in cache", name, version.String())
	}

	// Read metadata
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached metadata for %s (%s): %w", name, version.String(), err)
	}

	var cookbook berkshelf.Cookbook
	if err := json.Unmarshal(data, &cookbook); err != nil {
		return nil, fmt.Errorf("failed to parse cached metadata for %s (%s): %w", name, version.String(), err)
	}

	// Set the path to the cached cookbook
	cookbook.Path = cookbookPath

	return &cookbook, nil
}

// Put stores a cookbook in the cache
func (c *FilesystemCache) Put(ctx context.Context, cookbook *berkshelf.Cookbook) error {
	if cookbook == nil {
		return fmt.Errorf("cookbook cannot be nil")
	}

	cookbookPath := c.getCookbookPath(cookbook.Name, cookbook.Version)

	// Create directory structure
	if err := os.MkdirAll(cookbookPath, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory for %s: %w", cookbook.Name, err)
	}

	// Write metadata
	metadataPath := filepath.Join(cookbookPath, "metadata.json")
	data, err := json.MarshalIndent(cookbook, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata for %s: %w", cookbook.Name, err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cached metadata for %s: %w", cookbook.Name, err)
	}

	return nil
}

// Has checks if a cookbook exists in the cache
func (c *FilesystemCache) Has(ctx context.Context, name string, version *berkshelf.Version) bool {
	cookbookPath := c.getCookbookPath(name, version)
	metadataPath := filepath.Join(cookbookPath, "metadata.json")

	_, err := os.Stat(metadataPath)
	return err == nil
}

// Delete removes a cookbook from the cache
func (c *FilesystemCache) Delete(ctx context.Context, name string, version *berkshelf.Version) error {
	cookbookPath := c.getCookbookPath(name, version)

	if err := os.RemoveAll(cookbookPath); err != nil {
		return fmt.Errorf("failed to delete cached cookbook %s (%s): %w", name, version.String(), err)
	}

	return nil
}

// Clear removes all cookbooks from the cache
func (c *FilesystemCache) Clear(ctx context.Context) error {
	if err := os.RemoveAll(c.basePath); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	// Recreate the base directory
	if err := os.MkdirAll(c.basePath, 0755); err != nil {
		return fmt.Errorf("failed to recreate cache directory: %w", err)
	}

	return nil
}

// List returns all cached cookbooks
func (c *FilesystemCache) List(ctx context.Context) ([]*CacheEntry, error) {
	var entries []*CacheEntry

	// Walk through the cache directory
	err := filepath.WalkDir(c.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Look for metadata.json files
		if d.Name() == "metadata.json" {
			entry, err := c.createCacheEntry(path)
			if err != nil {
				// Log error but continue walking
				return nil
			}
			entries = append(entries, entry)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list cache entries: %w", err)
	}

	return entries, nil
}

// Path returns the cache directory path
func (c *FilesystemCache) Path() string {
	return c.basePath
}

// Size returns the cache size in bytes
func (c *FilesystemCache) Size(ctx context.Context) (int64, error) {
	var size int64

	err := filepath.WalkDir(c.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate cache size: %w", err)
	}

	return size, nil
}

// getCookbookPath returns the filesystem path for a cached cookbook
func (c *FilesystemCache) getCookbookPath(name string, version *berkshelf.Version) string {
	// Use name-version format for directory
	dirName := fmt.Sprintf("%s-%s", name, version.String())
	return filepath.Join(c.basePath, dirName)
}

// createCacheEntry creates a CacheEntry from a metadata.json file
func (c *FilesystemCache) createCacheEntry(metadataPath string) (*CacheEntry, error) {
	// Read the metadata file
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}

	var cookbook berkshelf.Cookbook
	if err := json.Unmarshal(data, &cookbook); err != nil {
		return nil, err
	}

	// Get file info
	info, err := os.Stat(metadataPath)
	if err != nil {
		return nil, err
	}

	// Get cookbook directory path
	cookbookPath := filepath.Dir(metadataPath)

	// Calculate directory size
	var size int64
	filepath.WalkDir(cookbookPath, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if fileInfo, err := d.Info(); err == nil {
				size += fileInfo.Size()
			}
		}
		return nil
	})

	return &CacheEntry{
		Name:     cookbook.Name,
		Version:  cookbook.Version,
		Path:     cookbookPath,
		Size:     size,
		ModTime:  info.ModTime().Unix(),
		Cookbook: &cookbook,
	}, nil
}
