package outdated

import (
	"context"
	"fmt"
	"sort"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyertech/go-berkshelf/pkg/lockfile"
	"github.com/bdwyertech/go-berkshelf/pkg/source"
)

// Cookbook represents an outdated cookbook
type Cookbook struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	Source         string `json:"source"`
}

// Checker checks for outdated cookbooks
type Checker struct {
	lockFile      *lockfile.LockFile
	sourceManager *source.Manager
}

// New creates a new outdated checker
func New(lockFile *lockfile.LockFile, sourceManager *source.Manager) *Checker {
	return &Checker{
		lockFile:      lockFile,
		sourceManager: sourceManager,
	}
}

// Check checks for outdated cookbooks
// If cookbookNames is empty, all cookbooks from the lock file are checked
func (c *Checker) Check(ctx context.Context, cookbookNames []string) ([]Cookbook, error) {
	var outdatedCookbooks []Cookbook

	// Build map of cookbooks to check
	cookbooksToCheck := make(map[string]bool)
	if len(cookbookNames) > 0 {
		for _, name := range cookbookNames {
			cookbooksToCheck[name] = true
		}
	} else {
		// Check all cookbooks from lock file
		for _, source := range c.lockFile.Sources {
			for cookbookName := range source.Cookbooks {
				cookbooksToCheck[cookbookName] = true
			}
		}
	}

	// Check each cookbook
	for cookbookName := range cookbooksToCheck {
		outdated, err := c.checkCookbook(ctx, cookbookName)
		if err != nil {
			// Skip cookbooks with errors
			continue
		}
		if outdated != nil {
			outdatedCookbooks = append(outdatedCookbooks, *outdated)
		}
	}

	// Sort by cookbook name
	sort.Slice(outdatedCookbooks, func(i, j int) bool {
		return outdatedCookbooks[i].Name < outdatedCookbooks[j].Name
	})

	return outdatedCookbooks, nil
}

// checkCookbook checks if a single cookbook is outdated
func (c *Checker) checkCookbook(ctx context.Context, cookbookName string) (*Cookbook, error) {
	// Find current version in lock file
	var currentVersion, sourceURL string
	var found bool

	for _, source := range c.lockFile.Sources {
		if lockedCookbook, exists := source.Cookbooks[cookbookName]; exists {
			currentVersion = lockedCookbook.Version
			sourceURL = source.URL
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("cookbook %s not found in lock file", cookbookName)
	}

	// Get latest version from sources
	latestVersion, err := c.getLatestVersion(ctx, cookbookName)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version for %s: %w", cookbookName, err)
	}

	// Compare versions
	current, err := berkshelf.NewVersion(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid current version for %s: %w", cookbookName, err)
	}

	latest, err := berkshelf.NewVersion(latestVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid latest version for %s: %w", cookbookName, err)
	}

	// Check if outdated
	if latest.GreaterThan(current) {
		return &Cookbook{
			Name:           cookbookName,
			CurrentVersion: currentVersion,
			LatestVersion:  latestVersion,
			Source:         sourceURL,
		}, nil
	}

	return nil, nil
}

// getLatestVersion gets the latest version of a cookbook from available sources
func (c *Checker) getLatestVersion(ctx context.Context, cookbookName string) (string, error) {
	for _, src := range c.sourceManager.GetSources() {
		versions, err := src.ListVersions(ctx, cookbookName)
		if err != nil {
			continue // Try next source
		}

		if len(versions) > 0 {
			// Versions should be sorted with latest first
			return versions[0].String(), nil
		}
	}

	return "", fmt.Errorf("no versions found for cookbook %s", cookbookName)
}
