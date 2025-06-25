package lockfile

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

func TestNewLockFile(t *testing.T) {
	lf := NewLockFile()

	if lf.Revision != 7 {
		t.Errorf("Expected revision 7, got %d", lf.Revision)
	}

	if lf.Sources == nil {
		t.Error("Sources map should be initialized")
	}

	if len(lf.Sources) != 0 {
		t.Errorf("Expected empty sources, got %d", len(lf.Sources))
	}
}

func TestAddCookbook(t *testing.T) {
	lf := NewLockFile()

	version, err := berkshelf.NewVersion("1.2.3")
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	constraint, err := berkshelf.NewConstraint("~> 1.0")
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	cookbook := &berkshelf.Cookbook{
		Name:    "nginx",
		Version: version,
		Dependencies: map[string]*berkshelf.Constraint{
			"apt": constraint,
		},
	}

	sourceInfo := &SourceInfo{
		Type: "supermarket",
		URL:  "https://supermarket.chef.io",
	}

	sourceURL := "https://supermarket.chef.io"
	lf.AddCookbook(sourceURL, cookbook, sourceInfo)

	// Check if source was created
	if len(lf.Sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(lf.Sources))
	}

	source, exists := lf.Sources[sourceURL]
	if !exists {
		t.Error("Source should exist")
	}

	if source.Type != "supermarket" {
		t.Errorf("Expected source type 'supermarket', got '%s'", source.Type)
	}

	// Check if cookbook was added
	if len(source.Cookbooks) != 1 {
		t.Errorf("Expected 1 cookbook, got %d", len(source.Cookbooks))
	}

	cookbookLock, exists := source.Cookbooks["nginx"]
	if !exists {
		t.Error("Cookbook 'nginx' should exist")
	}

	if cookbookLock.Version != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got '%s'", cookbookLock.Version)
	}

	// Check dependencies
	if len(cookbookLock.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(cookbookLock.Dependencies))
	}

	aptConstraint, exists := cookbookLock.Dependencies["apt"]
	if !exists {
		t.Error("Dependency 'apt' should exist")
	}

	if aptConstraint != "~> 1.0" {
		t.Errorf("Expected constraint '~> 1.0', got '%s'", aptConstraint)
	}
}

func TestGetCookbook(t *testing.T) {
	lf := NewLockFile()

	version, _ := berkshelf.NewVersion("1.2.3")
	cookbook := &berkshelf.Cookbook{
		Name:         "nginx",
		Version:      version,
		Dependencies: make(map[string]*berkshelf.Constraint),
	}

	sourceInfo := &SourceInfo{Type: "supermarket"}
	sourceURL := "https://supermarket.chef.io"

	lf.AddCookbook(sourceURL, cookbook, sourceInfo)

	// Test existing cookbook
	cookbookLock, source, exists := lf.GetCookbook("nginx")
	if !exists {
		t.Error("Cookbook should exist")
	}

	if cookbookLock.Version != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got '%s'", cookbookLock.Version)
	}

	if source != sourceURL {
		t.Errorf("Expected source URL '%s', got '%s'", sourceURL, source)
	}

	// Test non-existing cookbook
	_, _, exists = lf.GetCookbook("nonexistent")
	if exists {
		t.Error("Non-existent cookbook should not exist")
	}
}

func TestHasCookbook(t *testing.T) {
	lf := NewLockFile()

	version, _ := berkshelf.NewVersion("1.2.3")
	cookbook := &berkshelf.Cookbook{
		Name:         "nginx",
		Version:      version,
		Dependencies: make(map[string]*berkshelf.Constraint),
	}

	sourceInfo := &SourceInfo{Type: "supermarket"}
	sourceURL := "https://supermarket.chef.io"

	lf.AddCookbook(sourceURL, cookbook, sourceInfo)

	if !lf.HasCookbook("nginx") {
		t.Error("Cookbook 'nginx' should exist")
	}

	if lf.HasCookbook("nonexistent") {
		t.Error("Non-existent cookbook should not exist")
	}
}

