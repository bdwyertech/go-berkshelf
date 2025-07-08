package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"

	"github.com/bdwyer/go-berkshelf/internal/config"
	"github.com/bdwyer/go-berkshelf/pkg/cache"
)

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cacheInfoCmd)
	cacheCmd.AddCommand(cacheClearCmd)

	// Add flags for cache list command
	cacheListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
	cacheListCmd.Flags().BoolP("verbose", "v", false, "Show detailed information")

	// Add flags for cache clean command
	cacheCleanCmd.Flags().BoolP("dry-run", "n", false, "Show what would be cleaned without actually cleaning")
	cacheCleanCmd.Flags().StringP("max-age", "a", "", "Clean entries older than specified duration (e.g., 30d, 1w)")

	// Add flags for cache info command
	cacheInfoCmd.Flags().StringP("format", "f", "text", "Output format (text, json)")

	// Add flags for cache clear command
	cacheClearCmd.Flags().BoolP("force", "f", false, "Force clear without confirmation")
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage cookbook cache",
	Long: `Manage the local cookbook cache with subcommands for listing, cleaning, and inspecting cached cookbooks.

The cache stores downloaded cookbooks locally to improve performance on subsequent operations.
Cache location can be configured via BERKSHELF_CACHE_PATH environment variable.`,
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached cookbooks",
	Long: `List all cookbooks currently stored in the cache.

Shows cookbook names, versions, sizes, and access information.`,
	RunE: runCacheList,
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean cookbook cache",
	Long: `Clean expired or old entries from the cookbook cache.

By default, removes entries based on the configured cache expiration policy.
Use --max-age to specify a custom age threshold.`,
	RunE: runCacheClean,
}

var cacheInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show cache information",
	Long: `Display detailed information about the cookbook cache including:
- Cache location and size
- Hit/miss statistics
- Number of cached cookbooks
- Last cleanup time`,
	RunE: runCacheInfo,
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cached cookbooks",
	Long: `Remove all cookbooks from the cache.

This will force all cookbooks to be re-downloaded on the next install operation.
Use with caution in CI/CD environments.`,
	RunE: runCacheClear,
}

func runCacheList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize cache
	cachePath := cfg.GetCachePathResolved()
	cacheInstance, err := cache.NewCache(cachePath, 24*time.Hour, 1024*1024*1024) // 1GB default
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Get cache entries
	entries, err := getCacheEntries(cachePath)
	if err != nil {
		return fmt.Errorf("failed to list cache entries: %w", err)
	}

	format, _ := cmd.Flags().GetString("format")
	verbose, _ := cmd.Flags().GetBool("verbose")

	switch format {
	case "json":
		return outputCacheEntriesJSON(entries)
	case "table":
		return outputCacheEntriesTable(entries, verbose)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func runCacheClean(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize cache
	cachePath := cfg.GetCachePathResolved()
	cacheInstance, err := cache.NewCache(cachePath, 24*time.Hour, 1024*1024*1024) // 1GB default
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	maxAgeStr, _ := cmd.Flags().GetString("max-age")

	// Parse max-age if provided
	var maxAge time.Duration
	if maxAgeStr != "" {
		maxAge, err = parseDuration(maxAgeStr)
		if err != nil {
			return fmt.Errorf("invalid max-age format: %w", err)
		}
	}

	if dryRun {
		log.Info("Dry run mode - showing what would be cleaned")
		return showCleanupPreview(cacheInstance, maxAge)
	}

	// Perform cleanup
	ctx := context.Background()
	if err := cacheInstance.Cleanup(ctx); err != nil {
		return fmt.Errorf("failed to clean cache: %w", err)
	}

	stats := cacheInstance.Stats()
	log.Infof("Cache cleanup completed")
	log.Infof("Evicted %d entries", stats.Evictions)
	log.Infof("Current cache size: %s", formatBytes(stats.TotalSize))

	return nil
}

func runCacheInfo(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize cache
	cachePath := cfg.GetCachePathResolved()
	cacheInstance, err := cache.NewCache(cachePath, 24*time.Hour, 1024*1024*1024) // 1GB default
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	format, _ := cmd.Flags().GetString("format")
	stats := cacheInstance.Stats()

	// Count entries
	entries, err := getCacheEntries(cachePath)
	if err != nil {
		return fmt.Errorf("failed to count cache entries: %w", err)
	}

	info := map[string]interface{}{
		"cache_path":     cachePath,
		"total_size":     stats.TotalSize,
		"entry_count":    len(entries),
		"hits":           stats.Hits,
		"misses":         stats.Misses,
		"evictions":      stats.Evictions,
		"hit_rate":       cacheInstance.HitRate(),
		"last_cleanup":   stats.LastCleanup,
	}

	switch format {
	case "json":
		return outputJSON(info)
	case "text":
		return outputCacheInfoText(info)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	if !force {
		fmt.Print("This will remove all cached cookbooks. Are you sure? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" {
			log.Info("Cache clear cancelled")
			return nil
		}
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize cache
	cachePath := cfg.GetCachePathResolved()
	cacheInstance, err := cache.NewCache(cachePath, 24*time.Hour, 1024*1024*1024) // 1GB default
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Clear cache
	if err := cacheInstance.Clear(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	log.Info("Cache cleared successfully")
	return nil
}

// Helper functions

func getCacheEntries(cachePath string) ([]*cache.CacheEntry, error) {
	var entries []*cache.CacheEntry

	err := filepath.Walk(cachePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".meta" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil // Skip corrupted metadata
			}

			var entry cache.CacheEntry
			if err := json.Unmarshal(data, &entry); err != nil {
				return nil // Skip corrupted metadata
			}

			entries = append(entries, &entry)
		}

		return nil
	})

	return entries, err
}

func outputCacheEntriesJSON(entries []*cache.CacheEntry) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

func outputCacheEntriesTable(entries []*cache.CacheEntry, verbose bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	if verbose {
		fmt.Fprintln(w, "KEY\tSIZE\tCREATED\tACCESSED\tACCESS COUNT")
		for _, entry := range entries {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
				entry.Key,
				formatBytes(entry.Size),
				entry.CreatedAt.Format("2006-01-02 15:04:05"),
				entry.AccessedAt.Format("2006-01-02 15:04:05"),
				entry.AccessCount,
			)
		}
	} else {
		fmt.Fprintln(w, "KEY\tSIZE\tCREATED")
		for _, entry := range entries {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				entry.Key,
				formatBytes(entry.Size),
				entry.CreatedAt.Format("2006-01-02 15:04:05"),
			)
		}
	}

	return nil
}

