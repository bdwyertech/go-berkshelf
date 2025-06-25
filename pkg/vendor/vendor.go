package vendor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyer/go-berkshelf/pkg/lockfile"
	"github.com/bdwyer/go-berkshelf/pkg/source"
)

// Options configures the vendor operation
type Options struct {
	// TargetPath is the directory to vendor cookbooks to
	TargetPath string
	// Delete existing directory before vendoring
	Delete bool
	// DryRun shows what would be done without doing it
	DryRun bool
	// OnlyCookbooks is a list of cookbook names to vendor (if empty, all cookbooks are vendored)
	OnlyCookbooks []string
}

// Result contains the result of a vendor operation
type Result struct {
	// TotalCookbooks is the number of cookbooks vendored
	TotalCookbooks int
	// SuccessfulDownloads is the number of successful downloads
	SuccessfulDownloads int
	// FailedDownloads is the list of failed cookbook downloads
	FailedDownloads []string
	// TargetPath is the absolute path where cookbooks were vendored
	TargetPath string
}

// Vendorer handles cookbook vendoring operations
type Vendorer struct {
	lockFile      *lockfile.LockFile
	sourceManager *source.Manager
	options       Options
}

// New creates a new Vendorer
func New(lockFile *lockfile.LockFile, sourceManager *source.Manager, options Options) *Vendorer {
	return &Vendorer{
		lockFile:      lockFile,
		sourceManager: sourceManager,
		options:       options,
	}
}

// Vendor downloads all cookbooks from the lock file to the target directory
func (v *Vendorer) Vendor(ctx context.Context) (*Result, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(v.options.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target path: %w", err)
	}

	result := &Result{
		TargetPath: absPath,
	}

	// Create allowed set for filtering
	allowedCookbooks := make(map[string]bool)
	if len(v.options.OnlyCookbooks) > 0 {
		for _, name := range v.options.OnlyCookbooks {
			allowedCookbooks[name] = true
		}
	}

	// Count total cookbooks considering filter
	if len(v.options.OnlyCookbooks) > 0 {
		result.TotalCookbooks = len(v.options.OnlyCookbooks)
	} else {
		for _, source := range v.lockFile.Sources {
			result.TotalCookbooks += len(source.Cookbooks)
		}
	}

	// Delete target directory if requested
	if v.options.Delete && !v.options.DryRun {
		if err := os.RemoveAll(absPath); err != nil {
			return nil, fmt.Errorf("failed to delete target directory: %w", err)
		}
	}

	// Create target directory
	if !v.options.DryRun {
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	// Download each cookbook from lock file
	for _, lockSource := range v.lockFile.Sources {
		for cookbookName, lockedCookbook := range lockSource.Cookbooks {
			// Skip if filtering is active and cookbook not in allowed list
			if len(allowedCookbooks) > 0 && !allowedCookbooks[cookbookName] {
				continue
			}

			if v.options.DryRun {
				result.SuccessfulDownloads++
				continue
			}

			// Find the cookbook version
			version, err := berkshelf.NewVersion(lockedCookbook.Version)
			if err != nil {
				result.FailedDownloads = append(result.FailedDownloads, cookbookName)
				continue
			}

			// Create cookbook directory
			cookbookDir := filepath.Join(absPath, cookbookName)
			if err := os.MkdirAll(cookbookDir, 0755); err != nil {
				result.FailedDownloads = append(result.FailedDownloads, cookbookName)
				continue
			}

			// Download cookbook from appropriate source
			if err := v.downloadCookbook(ctx, cookbookName, version, cookbookDir); err != nil {
				result.FailedDownloads = append(result.FailedDownloads, cookbookName)
				continue
			}

			result.SuccessfulDownloads++
		}
	}

	return result, nil
}

// downloadCookbook downloads a specific cookbook version to the target directory
func (v *Vendorer) downloadCookbook(ctx context.Context, cookbookName string, version *berkshelf.Version, targetDir string) error {
	// First try to find the cookbook-specific source from the lock file
	for _, lockSource := range v.lockFile.Sources {
		if lockedCookbook, exists := lockSource.Cookbooks[cookbookName]; exists {
			// Create source from lock file source info
			src, err := v.createSourceFromLockFile(lockedCookbook.Source)
			if err == nil {
				// Fetch cookbook metadata
				cookbook, err := src.FetchCookbook(ctx, cookbookName, version)
				if err == nil {
					log.Infof("Vendoring %s (%s) to %s", cookbook.Name, version, targetDir)
					if err := src.DownloadAndExtractCookbook(ctx, cookbook, targetDir); err == nil {
						return nil
					}
				}
			}
		}
	}

	// Fallback to trying global sources
	var lastErr error
	for _, src := range v.sourceManager.GetSources() {
		// Fetch cookbook metadata
		cookbook, err := src.FetchCookbook(ctx, cookbookName, version)
		if err != nil {
			lastErr = fmt.Errorf("source %s failed: %w", src.Name(), err)
			continue // Try next source
		}
		log.Infof("Vendoring %s (%s) to %s", cookbook.Name, version, targetDir)
		if err := src.DownloadAndExtractCookbook(ctx, cookbook, targetDir); err == nil {
			return nil
		}
		lastErr = fmt.Errorf("source %s download failed: %w", src.Name(), err)
	}

	if lastErr != nil {
		return fmt.Errorf("failed to download cookbook %s: %w", cookbookName, lastErr)
	}
	return fmt.Errorf("failed to download cookbook %s from any source", cookbookName)
}

// createSourceFromLockFile creates a source from lock file source info
func (v *Vendorer) createSourceFromLockFile(sourceInfo *lockfile.SourceInfo) (source.CookbookSource, error) {
	if sourceInfo == nil {
		return nil, fmt.Errorf("no source info provided")
	}

	// Convert SourceInfo to berkshelf.SourceLocation
	sourceLocation := &berkshelf.SourceLocation{
		Type:    sourceInfo.Type,
		URL:     sourceInfo.URL,
		Path:    sourceInfo.Path,
		Ref:     sourceInfo.Ref,
		Options: make(map[string]any),
	}

	// Add Git options if present
	if sourceInfo.Branch != "" {
		sourceLocation.Options["branch"] = sourceInfo.Branch
	}
	if sourceInfo.Tag != "" {
		sourceLocation.Options["tag"] = sourceInfo.Tag
	}

	// Create source using factory
	factory := source.NewFactory()
	return factory.CreateFromLocation(sourceLocation)
}
