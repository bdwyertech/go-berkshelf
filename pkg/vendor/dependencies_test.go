package vendor

import (
	"sort"
	"testing"
	"time"

	"github.com/bdwyertech/go-berkshelf/pkg/lockfile"
)

func TestFindTransitiveDependencies(t *testing.T) {
	tests := []struct {
		name          string
		lockFile      *lockfile.LockFile
		cookbookNames []string
		expectedNames []string
	}{
		{
			name: "single cookbook with no dependencies",
			lockFile: &lockfile.LockFile{
				Revision:    7,
				GeneratedAt: time.Now(),
				Sources: map[string]*lockfile.SourceLock{
					"https://supermarket.chef.io": {
						Type: "chef_supermarket",
						URL:  "https://supermarket.chef.io",
						Cookbooks: map[string]*lockfile.CookbookLock{
							"app": {
								Version:      "1.0.0",
								Dependencies: map[string]string{},
							},
						},
					},
				},
			},
			cookbookNames: []string{"app"},
			expectedNames: []string{"app"},
		},
		{
			name: "cookbook with single dependency",
			lockFile: &lockfile.LockFile{
				Revision:    7,
				GeneratedAt: time.Now(),
				Sources: map[string]*lockfile.SourceLock{
					"https://supermarket.chef.io": {
						Type: "chef_supermarket",
						URL:  "https://supermarket.chef.io",
						Cookbooks: map[string]*lockfile.CookbookLock{
							"app": {
								Version:      "1.0.0",
								Dependencies: map[string]string{"base": ">= 0.0.0"},
							},
							"base": {
								Version:      "2.0.0",
								Dependencies: map[string]string{},
							},
						},
					},
				},
			},
			cookbookNames: []string{"app"},
			expectedNames: []string{"app", "base"},
		},
		{
			name: "cookbook with transitive dependencies",
			lockFile: &lockfile.LockFile{
				Revision:    7,
				GeneratedAt: time.Now(),
				Sources: map[string]*lockfile.SourceLock{
					"https://supermarket.chef.io": {
						Type: "chef_supermarket",
						URL:  "https://supermarket.chef.io",
						Cookbooks: map[string]*lockfile.CookbookLock{
							"app": {
								Version:      "1.0.0",
								Dependencies: map[string]string{"base": ">= 0.0.0"},
							},
							"base": {
								Version:      "2.0.0",
								Dependencies: map[string]string{"utils": ">= 1.0.0"},
							},
							"utils": {
								Version:      "1.5.0",
								Dependencies: map[string]string{},
							},
						},
					},
				},
			},
			cookbookNames: []string{"app"},
			expectedNames: []string{"app", "base", "utils"},
		},
		{
			name: "multiple cookbooks with shared dependencies",
			lockFile: &lockfile.LockFile{
				Revision:    7,
				GeneratedAt: time.Now(),
				Sources: map[string]*lockfile.SourceLock{
					"https://supermarket.chef.io": {
						Type: "chef_supermarket",
						URL:  "https://supermarket.chef.io",
						Cookbooks: map[string]*lockfile.CookbookLock{
							"web": {
								Version:      "1.0.0",
								Dependencies: map[string]string{"base": ">= 0.0.0"},
							},
							"db": {
								Version:      "2.0.0",
								Dependencies: map[string]string{"base": ">= 0.0.0"},
							},
							"base": {
								Version:      "2.0.0",
								Dependencies: map[string]string{},
							},
						},
					},
				},
			},
			cookbookNames: []string{"web", "db"},
			expectedNames: []string{"web", "db", "base"},
		},
		{
			name: "cookbook with missing dependency",
			lockFile: &lockfile.LockFile{
				Revision:    7,
				GeneratedAt: time.Now(),
				Sources: map[string]*lockfile.SourceLock{
					"https://supermarket.chef.io": {
						Type: "chef_supermarket",
						URL:  "https://supermarket.chef.io",
						Cookbooks: map[string]*lockfile.CookbookLock{
							"app": {
								Version:      "1.0.0",
								Dependencies: map[string]string{"missing": ">= 0.0.0"},
							},
						},
					},
				},
			},
			cookbookNames: []string{"app"},
			expectedNames: []string{"app"}, // missing dependency should be ignored
		},
		{
			name: "empty cookbook list",
			lockFile: &lockfile.LockFile{
				Revision:    7,
				GeneratedAt: time.Now(),
				Sources:     map[string]*lockfile.SourceLock{},
			},
			cookbookNames: []string{},
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindTransitiveDependencies(tt.lockFile, tt.cookbookNames)

			// Sort both slices for comparison
			sort.Strings(result)
			sort.Strings(tt.expectedNames)

			if len(result) != len(tt.expectedNames) {
				t.Errorf("FindTransitiveDependencies() returned %d cookbooks, expected %d", len(result), len(tt.expectedNames))
				t.Errorf("Got: %v", result)
				t.Errorf("Expected: %v", tt.expectedNames)
				return
			}

			for i, name := range result {
				if name != tt.expectedNames[i] {
					t.Errorf("FindTransitiveDependencies() mismatch at index %d: got %q, expected %q", i, name, tt.expectedNames[i])
					t.Errorf("Got: %v", result)
					t.Errorf("Expected: %v", tt.expectedNames)
					break
				}
			}
		})
	}
}

func TestFindTransitiveDependenciesCircular(t *testing.T) {
	// Test case with circular dependencies
	lockFile := &lockfile.LockFile{
		Revision:    7,
		GeneratedAt: time.Now(),
		Sources: map[string]*lockfile.SourceLock{
			"https://supermarket.chef.io": {
				Type: "chef_supermarket",
				URL:  "https://supermarket.chef.io",
				Cookbooks: map[string]*lockfile.CookbookLock{
					"app": {
						Version:      "1.0.0",
						Dependencies: map[string]string{"web": ">= 0.0.0"},
					},
					"web": {
						Version:      "2.0.0",
						Dependencies: map[string]string{"db": ">= 0.0.0"},
					},
					"db": {
						Version:      "1.5.0",
						Dependencies: map[string]string{"app": ">= 0.0.0"}, // circular dependency
					},
				},
			},
		},
	}

	result := FindTransitiveDependencies(lockFile, []string{"app"})
	expectedNames := []string{"app", "web", "db"}

	// Sort both slices for comparison
	sort.Strings(result)
	sort.Strings(expectedNames)

	if len(result) != len(expectedNames) {
		t.Errorf("FindTransitiveDependencies() with circular dependencies returned %d cookbooks, expected %d", len(result), len(expectedNames))
		t.Errorf("Got: %v", result)
		t.Errorf("Expected: %v", expectedNames)
		return
	}

	for i, name := range result {
		if name != expectedNames[i] {
			t.Errorf("FindTransitiveDependencies() circular dependency mismatch at index %d: got %q, expected %q", i, name, expectedNames[i])
			break
		}
	}
}
