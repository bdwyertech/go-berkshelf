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
func TestParsePolicyfile_CookbookWithChefServer(t *testing.T) {
	input := `cookbook "windows-security-policy", chef_server: "https://chef.my.org/", client_name: "myself", client_key: "~/.chef/key.pem"`
	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.Cookbooks) != 1 {
		t.Fatalf("Expected 1 cookbook, got %d", len(policyfile.Cookbooks))
	}

	cookbook := policyfile.Cookbooks[0]
	if cookbook.Name != "windows-security-policy" {
		t.Errorf("Expected cookbook name 'windows-security-policy', got %s", cookbook.Name)
	}

	if cookbook.Source == nil {
		t.Fatalf("Expected source, got nil")
	}

	if cookbook.Source.Type != "chef_server" {
		t.Errorf("Expected chef_server source type, got %s", cookbook.Source.Type)
	}

	if cookbook.Source.URL != "https://chef.my.org/" {
		t.Errorf("Expected chef server URL, got %s", cookbook.Source.URL)
	}

	if cookbook.Source.Options == nil {
		t.Fatalf("Expected options, got nil")
	}

	if clientName, ok := cookbook.Source.Options["client_name"]; !ok || clientName != "myself" {
		t.Errorf("Expected client_name 'myself', got %v", clientName)
	}

	if clientKey, ok := cookbook.Source.Options["client_key"]; !ok || clientKey != "~/.chef/key.pem" {
		t.Errorf("Expected client_key path, got %v", clientKey)
	}
}

func TestParsePolicyfile_CookbookWithSupermarket(t *testing.T) {
	input := `cookbook "nginx", supermarket: "https://private.supermarket.com", api_key: "my-api-key"`
	policyfile, err := ParsePolicyfile(input)
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

	if cookbook.Source == nil {
		t.Fatalf("Expected source, got nil")
	}

	if cookbook.Source.Type != "supermarket" {
		t.Errorf("Expected supermarket source type, got %s", cookbook.Source.Type)
	}

	if cookbook.Source.URL != "https://private.supermarket.com" {
		t.Errorf("Expected supermarket URL, got %s", cookbook.Source.URL)
	}

	if cookbook.Source.Options == nil {
		t.Fatalf("Expected options, got nil")
	}

	if apiKey, ok := cookbook.Source.Options["api_key"]; !ok || apiKey != "my-api-key" {
		t.Errorf("Expected api_key 'my-api-key', got %v", apiKey)
	}
}

func TestParsePolicyfile_CookbookWithArtifactory(t *testing.T) {
	input := `cookbook "mysql", artifactory: "https://artifactory.example/api/chef/my-supermarket", artifactory_api_key: "my-artifactory-key"`
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

	if cookbook.Source.Type != "supermarket" {
		t.Errorf("Expected supermarket source type (artifactory treated as supermarket), got %s", cookbook.Source.Type)
	}

	if cookbook.Source.URL != "https://artifactory.example/api/chef/my-supermarket" {
		t.Errorf("Expected artifactory URL, got %s", cookbook.Source.URL)
	}

	if cookbook.Source.Options == nil {
		t.Fatalf("Expected options, got nil")
	}

	if apiKey, ok := cookbook.Source.Options["artifactory_api_key"]; !ok || apiKey != "my-artifactory-key" {
		t.Errorf("Expected artifactory_api_key 'my-artifactory-key', got %v", apiKey)
	}
}

func TestParsePolicyfile_CookbookWithArtifactoryIdentityToken(t *testing.T) {
	input := `cookbook "apache2", artifactory: "https://artifactory.example/api/chef/my-supermarket", artifactory_identity_token: "my-identity-token"`
	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.Cookbooks) != 1 {
		t.Fatalf("Expected 1 cookbook, got %d", len(policyfile.Cookbooks))
	}

	cookbook := policyfile.Cookbooks[0]
	if cookbook.Source == nil {
		t.Fatalf("Expected source, got nil")
	}

	if cookbook.Source.Options == nil {
		t.Fatalf("Expected options, got nil")
	}

	if identityToken, ok := cookbook.Source.Options["artifactory_identity_token"]; !ok || identityToken != "my-identity-token" {
		t.Errorf("Expected artifactory_identity_token 'my-identity-token', got %v", identityToken)
	}
}

