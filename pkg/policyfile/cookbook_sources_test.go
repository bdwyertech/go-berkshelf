package policyfile

import (
	"testing"
)

func TestParsePolicyfile_CookbookWithPath(t *testing.T) {
	input := `cookbook "my_app", path: "cookbooks/my_app"`
	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.Cookbooks) != 1 {
		t.Fatalf("Expected 1 cookbook, got %d", len(policyfile.Cookbooks))
	}

	cookbook := policyfile.Cookbooks[0]
	if cookbook.Name != "my_app" {
		t.Errorf("Expected cookbook name 'my_app', got %s", cookbook.Name)
	}

	if cookbook.Source == nil {
		t.Fatalf("Expected source, got nil")
	}

	if cookbook.Source.Type != "path" {
		t.Errorf("Expected path source type, got %s", cookbook.Source.Type)
	}

	if cookbook.Source.Path != "cookbooks/my_app" {
		t.Errorf("Expected path 'cookbooks/my_app', got %s", cookbook.Source.Path)
	}
}

func TestParsePolicyfile_CookbookWithGit(t *testing.T) {
	input := `cookbook "chef-ingredient", git: "https://github.com/chef-cookbooks/chef-ingredient.git", tag: "v0.12.0"`
	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.Cookbooks) != 1 {
		t.Fatalf("Expected 1 cookbook, got %d", len(policyfile.Cookbooks))
	}

	cookbook := policyfile.Cookbooks[0]
	if cookbook.Name != "chef-ingredient" {
		t.Errorf("Expected cookbook name 'chef-ingredient', got %s", cookbook.Name)
	}

	if cookbook.Source == nil {
		t.Fatalf("Expected source, got nil")
	}

	if cookbook.Source.Type != "git" {
		t.Errorf("Expected git source type, got %s", cookbook.Source.Type)
	}

	if cookbook.Source.URL != "https://github.com/chef-cookbooks/chef-ingredient.git" {
		t.Errorf("Expected git URL, got %s", cookbook.Source.URL)
	}

	if cookbook.Source.Ref != "v0.12.0" {
		t.Errorf("Expected tag 'v0.12.0', got %s", cookbook.Source.Ref)
	}
}

func TestParsePolicyfile_CookbookWithGitBranch(t *testing.T) {
	input := `cookbook "mysql", git: "https://github.com/opscode-cookbooks/mysql.git", branch: "master"`
	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.Cookbooks) != 1 {
		t.Fatalf("Expected 1 cookbook, got %d", len(policyfile.Cookbooks))
	}

	cookbook := policyfile.Cookbooks[0]
	if cookbook.Name != "mysql" {
		t.Errorf("Expected cookbook name 'mysql', got %s", cookbook.Name)
	}

	if cookbook.Source == nil {
		t.Fatalf("Expected source, got nil")
	}

	if cookbook.Source.Type != "git" {
		t.Errorf("Expected git source type, got %s", cookbook.Source.Type)
	}

	if cookbook.Source.URL != "https://github.com/opscode-cookbooks/mysql.git" {
		t.Errorf("Expected git URL, got %s", cookbook.Source.URL)
	}

	if cookbook.Source.Ref != "master" {
		t.Errorf("Expected branch 'master', got %s", cookbook.Source.Ref)
	}
}

func TestParsePolicyfile_CookbookWithGithub(t *testing.T) {
	input := `cookbook "mysql", github: "opscode-cookbooks/mysql", branch: "master"`
	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.Cookbooks) != 1 {
		t.Fatalf("Expected 1 cookbook, got %d", len(policyfile.Cookbooks))
	}

	cookbook := policyfile.Cookbooks[0]
	if cookbook.Name != "mysql" {
		t.Errorf("Expected cookbook name 'mysql', got %s", cookbook.Name)
	}

	if cookbook.Source == nil {
		t.Fatalf("Expected source, got nil")
	}

	if cookbook.Source.Type != "git" {
		t.Errorf("Expected git source type, got %s", cookbook.Source.Type)
	}

	expectedURL := "https://github.com/opscode-cookbooks/mysql.git"
	if cookbook.Source.URL != expectedURL {
		t.Errorf("Expected github URL %s, got %s", expectedURL, cookbook.Source.URL)
	}

	if cookbook.Source.Ref != "master" {
		t.Errorf("Expected branch 'master', got %s", cookbook.Source.Ref)
	}
}

