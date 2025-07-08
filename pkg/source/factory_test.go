package source

import (
	"os"
	"testing"

	"github.com/bdwyertech/go-berkshelf/pkg/berksfile"
	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
)

func TestFactory_CreateFromBerksfile(t *testing.T) {
	factory := NewFactory()

	// Test with default sources
	bf := &berksfile.Berksfile{
		Sources: []*berkshelf.SourceLocation{
			{Type: "supermarket", URL: "https://supermarket.chef.io"},
			{Type: "supermarket", URL: "https://internal.example.com"},
		},
	}

	manager, err := factory.CreateFromBerksfile(bf)
	if err != nil {
		t.Fatalf("CreateFromBerksfile() error = %v", err)
	}

	if manager == nil {
		t.Fatal("CreateFromBerksfile() returned nil manager")
	}

	// Should have created 2 sources
	if len(manager.sources) != 2 {
		t.Errorf("Manager has %d sources, want 2", len(manager.sources))
	}
}

func TestFactory_CreateFromBerksfile_NoSources(t *testing.T) {
	factory := NewFactory()

	// Test with no sources - should add default Supermarket
	bf := &berksfile.Berksfile{
		Sources: []*berkshelf.SourceLocation{},
	}

	manager, err := factory.CreateFromBerksfile(bf)
	if err != nil {
		t.Fatalf("CreateFromBerksfile() error = %v", err)
	}

	// Should have created 1 default source
	if len(manager.sources) != 1 {
		t.Errorf("Manager has %d sources, want 1", len(manager.sources))
	}

	// Should be Supermarket
	if manager.sources[0].Name() != "supermarket (https://supermarket.chef.io)" {
		t.Errorf("Default source name = %s, want supermarket", manager.sources[0].Name())
	}
}

func TestFactory_CreateFromLocation(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name     string
		location *berkshelf.SourceLocation
		wantType string
		wantErr  bool
	}{
		{
			name: "supermarket",
			location: &berkshelf.SourceLocation{
				Type: "supermarket",
				URL:  "https://supermarket.chef.io",
			},
			wantType: "supermarket",
			wantErr:  false,
		},
		{
			name: "git",
			location: &berkshelf.SourceLocation{
				Type: "git",
				URL:  "https://github.com/user/cookbook.git",
				Options: map[string]any{
					"branch": "master",
				},
			},
			wantType: "git",
			wantErr:  false,
		},
		{
			name: "github",
			location: &berkshelf.SourceLocation{
				Type: "github",
				URL:  "user/cookbook",
			},
			wantType: "git",
			wantErr:  false,
		},
		{
			name: "unknown",
			location: &berkshelf.SourceLocation{
				Type: "unknown",
				URL:  "something",
			},
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := factory.CreateFromLocation(tt.location)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFromLocation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && source == nil {
				t.Error("CreateFromLocation() returned nil source")
			}
		})
	}
}

func TestFactory_CreateFromURL(t *testing.T) {
	// Create a temp directory for testing file:// URLs
	tmpDir, err := os.MkdirTemp("", "berkshelf-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	factory := NewFactory()

	tests := []struct {
		url      string
		wantType string
		wantErr  bool
	}{
		{"https://supermarket.chef.io", "supermarket", false},
		{"http://internal.example.com", "supermarket", false},
		{"git://github.com/user/repo.git", "git", false},
		{"git@github.com:user/repo.git", "git", false},
		{"file://" + tmpDir, "path", false},
		{"file:///nonexistent/path", "path", true},
		{"custom-url", "supermarket", false}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			source, err := factory.createFromURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("createFromURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && source == nil {
				t.Error("createFromURL() returned nil")
			}
		})
	}
}

func TestFactory_AddDefaultSource(t *testing.T) {
	factory := NewFactory()

	// Add a default source
	defaultSource := NewSupermarketSource("https://internal.example.com")
	factory.AddDefaultSource(defaultSource)

	// Create manager with no sources
	bf := &berksfile.Berksfile{
		Sources: []*berkshelf.SourceLocation{},
	}

	manager, err := factory.CreateFromBerksfile(bf)
	if err != nil {
		t.Fatalf("CreateFromBerksfile() error = %v", err)
	}

	// Should use the default source
	if len(manager.sources) != 1 {
		t.Errorf("Manager has %d sources, want 1", len(manager.sources))
	}

	if manager.sources[0].Name() != "supermarket (https://internal.example.com)" {
		t.Errorf("Source name = %s, want internal supermarket", manager.sources[0].Name())
	}
}