func TestParsePolicyfile_CookbookWithChefServerMultipleOptions(t *testing.T) {
	input := `cookbook "test-cookbook", chef_server: "https://chef.example.com/organizations/test", client_name: "testuser", client_key: "~/.chef/testuser.pem", node_name: "test-node"`
	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.Cookbooks) != 1 {
		t.Fatalf("Expected 1 cookbook, got %d", len(policyfile.Cookbooks))
	}

	cookbook := policyfile.Cookbooks[0]
	if cookbook.Source == nil {
		t.Fatalf("Expected source, got nil")
	}

	if cookbook.Source.Type != "chef_server" {
		t.Errorf("Expected chef_server source type, got %s", cookbook.Source.Type)
	}

	if cookbook.Source.Options == nil {
		t.Fatalf("Expected options, got nil")
	}

	expectedOptions := map[string]string{
		"client_name": "testuser",
		"client_key":  "~/.chef/testuser.pem",
		"node_name":   "test-node",
	}

	for key, expectedValue := range expectedOptions {
		if actualValue, ok := cookbook.Source.Options[key]; !ok || actualValue != expectedValue {
			t.Errorf("Expected %s '%s', got %v", key, expectedValue, actualValue)
		}
	}
}

func TestParsePolicyfile_CookbookWithVersionAndChefServer(t *testing.T) {
	input := `cookbook "windows-security-policy", "~> 1.0", chef_server: "https://chef.my.org/", client_name: "myself"`
	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.Cookbooks) != 1 {
		t.Fatalf("Expected 1 cookbook, got %d", len(policyfile.Cookbooks))
	}

	cookbook := policyfile.Cookbooks[0]
	if cookbook.Name != "windows-security-policy" {
		t.Errorf("Expected cookbook name 'windows-security-policy', got %s", cookbook.Name)
	}

	if cookbook.Constraint == nil {
		t.Fatalf("Expected constraint, got nil")
	}

	if cookbook.Constraint.String() != "~> 1.0" {
		t.Errorf("Expected constraint '~> 1.0', got %s", cookbook.Constraint.String())
	}

	if cookbook.Source == nil {
		t.Fatalf("Expected source, got nil")
	}

	if cookbook.Source.Type != "chef_server" {
		t.Errorf("Expected chef_server source type, got %s", cookbook.Source.Type)
	}
}

func TestParsePolicyfile_ComprehensiveMixedCookbookSources(t *testing.T) {
	input := `
default_source :supermarket

cookbook "nginx", "~> 2.7"
cookbook "my_app", path: "cookbooks/my_app"
cookbook "mysql", github: "opscode-cookbooks/mysql", branch: "master"
cookbook "chef-ingredient", git: "https://github.com/chef-cookbooks/chef-ingredient.git", tag: "v0.12.0"
cookbook "windows-security-policy", chef_server: "https://chef.my.org/", client_name: "myself", client_key: "~/.chef/key.pem"
cookbook "private-cookbook", supermarket: "https://private.supermarket.com", api_key: "my-api-key"
cookbook "artifactory-cookbook", artifactory: "https://artifactory.example/api/chef/my-supermarket", artifactory_api_key: "my-artifactory-key"
`
	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(policyfile.DefaultSources) != 1 {
		t.Fatalf("Expected 1 default source, got %d", len(policyfile.DefaultSources))
	}

	if len(policyfile.Cookbooks) != 7 {
		t.Fatalf("Expected 7 cookbooks, got %d", len(policyfile.Cookbooks))
	}

	// Verify each cookbook type
	cookbookTests := []struct {
		name       string
		sourceType string
		hasOptions bool
	}{
		{"nginx", "", false},                             // Uses default source
		{"my_app", "path", false},                        // Path source
		{"mysql", "git", false},                          // GitHub source
		{"chef-ingredient", "git", false},                // Git source
		{"windows-security-policy", "chef_server", true}, // Chef server with options
		{"private-cookbook", "supermarket", true},        // Private supermarket with API key
		{"artifactory-cookbook", "supermarket", true},    // Artifactory with API key
	}

	for i, test := range cookbookTests {
		cookbook := policyfile.Cookbooks[i]
		if cookbook.Name != test.name {
			t.Errorf("Cookbook %d: expected name '%s', got '%s'", i, test.name, cookbook.Name)
		}

		if test.sourceType == "" {
			if cookbook.Source != nil {
				t.Errorf("Cookbook %d (%s): expected no specific source, got %v", i, test.name, cookbook.Source)
			}
		} else {
			if cookbook.Source == nil {
				t.Errorf("Cookbook %d (%s): expected source, got nil", i, test.name)
				continue
			}
			if cookbook.Source.Type != test.sourceType {
				t.Errorf("Cookbook %d (%s): expected source type '%s', got '%s'", i, test.name, test.sourceType, cookbook.Source.Type)
			}
		}

		if test.hasOptions {
			if cookbook.Source == nil || cookbook.Source.Options == nil || len(cookbook.Source.Options) == 0 {
				t.Errorf("Cookbook %d (%s): expected options, got none", i, test.name)
			}
		}
	}
}
