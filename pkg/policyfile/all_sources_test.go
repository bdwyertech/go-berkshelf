package policyfile

import (
	"testing"
)

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

func TestParsePolicyfile_MixedCookbookSources(t *testing.T) {
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
