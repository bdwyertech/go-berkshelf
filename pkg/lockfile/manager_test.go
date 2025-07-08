package lockfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyertech/go-berkshelf/pkg/resolver"
	"github.com/bdwyertech/go-berkshelf/pkg/source"
)

func TestNewManager(t *testing.T) {
	workDir := "/tmp/test"
	manager := NewManager(workDir)

	expectedPath := filepath.Join(workDir, DefaultLockFileName)
	if manager.GetPath() != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, manager.GetPath())
	}
}

func TestNewManagerWithPath(t *testing.T) {
	customPath := "/tmp/custom/Berksfile.lock"
	manager := NewManagerWithPath(customPath)

	if manager.GetPath() != customPath {
		t.Errorf("Expected path %s, got %s", customPath, manager.GetPath())
	}
}

func TestManagerExists(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lockfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Should not exist initially
	if manager.Exists() {
		t.Error("Lock file should not exist initially")
	}

	// Create the lock file
	lockFile := NewLockFile()
	if err := manager.Save(lockFile); err != nil {
		t.Fatalf("Failed to save lock file: %v", err)
	}

	// Should exist now
	if !manager.Exists() {
		t.Error("Lock file should exist after saving")
	}
}

func TestManagerSaveAndLoad(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lockfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Create a lock file with some data
	originalLockFile := NewLockFile()

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
		URL:  source.PUBLIC_SUPERMARKET,
	}

	originalLockFile.AddCookbook(source.PUBLIC_SUPERMARKET, cookbook, sourceInfo)

	// Save the lock file
	if err := manager.Save(originalLockFile); err != nil {
		t.Fatalf("Failed to save lock file: %v", err)
	}

	// Load the lock file
	loadedLockFile, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load lock file: %v", err)
	}

	// Verify the loaded lock file
	if loadedLockFile.Revision != originalLockFile.Revision {
		t.Errorf("Expected revision %d, got %d", originalLockFile.Revision, loadedLockFile.Revision)
	}

	if !loadedLockFile.HasCookbook("nginx") {
		t.Error("Loaded lock file should contain nginx cookbook")
	}

	loadedCookbook, _, exists := loadedLockFile.GetCookbook("nginx")
	if !exists {
		t.Fatal("nginx cookbook should exist in loaded lock file")
	}

	if loadedCookbook.Version != "1.2.3" {
		t.Errorf("Expected version 1.2.3, got %s", loadedCookbook.Version)
	}
}

func TestManagerLoadNonExistent(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lockfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Load non-existent lock file should return empty lock file
	lockFile, err := manager.Load()
	if err != nil {
		t.Fatalf("Loading non-existent lock file should not error: %v", err)
	}

	if lockFile == nil {
		t.Fatal("Lock file should not be nil")
	}

	if len(lockFile.Sources) != 0 {
		t.Error("New lock file should have no sources")
	}
}

func TestManagerGenerate(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lockfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Create a mock resolution
	resolution := resolver.NewResolution()

	version, _ := berkshelf.NewVersion("1.2.3")
	cookbook := &berkshelf.Cookbook{
		Name:         "nginx",
		Version:      version,
		Dependencies: make(map[string]*berkshelf.Constraint),
	}

	resolvedCookbook := &resolver.ResolvedCookbook{
		Name:         "nginx",
		Version:      version,
		Source:       nil, // This will use default source
		Dependencies: make(map[string]*berkshelf.Version),
		Cookbook:     cookbook,
	}

	resolution.AddCookbook(resolvedCookbook)

	// Generate lock file
	lockFile, err := manager.Generate(resolution)
	if err != nil {
		t.Fatalf("Failed to generate lock file: %v", err)
	}

	if !lockFile.HasCookbook("nginx") {
		t.Error("Generated lock file should contain nginx cookbook")
	}
}

