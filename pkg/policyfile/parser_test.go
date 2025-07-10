package policyfile

import (
	"testing"
)

func TestParsePolicyfile_Empty(t *testing.T) {
	input := ""
	policyfile, err := Parse(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.DefaultSources) != 0 {
		t.Errorf("Expected 0 default sources, got %d", len(policyfile.DefaultSources))
	}

	if len(policyfile.Cookbooks) != 0 {
		t.Errorf("Expected 0 cookbooks, got %d", len(policyfile.Cookbooks))
	}
}

func TestParsePolicyfile_DefaultSourceSupermarket(t *testing.T) {
	input := "default_source :supermarket"
	policyfile, err := Parse(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.DefaultSources) != 1 {
		t.Fatalf("Expected 1 default source, got %d", len(policyfile.DefaultSources))
	}

	source := policyfile.DefaultSources[0]
	if source.Type != "supermarket" {
		t.Errorf("Expected supermarket source type, got %v", source.Type)
	}

	if source.URL != "https://supermarket.chef.io" {
		t.Errorf("Expected default supermarket URL, got %s", source.URL)
	}
}

func TestParsePolicyfile_DefaultSourceWithURI(t *testing.T) {
	input := `default_source :supermarket, "https://private.supermarket.com"`
	policyfile, err := Parse(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.DefaultSources) != 1 {
		t.Fatalf("Expected 1 default source, got %d", len(policyfile.DefaultSources))
	}

	source := policyfile.DefaultSources[0]
	if source.Type != "supermarket" {
		t.Errorf("Expected supermarket source type, got %v", source.Type)
	}

	if source.URL != "https://private.supermarket.com" {
		t.Errorf("Expected private supermarket URL, got %s", source.URL)
	}
}

func TestParsePolicyfile_ChefServer(t *testing.T) {
	input := `default_source :chef_server, "https://chef.example.com/organizations/myorg"`
	policyfile, err := Parse(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.DefaultSources) != 1 {
		t.Fatalf("Expected 1 default source, got %d", len(policyfile.DefaultSources))
	}

	source := policyfile.DefaultSources[0]
	if source.Type != "chef_server" {
		t.Errorf("Expected chef_server source type, got %v", source.Type)
	}

	if source.URL != "https://chef.example.com/organizations/myorg" {
		t.Errorf("Expected chef server URL, got %s", source.URL)
	}
}

func TestParsePolicyfile_ChefRepo(t *testing.T) {
	input := `default_source :chef_repo, "/path/to/cookbooks"`
	policyfile, err := Parse(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.DefaultSources) != 1 {
		t.Fatalf("Expected 1 default source, got %d", len(policyfile.DefaultSources))
	}

	source := policyfile.DefaultSources[0]
	if source.Type != "path" {
		t.Errorf("Expected path source type, got %v", source.Type)
	}

	if source.Path != "/path/to/cookbooks" {
		t.Errorf("Expected path, got %s", source.Path)
	}
}

func TestParsePolicyfile_Cookbook(t *testing.T) {
	input := `cookbook "nginx"`
	policyfile, err := Parse(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.Cookbooks) != 1 {
		t.Fatalf("Expected 1 cookbook, got %d", len(policyfile.Cookbooks))
	}

	cookbook := policyfile.Cookbooks[0]
	if cookbook.Name != "nginx" {
		t.Errorf("Expected cookbook name 'nginx', got %s", cookbook.Name)
	}

	if cookbook.Constraint != nil {
		t.Errorf("Expected no constraint, got %v", cookbook.Constraint)
	}
}

func TestParsePolicyfile_CookbookWithVersion(t *testing.T) {
	input := `cookbook "nginx", "~> 2.7"`
	policyfile, err := Parse(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.Cookbooks) != 1 {
		t.Fatalf("Expected 1 cookbook, got %d", len(policyfile.Cookbooks))
	}

	cookbook := policyfile.Cookbooks[0]
	if cookbook.Name != "nginx" {
		t.Errorf("Expected cookbook name 'nginx', got %s", cookbook.Name)
	}

	if cookbook.Constraint == nil {
		t.Fatalf("Expected constraint, got nil")
	}

	if cookbook.Constraint.String() != "~> 2.7" {
		t.Errorf("Expected constraint '~> 2.7', got %s", cookbook.Constraint.String())
	}
}

func TestParsePolicyfile_MultipleStatements(t *testing.T) {
	input := `
default_source :supermarket
default_source :chef_repo, "/path/to/cookbooks"

cookbook "nginx", "~> 2.7"
cookbook "mysql"
`
	policyfile, err := Parse(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.DefaultSources) != 2 {
		t.Fatalf("Expected 2 default sources, got %d", len(policyfile.DefaultSources))
	}

	if len(policyfile.Cookbooks) != 2 {
		t.Fatalf("Expected 2 cookbooks, got %d", len(policyfile.Cookbooks))
	}

	// Check first source
	source1 := policyfile.DefaultSources[0]
	if source1.Type != "supermarket" {
		t.Errorf("Expected first source to be supermarket, got %v", source1.Type)
	}

	// Check second source
	source2 := policyfile.DefaultSources[1]
	if source2.Type != "path" {
		t.Errorf("Expected second source to be path, got %v", source2.Type)
	}

	// Check cookbooks
	if policyfile.Cookbooks[0].Name != "nginx" {
		t.Errorf("Expected first cookbook to be nginx, got %s", policyfile.Cookbooks[0].Name)
	}

	if policyfile.Cookbooks[1].Name != "mysql" {
		t.Errorf("Expected second cookbook to be mysql, got %s", policyfile.Cookbooks[1].Name)
	}
}

func TestParsePolicyfile_WithComments(t *testing.T) {
	input := `
# This is a comment
default_source :supermarket  # Another comment

# Cookbook definitions
cookbook "nginx", "~> 2.7"  # Version constraint
cookbook "mysql"            # No version
`
	policyfile, err := Parse(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.DefaultSources) != 1 {
		t.Fatalf("Expected 1 default source, got %d", len(policyfile.DefaultSources))
	}

	if len(policyfile.Cookbooks) != 2 {
		t.Fatalf("Expected 2 cookbooks, got %d", len(policyfile.Cookbooks))
	}
}