func outputCacheInfoText(info map[string]interface{}) error {
	fmt.Printf("Cache Information:\n")
	fmt.Printf("  Location: %s\n", info["cache_path"])
	fmt.Printf("  Total Size: %s\n", formatBytes(info["total_size"].(int64)))
	fmt.Printf("  Entry Count: %d\n", info["entry_count"])
	fmt.Printf("  Cache Hits: %d\n", info["hits"])
	fmt.Printf("  Cache Misses: %d\n", info["misses"])
	fmt.Printf("  Evictions: %d\n", info["evictions"])
	fmt.Printf("  Hit Rate: %.2f%%\n", info["hit_rate"])
	
	if lastCleanup, ok := info["last_cleanup"].(time.Time); ok && !lastCleanup.IsZero() {
		fmt.Printf("  Last Cleanup: %s\n", lastCleanup.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("  Last Cleanup: Never\n")
	}

	return nil
}

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func showCleanupPreview(cache *cache.Cache, maxAge time.Duration) error {
	// This would need to be implemented in the cache package
	// For now, just show current stats
	stats := cache.Stats()
	fmt.Printf("Current cache statistics:\n")
	fmt.Printf("  Total Size: %s\n", formatBytes(stats.TotalSize))
	fmt.Printf("  Hits: %d\n", stats.Hits)
	fmt.Printf("  Misses: %d\n", stats.Misses)
	fmt.Printf("  Evictions: %d\n", stats.Evictions)
	
	return nil
}

func parseDuration(s string) (time.Duration, error) {
	// Handle common suffixes like "30d", "1w", etc.
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	suffix := s[len(s)-1:]
	valueStr := s[:len(s)-1]

	var multiplier time.Duration
	switch suffix {
	case "d":
		multiplier = 24 * time.Hour
	case "w":
		multiplier = 7 * 24 * time.Hour
	case "h":
		multiplier = time.Hour
	case "m":
		multiplier = time.Minute
	default:
		// Try parsing as standard duration
		return time.ParseDuration(s)
	}

	value, err := time.ParseDuration(valueStr + "h")
	if err != nil {
		return 0, err
	}

	return time.Duration(value.Hours()) * multiplier, nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
