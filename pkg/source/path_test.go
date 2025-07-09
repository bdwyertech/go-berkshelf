package source

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
)

func TestPathSource_NewPathSource(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "berkshelf-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test valid path
	source, err := NewPathSource(tmpDir)
	if err != nil {
		t.Errorf("NewPathSource() error = %v", err)
	}
	if source == nil {
		t.Error("NewPathSource() returned nil")
	}

	// Test invalid path
	_, err = NewPathSource("/nonexistent/path")
	if err == nil {
		t.Error("NewPathSource() should error on non-existent path")
	}
}

func TestPathSource_ListVersions(t *testing.T) {
	// Create a test cookbook directory
	tmpDir, err := os.MkdirTemp("", "berkshelf-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a cookbook with metadata.json
	cookbookDir := filepath.Join(tmpDir, "test-cookbook")
	os.MkdirAll(cookbookDir, 0755)

	metadata := map[string]interface{}{
		"name":    "test-cookbook",
		"version": "1.2.3",
		"dependencies": map[string]string{
			"apt": ">= 2.0.0",
		},
	}

	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	metadataPath := filepath.Join(cookbookDir, "metadata.json")
	os.WriteFile(metadataPath, metadataJSON, 0644)

	// Test listing versions
	source, _ := NewPathSource(tmpDir)
	versions, err := source.ListVersions(context.Background(), "test-cookbook")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}

	if len(versions) != 1 {
		t.Errorf("ListVersions() returned %d versions, want 1", len(versions))
	}

	if versions[0].String() != "1.2.3" {
		t.Errorf("Version = %s, want 1.2.3", versions[0].String())
	}
}

func TestPathSource_FetchMetadata(t *testing.T) {
	// Create a test cookbook directory
	tmpDir, err := os.MkdirTemp("", "berkshelf-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a cookbook with metadata.json
	cookbookDir := filepath.Join(tmpDir, "nginx")
	os.MkdirAll(cookbookDir, 0755)

	metadata := map[string]interface{}{
		"name":        "nginx",
		"version":     "2.7.6",
		"description": "Installs and configures nginx",
		"maintainer":  "Test Author",
		"license":     "Apache-2.0",
		"dependencies": map[string]string{
			"apt":             "~> 2.2",
			"build-essential": "~> 2.0",
		},
	}

	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	metadataPath := filepath.Join(cookbookDir, "metadata.json")
	os.WriteFile(metadataPath, metadataJSON, 0644)

	// Test fetching metadata
	source, _ := NewPathSource(tmpDir)
	version, _ := berkshelf.NewVersion("2.7.6")
	meta, err := source.FetchMetadata(context.Background(), "nginx", version)
	if err != nil {
		t.Fatalf("FetchMetadata() error = %v", err)
	}

	if meta.Name != "nginx" {
		t.Errorf("Name = %s, want nginx", meta.Name)
	}

	if meta.Version.String() != "2.7.6" {
		t.Errorf("Version = %s, want 2.7.6", meta.Version.String())
	}

	if meta.Description != "Installs and configures nginx" {
		t.Errorf("Description = %s, want 'Installs and configures nginx'", meta.Description)
	}

	if len(meta.Dependencies) != 2 {
		t.Errorf("Dependencies count = %d, want 2", len(meta.Dependencies))
	}
}

func TestPathSource_FetchCookbook(t *testing.T) {
	// Create a test cookbook directory
	tmpDir, err := os.MkdirTemp("", "berkshelf-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a cookbook
	cookbookDir := filepath.Join(tmpDir, "test-cookbook")
	os.MkdirAll(cookbookDir, 0755)

	metadata := map[string]interface{}{
		"name":    "test-cookbook",
		"version": "1.0.0",
	}

	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	metadataPath := filepath.Join(cookbookDir, "metadata.json")
	os.WriteFile(metadataPath, metadataJSON, 0644)

	// Create some cookbook files
	recipesDir := filepath.Join(cookbookDir, "recipes")
	os.MkdirAll(recipesDir, 0755)
	os.WriteFile(filepath.Join(recipesDir, "default.rb"), []byte("# Default recipe"), 0644)

	// Test fetching cookbook
	source, _ := NewPathSource(tmpDir)
	version, _ := berkshelf.NewVersion("1.0.0")
	cookbook, err := source.FetchCookbook(context.Background(), "test-cookbook", version)
	if err != nil {
		t.Fatalf("FetchCookbook() error = %v", err)
	}

	if cookbook.Name != "test-cookbook" {
		t.Errorf("Name = %s, want test-cookbook", cookbook.Name)
	}

	if cookbook.Path != cookbookDir {
		t.Errorf("Path = %s, want %s", cookbook.Path, cookbookDir)
	}
}

func TestPathSource_MetadataRB(t *testing.T) {
	// Create a test cookbook directory with metadata.rb
	tmpDir, err := os.MkdirTemp("", "berkshelf-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a cookbook with metadata.rb
	cookbookDir := filepath.Join(tmpDir, "ruby-cookbook")
	os.MkdirAll(cookbookDir, 0755)

	metadataRB := `name 'ruby-cookbook'
maintainer 'Test Author'
maintainer_email 'test@example.com'
license 'Apache-2.0'
description 'Test cookbook with Ruby metadata'
version '0.1.0'

depends 'apt', '>= 2.0.0'
depends 'build-essential'
`

	metadataPath := filepath.Join(cookbookDir, "metadata.rb")
	os.WriteFile(metadataPath, []byte(metadataRB), 0644)

	// Test reading metadata.rb
	source, _ := NewPathSource(tmpDir)
	meta, err := source.FetchMetadata(context.Background(), "ruby-cookbook", nil)
	if err != nil {
		t.Fatalf("FetchMetadata() error = %v", err)
	}

	if meta.Name != "ruby-cookbook" {
		t.Errorf("Name = %s, want ruby-cookbook", meta.Name)
	}

	if meta.Version.String() != "0.1.0" {
		t.Errorf("Version = %s, want 0.1.0", meta.Version.String())
	}

	if len(meta.Dependencies) != 2 {
		t.Errorf("Dependencies count = %d, want 2", len(meta.Dependencies))
	}
}

func TestPathSource_DirectCookbookPath(t *testing.T) {
	// Create a cookbook directory that IS the path itself
	tmpDir, err := os.MkdirTemp("", "berkshelf-cookbook")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create metadata.json in the root
	metadata := map[string]interface{}{
		"name":    "direct-cookbook",
		"version": "1.0.0",
	}

	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	metadataPath := filepath.Join(tmpDir, "metadata.json")
	os.WriteFile(metadataPath, metadataJSON, 0644)

	// Test when the path itself is the cookbook
	source, _ := NewPathSource(tmpDir)
	cookbook, err := source.FetchCookbook(context.Background(), "direct-cookbook", nil)
	if err != nil {
		t.Fatalf("FetchCookbook() error = %v", err)
	}

	if cookbook.Path != tmpDir {
		t.Errorf("Path = %s, want %s", cookbook.Path, tmpDir)
	}
}

func TestPathSource_Priority(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "berkshelf-test")
	defer os.RemoveAll(tmpDir)

	source, _ := NewPathSource(tmpDir)

	if source.Priority() != 200 {
		t.Errorf("Priority() = %d, want 200", source.Priority())
	}
}

func TestPathSource_Search(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "berkshelf-test")
	defer os.RemoveAll(tmpDir)

	source, _ := NewPathSource(tmpDir)

	_, err := source.Search(context.Background(), "test")
	if err != ErrNotImplemented {
		t.Errorf("Search() error = %v, want ErrNotImplemented", err)
	}
}
