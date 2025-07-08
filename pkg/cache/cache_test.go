package cache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
)

func TestCache_Basic(t *testing.T) {
	// Create temporary cache directory
	tempDir, err := os.MkdirTemp("", "berkshelf-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create cache
	cache, err := NewCache(tempDir, time.Hour, 1024*1024) // 1MB limit
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test Put and Get
	key := "test-key"
	data := []byte("test data")

	err = cache.Put(key, data)
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	retrieved, found := cache.Get(key)
	if !found {
		t.Fatal("Expected to find cached data")
	}

	if string(retrieved) != string(data) {
		t.Errorf("Expected %s, got %s", string(data), string(retrieved))
	}

	// Test cache stats
	stats := cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 0 {
		t.Errorf("Expected 0 misses, got %d", stats.Misses)
	}
}

func TestCache_Miss(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "berkshelf-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewCache(tempDir, time.Hour, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test cache miss
	_, found := cache.Get("nonexistent-key")
	if found {
		t.Error("Expected cache miss for nonexistent key")
	}

	stats := cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
}

func TestCache_Expiration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "berkshelf-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create cache with very short expiration
	cache, err := NewCache(tempDir, 10*time.Millisecond, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	key := "test-key"
	data := []byte("test data")

	// Put data
	err = cache.Put(key, data)
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	// Should be available immediately
	retrieved, found := cache.Get(key)
	if !found {
		t.Error("Expected to find cached data immediately")
	} else if string(retrieved) != string(data) {
		t.Errorf("Expected %s, got %s", string(data), string(retrieved))
	}

	// Wait for expiration
	time.Sleep(50 * time.Millisecond) // Increased wait time

	// Should be expired now
	_, found = cache.Get(key)
	if found {
		t.Error("Expected cached data to be expired")
	}
}

func TestCache_Cookbook(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "berkshelf-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewCache(tempDir, time.Hour, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create a test cookbook
	version, _ := berkshelf.NewVersion("1.2.3")
	cookbook := &berkshelf.Cookbook{
		Name:    "nginx",
		Version: version,
	}

	data := []byte("cookbook tarball data")

	// Test cookbook-specific methods
	err = cache.PutCookbook(cookbook, data)
	if err != nil {
		t.Fatalf("Failed to put cookbook: %v", err)
	}

	retrieved, found := cache.GetCookbook("nginx", "1.2.3")
	if !found {
		t.Fatal("Expected to find cached cookbook")
	}

	if string(retrieved) != string(data) {
		t.Errorf("Expected %s, got %s", string(data), string(retrieved))
	}
}

func TestCache_Delete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "berkshelf-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewCache(tempDir, time.Hour, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	key := "test-key"
	data := []byte("test data")

	// Put and verify
	err = cache.Put(key, data)
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	_, found := cache.Get(key)
	if !found {
		t.Fatal("Expected to find cached data")
	}

	// Delete and verify
	err = cache.Delete(key)
	if err != nil {
		t.Fatalf("Failed to delete data: %v", err)
	}

	_, found = cache.Get(key)
	if found {
		t.Error("Expected data to be deleted")
	}
}

func TestCache_Clear(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "berkshelf-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewCache(tempDir, time.Hour, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Put multiple items
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		data := []byte(fmt.Sprintf("data-%d", i))
		err = cache.Put(key, data)
		if err != nil {
			t.Fatalf("Failed to put data %d: %v", i, err)
		}
	}

	// Verify items exist
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		_, found := cache.Get(key)
		if !found {
			t.Errorf("Expected to find key-%d", i)
		}
	}

	// Clear cache
	err = cache.Clear()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Verify items are gone
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		_, found := cache.Get(key)
		if found {
			t.Errorf("Expected key-%d to be cleared", i)
		}
	}

	// Verify size is reset
	if cache.Size() != 0 {
		t.Errorf("Expected cache size to be 0, got %d", cache.Size())
	}
}

func TestCache_SizeLimit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "berkshelf-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create cache with small size limit
	cache, err := NewCache(tempDir, time.Hour, 100) // 100 bytes limit
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Put data that exceeds the limit
	largeData := make([]byte, 150) // 150 bytes
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err = cache.Put("large-key", largeData)
	if err != nil {
		t.Fatalf("Failed to put large data: %v", err)
	}

	// Put more data to trigger eviction
	smallData := []byte("small data")
	err = cache.Put("small-key", smallData)
	if err != nil {
		t.Fatalf("Failed to put small data: %v", err)
	}

	// The cache should have evicted some data to stay under the limit
	if cache.Size() > 100 {
		t.Errorf("Expected cache size <= 100, got %d", cache.Size())
	}
}

func TestCache_Cleanup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "berkshelf-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create cache with short expiration
	cache, err := NewCache(tempDir, 10*time.Millisecond, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Put some data
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("key-%d", i)
		data := []byte(fmt.Sprintf("data-%d", i))
		err = cache.Put(key, data)
		if err != nil {
			t.Fatalf("Failed to put data %d: %v", i, err)
		}
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Run cleanup
	ctx := context.Background()
	err = cache.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Failed to cleanup cache: %v", err)
	}

	// Verify expired items are removed
	stats := cache.Stats()
	if stats.Evictions == 0 {
		t.Error("Expected some evictions during cleanup")
	}
}

func TestCache_HitRate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "berkshelf-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewCache(tempDir, time.Hour, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Initial hit rate should be 0
	if cache.HitRate() != 0 {
		t.Errorf("Expected initial hit rate to be 0, got %f", cache.HitRate())
	}

	// Put some data
	key := "test-key"
	data := []byte("test data")
	err = cache.Put(key, data)
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	// Generate some hits and misses
	cache.Get(key)      // hit
	cache.Get(key)      // hit
	cache.Get("miss-1") // miss
	cache.Get("miss-2") // miss

	// Hit rate should be 50% (2 hits, 2 misses)
	hitRate := cache.HitRate()
	if hitRate != 50.0 {
		t.Errorf("Expected hit rate to be 50.0, got %f", hitRate)
	}
}

func TestCache_ChecksumValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "berkshelf-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewCache(tempDir, time.Hour, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	key := "test-key"
	data := []byte("test data")

	// Put data
	err = cache.Put(key, data)
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	// Manually corrupt the cached file
	entry, exists := cache.getEntry(key)
	if !exists {
		t.Fatal("Expected cache entry to exist")
	}

	corruptedData := []byte("corrupted data")
	err = os.WriteFile(entry.Path, corruptedData, 0644)
	if err != nil {
		t.Fatalf("Failed to corrupt cache file: %v", err)
	}

	// Try to get the data - should fail checksum validation
	_, found := cache.Get(key)
	if found {
		t.Error("Expected checksum validation to fail for corrupted data")
	}

	// Should record a miss
	stats := cache.Stats()
	if stats.Misses == 0 {
		t.Error("Expected a cache miss due to checksum validation failure")
	}
}
