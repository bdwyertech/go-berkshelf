package berksfile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bdwyertech/go-berkshelf/pkg/template"
)

// Load loads and parses a Policyfile.rb from the given path
func Load(filepath string) (*Berksfile, error) {
	data, err := template.Render(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Berksfile: %w", err)
	}

	return Parse(data)
}

// Find searches for a Policyfile.rb in the given directory and parent directories
func Find(startDir string) (string, error) {
	dir := startDir
	for {
		policyfilePath := filepath.Join(dir, "Berksfile")
		if _, err := os.Stat(policyfilePath); err == nil {
			return policyfilePath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("Berksfile not found in %s or any parent directory", startDir)
}

// FilterCookbooksByGroup filters cookbooks based on --only and --except flags
func FilterCookbooksByGroup(cookbooks []*CookbookDef, only []string, except []string) []*CookbookDef {
	if len(only) == 0 && len(except) == 0 {
		return cookbooks
	}

	var filtered []*CookbookDef

	// Create maps for faster lookup
	onlyMap := make(map[string]bool)
	for _, group := range only {
		onlyMap[group] = true
	}

	exceptMap := make(map[string]bool)
	for _, group := range except {
		exceptMap[group] = true
	}

	for _, cookbook := range cookbooks {
		include := true

		// If --only is specified, cookbook must be in at least one of those groups
		if len(only) > 0 {
			include = false
			for _, group := range cookbook.Groups {
				if onlyMap[group] {
					include = true
					break
				}
			}
		}

		// If --except is specified, cookbook must not be in any of those groups
		if len(except) > 0 && include {
			for _, group := range cookbook.Groups {
				if exceptMap[group] {
					include = false
					break
				}
			}
		}

		if include {
			filtered = append(filtered, cookbook)
		}
	}

	return filtered
}

// FindCookbooksByNames finds cookbooks by their names from the list
func FindCookbooksByNames(cookbooks []*CookbookDef, names []string) ([]*CookbookDef, []string) {
	requestedSet := make(map[string]bool)
	for _, name := range names {
		requestedSet[name] = true
	}

	var found []*CookbookDef
	foundSet := make(map[string]bool)

	for _, cookbook := range cookbooks {
		if requestedSet[cookbook.Name] {
			found = append(found, cookbook)
			foundSet[cookbook.Name] = true
		}
	}

	// Find missing cookbooks
	var missing []string
	for _, name := range names {
		if !foundSet[name] {
			missing = append(missing, name)
		}
	}

	return found, missing
}

// ExtractDirectDependencies extracts the direct dependencies from a Berksfile
func (b *Berksfile) ExtractDirectDependencies(groups []string) ([]string, error) {
	// Get cookbooks, optionally filtered by groups
	var cookbooks []*CookbookDef
	if len(groups) > 0 {
		cookbooks = b.GetCookbooks(groups...)
	} else {
		cookbooks = b.GetCookbooks()
	}

	// Extract cookbook names
	var dependencies []string
	for _, cookbook := range cookbooks {
		dependencies = append(dependencies, cookbook.Name)
	}

	// Sort dependencies for consistent output
	for i := 0; i < len(dependencies); i++ {
		for j := i + 1; j < len(dependencies); j++ {
			if dependencies[i] > dependencies[j] {
				dependencies[i], dependencies[j] = dependencies[j], dependencies[i]
			}
		}
	}

	return dependencies, nil
}
