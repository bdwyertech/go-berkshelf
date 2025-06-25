package vendor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	// Count total cookbooks
	for _, source := range v.lockFile.Sources {
		result.TotalCookbooks += len(source.Cookbooks)
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
	// Try each source until one succeeds
	var lastErr error
	for _, src := range v.sourceManager.GetSources() {
		// Fetch cookbook metadata
		cookbook, err := src.FetchCookbook(ctx, cookbookName, version)
		if err != nil {
			lastErr = fmt.Errorf("source %s failed: %w", src.Name(), err)
			continue // Try next source
		}

		// Check if this is a supermarket source with tarball URL
		if cookbook.TarballURL != "" {
			if err := downloadAndExtractTarball(ctx, cookbook.TarballURL, targetDir); err == nil {
				return nil
			}
			lastErr = fmt.Errorf("tarball download failed: %w", err)
		}
	}

	if lastErr != nil {
		return fmt.Errorf("failed to download cookbook %s: %w", cookbookName, lastErr)
	}
	return fmt.Errorf("failed to download cookbook %s from any source", cookbookName)
}

// downloadAndExtractTarball downloads and extracts a cookbook tarball
func downloadAndExtractTarball(ctx context.Context, tarballURL, targetDir string) error {
	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", tarballURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Download the tarball
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download tarball: HTTP %d", resp.StatusCode)
	}

	// Create a gzip reader
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Skip directories and non-regular files
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Remove the top-level directory from the path
		// Supermarket tarballs have a structure like "cookbook-name-version/recipes/default.rb"
		pathParts := strings.Split(header.Name, "/")
		if len(pathParts) <= 1 {
			continue
		}

		// Join all parts except the first (which is the cookbook directory)
		relativePath := filepath.Join(pathParts[1:]...)
		if relativePath == "" {
			continue
		}

		targetPath := filepath.Join(targetDir, relativePath)

		// Create directory structure
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Create and write the file
		outFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", targetPath, err)
		}

		_, err = io.Copy(outFile, tarReader)
		outFile.Close()
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		// Set file permissions
		if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
			// Continue on permission errors
		}
	}

	return nil
}