func TestManagerUpdate(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lockfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Create and save initial lock file
	initialLockFile := NewLockFile()
	if err := manager.Save(initialLockFile); err != nil {
		t.Fatalf("Failed to save initial lock file: %v", err)
	}

	// Create a resolution with a cookbook
	resolution := resolver.NewResolution()

	version, _ := berkshelf.NewVersion("1.2.3")
	cookbook := &berkshelf.Cookbook{
		Name:         "nginx",
		Version:      version,
		Dependencies: make(map[string]*berkshelf.Constraint),
	}

	resolvedCookbook := &resolver.ResolvedCookbook{
		Name:         "nginx",
		Version:      version,
		Source:       nil,
		Dependencies: make(map[string]*berkshelf.Version),
		Cookbook:     cookbook,
	}

	resolution.AddCookbook(resolvedCookbook)

	// Update lock file
	if err := manager.Update(resolution); err != nil {
		t.Fatalf("Failed to update lock file: %v", err)
	}

	// Load and verify updated lock file
	updatedLockFile, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load updated lock file: %v", err)
	}

	if !updatedLockFile.HasCookbook("nginx") {
		t.Error("Updated lock file should contain nginx cookbook")
	}
}

func TestManagerIsOutdated(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lockfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Non-existent lock file should be outdated
	outdated, err := manager.IsOutdated()
	if err != nil {
		t.Fatalf("Failed to check if lock file is outdated: %v", err)
	}
	if !outdated {
		t.Error("Non-existent lock file should be outdated")
	}

	// Create lock file
	lockFile := NewLockFile()
	if err := manager.Save(lockFile); err != nil {
		t.Fatalf("Failed to save lock file: %v", err)
	}

	// Without Berksfile, should not be outdated
	outdated, err = manager.IsOutdated()
	if err != nil {
		t.Fatalf("Failed to check if lock file is outdated: %v", err)
	}
	if outdated {
		t.Error("Lock file without Berksfile should not be outdated")
	}
}

func TestManagerValidate(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lockfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Validate non-existent lock file should error
	if err := manager.Validate(); err == nil {
		t.Error("Validating non-existent lock file should error")
	}

	// Create and save valid lock file
	lockFile := NewLockFile()
	version, _ := berkshelf.NewVersion("1.2.3")
	cookbook := &berkshelf.Cookbook{
		Name:         "nginx",
		Version:      version,
		Dependencies: make(map[string]*berkshelf.Constraint),
	}

	sourceInfo := &SourceInfo{
		Type: "supermarket",
		URL:  "https://supermarket.chef.io",
	}

	lockFile.AddCookbook("https://supermarket.chef.io", cookbook, sourceInfo)

	if err := manager.Save(lockFile); err != nil {
		t.Fatalf("Failed to save lock file: %v", err)
	}

	// Validate should pass
	if err := manager.Validate(); err != nil {
		t.Errorf("Validation should pass for valid lock file: %v", err)
	}
}

func TestManagerRemove(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lockfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Remove non-existent lock file should not error
	if err := manager.Remove(); err != nil {
		t.Errorf("Removing non-existent lock file should not error: %v", err)
	}

	// Create lock file
	lockFile := NewLockFile()
	if err := manager.Save(lockFile); err != nil {
		t.Fatalf("Failed to save lock file: %v", err)
	}

	// Verify it exists
	if !manager.Exists() {
		t.Fatal("Lock file should exist after saving")
	}

	// Remove it
	if err := manager.Remove(); err != nil {
		t.Errorf("Failed to remove lock file: %v", err)
	}

	// Verify it's gone
	if manager.Exists() {
		t.Error("Lock file should not exist after removal")
	}
}

func TestManagerBackup(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lockfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Backup non-existent lock file should error
	if err := manager.Backup(); err == nil {
		t.Error("Backing up non-existent lock file should error")
	}

	// Create lock file
	lockFile := NewLockFile()
	version, _ := berkshelf.NewVersion("1.2.3")
	cookbook := &berkshelf.Cookbook{
		Name:         "nginx",
		Version:      version,
		Dependencies: make(map[string]*berkshelf.Constraint),
	}

	sourceInfo := &SourceInfo{
		Type: "supermarket",
		URL:  "https://supermarket.chef.io",
	}

	lockFile.AddCookbook("https://supermarket.chef.io", cookbook, sourceInfo)

	if err := manager.Save(lockFile); err != nil {
		t.Fatalf("Failed to save lock file: %v", err)
	}

	// Create backup
	if err := manager.Backup(); err != nil {
		t.Errorf("Failed to create backup: %v", err)
	}

	// Verify backup exists
	backupPath := manager.GetPath() + ".backup"
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("Backup file should exist: %v", err)
	}
}