func TestParsePolicyfile_CookbookWithVersionAndSource(t *testing.T) {
	input := `cookbook "jenkins", "~> 2.1", git: "https://github.com/chef-cookbooks/jenkins.git"`
	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.Cookbooks) != 1 {
		t.Fatalf("Expected 1 cookbook, got %d", len(policyfile.Cookbooks))
	}

	cookbook := policyfile.Cookbooks[0]
	if cookbook.Name != "jenkins" {
		t.Errorf("Expected cookbook name 'jenkins', got %s", cookbook.Name)
	}

	if cookbook.Constraint == nil {
		t.Fatalf("Expected constraint, got nil")
	}

	if cookbook.Constraint.String() != "~> 2.1" {
		t.Errorf("Expected constraint '~> 2.1', got %s", cookbook.Constraint.String())
	}

	if cookbook.Source == nil {
		t.Fatalf("Expected source, got nil")
	}

	if cookbook.Source.Type != "git" {
		t.Errorf("Expected git source type, got %s", cookbook.Source.Type)
	}

	if cookbook.Source.URL != "https://github.com/chef-cookbooks/jenkins.git" {
		t.Errorf("Expected git URL, got %s", cookbook.Source.URL)
	}
}

func TestParsePolicyfile_MultipleCookbooksWithSources(t *testing.T) {
	input := `
default_source :supermarket

cookbook "nginx", "~> 2.7"
cookbook "my_app", path: "cookbooks/my_app"
cookbook "mysql", github: "opscode-cookbooks/mysql", branch: "master"
cookbook "chef-ingredient", git: "https://github.com/chef-cookbooks/chef-ingredient.git", tag: "v0.12.0"
`
	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.DefaultSources) != 1 {
		t.Fatalf("Expected 1 default source, got %d", len(policyfile.DefaultSources))
	}

	if len(policyfile.Cookbooks) != 4 {
		t.Fatalf("Expected 4 cookbooks, got %d", len(policyfile.Cookbooks))
	}

	// Check nginx (version constraint, no specific source)
	nginx := policyfile.Cookbooks[0]
	if nginx.Name != "nginx" {
		t.Errorf("Expected first cookbook to be nginx, got %s", nginx.Name)
	}
	if nginx.Constraint == nil || nginx.Constraint.String() != "~> 2.7" {
		t.Errorf("Expected nginx constraint '~> 2.7', got %v", nginx.Constraint)
	}
	if nginx.Source != nil {
		t.Errorf("Expected nginx to have no specific source, got %v", nginx.Source)
	}

	// Check my_app (path source)
	myApp := policyfile.Cookbooks[1]
	if myApp.Name != "my_app" {
		t.Errorf("Expected second cookbook to be my_app, got %s", myApp.Name)
	}
	if myApp.Source == nil || myApp.Source.Type != "path" {
		t.Errorf("Expected my_app to have path source, got %v", myApp.Source)
	}

	// Check mysql (github source)
	mysql := policyfile.Cookbooks[2]
	if mysql.Name != "mysql" {
		t.Errorf("Expected third cookbook to be mysql, got %s", mysql.Name)
	}
	if mysql.Source == nil || mysql.Source.Type != "git" {
		t.Errorf("Expected mysql to have git source, got %v", mysql.Source)
	}

	// Check chef-ingredient (git source with tag)
	chefIngredient := policyfile.Cookbooks[3]
	if chefIngredient.Name != "chef-ingredient" {
		t.Errorf("Expected fourth cookbook to be chef-ingredient, got %s", chefIngredient.Name)
	}
	if chefIngredient.Source == nil || chefIngredient.Source.Type != "git" {
		t.Errorf("Expected chef-ingredient to have git source, got %v", chefIngredient.Source)
	}
	if chefIngredient.Source.Ref != "v0.12.0" {
		t.Errorf("Expected chef-ingredient tag 'v0.12.0', got %s", chefIngredient.Source.Ref)
	}
}