func TestListCookbooks(t *testing.T) {
	lf := NewLockFile()

	// Add cookbook to first source
	version1, _ := berkshelf.NewVersion("1.2.3")
	cookbook1 := &berkshelf.Cookbook{
		Name:         "nginx",
		Version:      version1,
		Dependencies: make(map[string]*berkshelf.Constraint),
	}

	sourceInfo1 := &SourceInfo{Type: "supermarket"}
	sourceURL1 := "https://supermarket.chef.io"
	lf.AddCookbook(sourceURL1, cookbook1, sourceInfo1)

	// Add cookbook to second source
	version2, _ := berkshelf.NewVersion("2.0.0")
	cookbook2 := &berkshelf.Cookbook{
		Name:         "apache",
		Version:      version2,
		Dependencies: make(map[string]*berkshelf.Constraint),
	}

	sourceInfo2 := &SourceInfo{Type: "git"}
	sourceURL2 := "https://github.com/example/apache"
	lf.AddCookbook(sourceURL2, cookbook2, sourceInfo2)

	cookbooks := lf.ListCookbooks()

	if len(cookbooks) != 2 {
		t.Errorf("Expected 2 cookbooks, got %d", len(cookbooks))
	}

	if _, exists := cookbooks["nginx"]; !exists {
		t.Error("Cookbook 'nginx' should be in list")
	}

	if _, exists := cookbooks["apache"]; !exists {
		t.Error("Cookbook 'apache' should be in list")
	}
}

func TestToJSON(t *testing.T) {
	lf := NewLockFile()
	lf.GeneratedAt = time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	version, _ := berkshelf.NewVersion("1.2.3")
	constraint, _ := berkshelf.NewConstraint("~> 1.0")
	cookbook := &berkshelf.Cookbook{
		Name:    "nginx",
		Version: version,
		Dependencies: map[string]*berkshelf.Constraint{
			"apt": constraint,
		},
	}

	sourceInfo := &SourceInfo{
		Type: "supermarket",
		URL:  "https://supermarket.chef.io",
	}

	lf.AddCookbook("https://supermarket.chef.io", cookbook, sourceInfo)

	data, err := lf.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Check basic structure
	if revision, ok := parsed["revision"].(float64); !ok || revision != 7 {
		t.Errorf("Expected revision 7, got %v", parsed["revision"])
	}
}

func TestFromJSON(t *testing.T) {
	jsonData := `{
		"revision": 7,
		"generated_at": "2023-01-01T12:00:00Z",
		"sources": {
			"https://supermarket.chef.io": {
				"type": "supermarket",
				"url": "https://supermarket.chef.io",
				"cookbooks": {
					"nginx": {
						"version": "1.2.3",
						"dependencies": {
							"apt": "~> 1.0"
						},
						"source": {
							"type": "supermarket",
							"url": "https://supermarket.chef.io"
						}
					}
				}
			}
		}
	}`

	lf, err := FromJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if lf.Revision != 7 {
		t.Errorf("Expected revision 7, got %d", lf.Revision)
	}

	if !lf.HasCookbook("nginx") {
		t.Error("Cookbook 'nginx' should exist")
	}

	cookbook, _, exists := lf.GetCookbook("nginx")
	if !exists {
		t.Fatal("Cookbook 'nginx' should exist")
	}

	if cookbook.Version != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got '%s'", cookbook.Version)
	}

	if len(cookbook.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(cookbook.Dependencies))
	}
}

func TestIsOutdated(t *testing.T) {
	lf := NewLockFile()
	lf.GeneratedAt = time.Now().Add(-2 * time.Hour)

	// Test with 1 hour max age (should be outdated)
	if !lf.IsOutdated(1 * time.Hour) {
		t.Error("Lock file should be outdated")
	}

	// Test with 3 hour max age (should not be outdated)
	if lf.IsOutdated(3 * time.Hour) {
		t.Error("Lock file should not be outdated")
	}
}

func TestUpdateGeneratedAt(t *testing.T) {
	lf := NewLockFile()
	oldTime := lf.GeneratedAt

	// Wait a small amount to ensure time difference
	time.Sleep(1 * time.Millisecond)

	lf.UpdateGeneratedAt()

	if !lf.GeneratedAt.After(oldTime) {
		t.Error("GeneratedAt should be updated to a later time")
	}
}
