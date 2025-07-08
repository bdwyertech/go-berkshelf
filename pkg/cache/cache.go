package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyer/go-berkshelf/pkg/errors"
)

// Cache provides advanced caching capabilities
type Cache struct {
	basePath    string
	maxAge      time.Duration
	maxSize     int64 // Maximum cache size in bytes
	currentSize int64
	mu          sync.RWMutex
	stats       *CacheStats
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	Evictions   int64     `json:"evictions"`
	TotalSize   int64     `json:"total_size"`
	LastCleanup time.Time `json:"last_cleanup"`
	mu          sync.RWMutex
}

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Key         string    `json:"key"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	AccessedAt  time.Time `json:"accessed_at"`
	AccessCount int64     `json:"access_count"`
	Checksum    string    `json:"checksum"`
}

// NewCache creates a new cache
func NewCache(basePath string, maxAge time.Duration, maxSize int64) (*Cache, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, errors.NewFileSystemError("failed to create cache directory", err)
	}

	cache := &Cache{
		basePath: basePath,
		maxAge:   maxAge,
		maxSize:  maxSize,
		stats:    &CacheStats{},
	}

	// Initialize cache size
	if err := cache.calculateSize(); err != nil {
		return nil, err
	}

	return cache, nil
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.getEntry(key)
	if !exists {
		c.stats.recordMiss()
		return nil, false
	}

	// Check if entry is expired
	if c.isExpired(entry) {
		c.stats.recordMiss()
		go c.removeEntry(key) // Async cleanup
		return nil, false
	}

	// Read the cached data
	data, err := os.ReadFile(entry.Path)
	if err != nil {
		c.stats.recordMiss()
		go c.removeEntry(key) // Async cleanup
		return nil, false
	}

	// Verify checksum
	if !c.verifyChecksum(data, entry.Checksum) {
		c.stats.recordMiss()
		go c.removeEntry(key) // Async cleanup
		return nil, false
	}

	// Update access statistics
	c.updateAccess(entry)
	c.stats.recordHit()

	return data, true
}

// Put stores an item in the cache
func (c *Cache) Put(key string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Calculate checksum
	checksum := c.calculateChecksum(data)

	// Create cache entry
	entry := &CacheEntry{
		Key:         key,
		Path:        c.getPath(key),
		Size:        int64(len(data)),
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
		Checksum:    checksum,
	}

	// Ensure we have space
	if err := c.ensureSpace(entry.Size); err != nil {
		return err
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(entry.Path), 0755); err != nil {
		return errors.NewFileSystemError("failed to create cache directory", err)
	}

	// Write data to cache
	if err := os.WriteFile(entry.Path, data, 0644); err != nil {
		return errors.NewFileSystemError("failed to write cache entry", err)
	}

	// Write metadata
	if err := c.writeEntry(entry); err != nil {
		os.Remove(entry.Path) // Cleanup on failure
		return err
	}

	// Update cache size
	c.currentSize += entry.Size

	return nil
}

// PutCookbook stores a cookbook in the cache
func (c *Cache) PutCookbook(cookbook *berkshelf.Cookbook, data []byte) error {
	key := c.getCookbookKey(cookbook.Name, cookbook.Version.String())
	return c.Put(key, data)
}

// GetCookbook retrieves a cookbook from the cache
func (c *Cache) GetCookbook(name, version string) ([]byte, bool) {
	key := c.getCookbookKey(name, version)
	return c.Get(key)
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.removeEntry(key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.RemoveAll(c.basePath); err != nil {
		return errors.NewFileSystemError("failed to clear cache", err)
	}

	if err := os.MkdirAll(c.basePath, 0755); err != nil {
		return errors.NewFileSystemError("failed to recreate cache directory", err)
	}

	c.currentSize = 0
	c.stats = &CacheStats{}

	return nil
}

// Cleanup removes expired entries and enforces size limits
func (c *Cache) Cleanup(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := c.getAllEntries()
	if err != nil {
		return err
	}

	var removed int64
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if c.isExpired(entry) {
			if err := c.removeEntry(entry.Key); err == nil {
				removed++
				c.stats.recordEviction()
			}
		}
	}

	// Enforce size limit by removing least recently used entries
	if c.currentSize > c.maxSize {
		if err := c.enforceSizeLimit(); err != nil {
			return err
		}
	}

	c.stats.LastCleanup = time.Now()
	return nil
}

// Stats returns cache statistics
func (c *Cache) Stats() *CacheStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	// Return a copy to prevent modification
	return &CacheStats{
		Hits:        c.stats.Hits,
		Misses:      c.stats.Misses,
		Evictions:   c.stats.Evictions,
		TotalSize:   c.currentSize,
		LastCleanup: c.stats.LastCleanup,
	}
}

// Size returns the current cache size in bytes
func (c *Cache) Size() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentSize
}

// HitRate returns the cache hit rate as a percentage
func (c *Cache) HitRate() float64 {
	stats := c.Stats()
	total := stats.Hits + stats.Misses
	if total == 0 {
		return 0
	}
	return float64(stats.Hits) / float64(total) * 100
}

// Private methods

func (c *Cache) getPath(key string) string {
	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])
	return filepath.Join(c.basePath, hashStr[:2], hashStr[2:4], hashStr)
}

func (c *Cache) getMetadataPath(key string) string {
	return c.getPath(key) + ".meta"
}

func (c *Cache) getCookbookKey(name, version string) string {
	return fmt.Sprintf("cookbook:%s:%s", name, version)
}

func (c *Cache) calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (c *Cache) verifyChecksum(data []byte, expectedChecksum string) bool {
	actualChecksum := c.calculateChecksum(data)
	return actualChecksum == expectedChecksum
}

func (c *Cache) getEntry(key string) (*CacheEntry, bool) {
	metaPath := c.getMetadataPath(key)

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, false
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	return &entry, true
}

func (c *Cache) writeEntry(entry *CacheEntry) error {
	metaPath := c.getMetadataPath(entry.Key)

	data, err := json.Marshal(entry)
	if err != nil {
		return errors.NewFileSystemError("failed to marshal cache entry", err)
	}

	if err := os.WriteFile(metaPath, data, 0644); err != nil {
		return errors.NewFileSystemError("failed to write cache metadata", err)
	}

	return nil
}

func (c *Cache) removeEntry(key string) error {
	entry, exists := c.getEntry(key)
	if !exists {
		return nil
	}

	// Remove data file
	if err := os.Remove(entry.Path); err != nil && !os.IsNotExist(err) {
		return errors.NewFileSystemError("failed to remove cache entry", err)
	}

	// Remove metadata file
	metaPath := c.getMetadataPath(key)
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		return errors.NewFileSystemError("failed to remove cache metadata", err)
	}

	// Update cache size
	c.currentSize -= entry.Size

	return nil
}

func (c *Cache) isExpired(entry *CacheEntry) bool {
	if c.maxAge <= 0 {
		return false
	}
	return time.Since(entry.CreatedAt) > c.maxAge
}

func (c *Cache) updateAccess(entry *CacheEntry) {
	entry.AccessedAt = time.Now()
	entry.AccessCount++
	c.writeEntry(entry) // Update metadata (ignore errors for performance)
}

func (c *Cache) calculateSize() error {
	var totalSize int64

	err := filepath.Walk(c.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) != ".meta" {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return errors.NewFileSystemError("failed to calculate cache size", err)
	}

	c.currentSize = totalSize
	return nil
}

func (c *Cache) ensureSpace(requiredSize int64) error {
	if c.maxSize <= 0 {
		return nil // No size limit
	}

	if c.currentSize+requiredSize <= c.maxSize {
		return nil // Enough space
	}

	return c.enforceSizeLimit()
}

func (c *Cache) enforceSizeLimit() error {
	entries, err := c.getAllEntries()
	if err != nil {
		return err
	}

	// Sort by access time (least recently used first)
	// This is a simplified LRU implementation
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].AccessedAt.After(entries[j].AccessedAt) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove entries until we're under the size limit
	targetSize := c.maxSize * 80 / 100 // Remove to 80% of max size
	for _, entry := range entries {
		if c.currentSize <= targetSize {
			break
		}

		if err := c.removeEntry(entry.Key); err == nil {
			c.stats.recordEviction()
		}
	}

	return nil
}

func (c *Cache) getAllEntries() ([]*CacheEntry, error) {
	var entries []*CacheEntry

	err := filepath.Walk(c.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".meta" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil // Skip corrupted metadata
			}

			var entry CacheEntry
			if err := json.Unmarshal(data, &entry); err != nil {
				return nil // Skip corrupted metadata
			}

			entries = append(entries, &entry)
		}

		return nil
	})

	if err != nil {
		return nil, errors.NewFileSystemError("failed to list cache entries", err)
	}

	return entries, nil
}

// CacheStats methods

func (s *CacheStats) recordHit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Hits++
}

func (s *CacheStats) recordMiss() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Misses++
}

func (s *CacheStats) recordEviction() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Evictions++
}
