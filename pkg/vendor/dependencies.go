package vendor

import (
	"maps"

	"github.com/bdwyertech/go-berkshelf/pkg/lockfile"
)

// FindTransitiveDependencies finds all transitive dependencies for given cookbooks
// using breadth-first search traversal of the dependency graph from the lock file.
func FindTransitiveDependencies(lockFile *lockfile.LockFile, cookbookNames []string) []string {
	// Create a map of all cookbooks for quick lookup
	allCookbooks := make(map[string]*lockfile.CookbookLock)
	for _, source := range lockFile.Sources {
		maps.Copy(allCookbooks, source.Cookbooks)
	}

	// Set to track all required cookbooks (prevents duplicates)
	required := make(map[string]bool)

	// Queue for BFS traversal
	queue := make([]string, 0, len(cookbookNames))

	// Initialize with requested cookbooks
	for _, name := range cookbookNames {
		if !required[name] {
			required[name] = true
			queue = append(queue, name)
		}
	}

	// BFS to find all transitive dependencies
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Find the cookbook in the lock file
		if cookbook, ok := allCookbooks[current]; ok {
			// Add all dependencies to the queue if not already processed
			// and if the dependency exists in the lock file
			for depName := range cookbook.Dependencies {
				if !required[depName] && allCookbooks[depName] != nil {
					required[depName] = true
					queue = append(queue, depName)
				}
			}
		}
	}

	// Convert set to slice
	result := make([]string, 0, len(required))
	for name := range required {
		result = append(result, name)
	}

	return result
}
