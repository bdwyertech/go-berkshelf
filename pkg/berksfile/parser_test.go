package berksfile

import (
	"strings"
	"testing"
)

func TestParser_BasicBerksfile(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, b *Berksfile)
	}{
		{
			name:  "empty berksfile",
			input: "",
			check: func(t *testing.T, b *Berksfile) {
				if len(b.Sources) != 0 {
					t.Errorf("expected 0 sources, got %d", len(b.Sources))
				}
				if len(b.Cookbooks) != 0 {
					t.Errorf("expected 0 cookbooks, got %d", len(b.Cookbooks))
				}
			},
		},
		{
			name:  "source declaration",
			input: `source 'https://supermarket.chef.io'`,
			check: func(t *testing.T, b *Berksfile) {
				if len(b.Sources) != 1 {
					t.Fatalf("expected 1 source, got %d", len(b.Sources))
				}
				if b.Sources[0] != "https://supermarket.chef.io" {
					t.Errorf("expected source 'https://supermarket.chef.io', got %q", b.Sources[0])
				}
			},
		},
		{
			name:  "metadata directive",
			input: `metadata`,
			check: func(t *testing.T, b *Berksfile) {
				if !b.HasMetadata {
					t.Error("expected HasMetadata to be true")
				}
			},
		},
		{
			name:  "simple cookbook",
			input: `cookbook 'nginx'`,
			check: func(t *testing.T, b *Berksfile) {
				if len(b.Cookbooks) != 1 {
					t.Fatalf("expected 1 cookbook, got %d", len(b.Cookbooks))
				}
				if b.Cookbooks[0].Name != "nginx" {
					t.Errorf("expected cookbook name 'nginx', got %q", b.Cookbooks[0].Name)
				}
				if b.Cookbooks[0].Constraint == nil {
					t.Fatal("expected default constraint")
				}
				if b.Cookbooks[0].Constraint.String() != ">= 0.0.0" {
					t.Errorf("expected default constraint '>= 0.0.0', got %q", b.Cookbooks[0].Constraint.String())
				}
			},
		},
		{
			name:  "cookbook with version",
			input: `cookbook 'nginx', '~> 2.7.6'`,
			check: func(t *testing.T, b *Berksfile) {
				if len(b.Cookbooks) != 1 {
					t.Fatalf("expected 1 cookbook, got %d", len(b.Cookbooks))
				}
				cookbook := b.Cookbooks[0]
				if cookbook.Name != "nginx" {
					t.Errorf("expected cookbook name 'nginx', got %q", cookbook.Name)
				}
				if cookbook.Constraint == nil {
					t.Fatal("expected constraint")
				}
				if cookbook.Constraint.String() != "~> 2.7.6" {
					t.Errorf("expected constraint '~> 2.7.6', got %q", cookbook.Constraint.String())
				}
			},
		},
		{
			name:  "cookbook with git source",
			input: `cookbook 'private', git: 'git@github.com:user/repo.git'`,
			check: func(t *testing.T, b *Berksfile) {
				if len(b.Cookbooks) != 1 {
					t.Fatalf("expected 1 cookbook, got %d", len(b.Cookbooks))
				}
				cookbook := b.Cookbooks[0]
				if cookbook.Name != "private" {
					t.Errorf("expected cookbook name 'private', got %q", cookbook.Name)
				}
				if cookbook.Source == nil {
					t.Fatal("expected source to be set")
				}
				if cookbook.Source.Type != "git" {
					t.Errorf("expected source type 'git', got %q", cookbook.Source.Type)
				}
				if cookbook.Source.URL != "git@github.com:user/repo.git" {
					t.Errorf("expected source URL 'git@github.com:user/repo.git', got %q", cookbook.Source.URL)
				}
			},
		},
		{
			name:  "cookbook with github shorthand",
			input: `cookbook 'private', github: 'user/repo'`,
			check: func(t *testing.T, b *Berksfile) {
				if len(b.Cookbooks) != 1 {
					t.Fatalf("expected 1 cookbook, got %d", len(b.Cookbooks))
				}
				cookbook := b.Cookbooks[0]
				if cookbook.Source == nil {
					t.Fatal("expected source to be set")
				}
				if cookbook.Source.Type != "git" {
					t.Errorf("expected source type 'git', got %q", cookbook.Source.Type)
				}
				if cookbook.Source.URL != "https://github.com/user/repo.git" {
					t.Errorf("expected source URL 'https://github.com/user/repo.git', got %q", cookbook.Source.URL)
				}
			},
		},
		{
			name:  "cookbook with path source",
			input: `cookbook 'myapp', path: '../myapp'`,
			check: func(t *testing.T, b *Berksfile) {
				if len(b.Cookbooks) != 1 {
					t.Fatalf("expected 1 cookbook, got %d", len(b.Cookbooks))
				}
				cookbook := b.Cookbooks[0]
				if cookbook.Source == nil {
					t.Fatal("expected source to be set")
				}
				if cookbook.Source.Type != "path" {
					t.Errorf("expected source type 'path', got %q", cookbook.Source.Type)
				}
				if cookbook.Source.Path != "../myapp" {
					t.Errorf("expected source path '../myapp', got %q", cookbook.Source.Path)
				}
			},
		},
		{
			name:  "cookbook with git source and branch",
			input: `cookbook 'private', git: 'git@github.com:user/repo.git', branch: 'develop'`,
			check: func(t *testing.T, b *Berksfile) {
				if len(b.Cookbooks) != 1 {
					t.Fatalf("expected 1 cookbook, got %d", len(b.Cookbooks))
				}
				cookbook := b.Cookbooks[0]
				if cookbook.Source == nil {
					t.Fatal("expected source to be set")
				}
				if branch, ok := cookbook.Source.Options["branch"]; !ok || branch != "develop" {
					t.Errorf("expected branch 'develop', got %v", branch)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			berksfile, err := ParseBerksfile(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, berksfile)
		})
	}
}

func TestParser_Groups(t *testing.T) {
	input := `
group :test do
  cookbook 'minitest-handler'
end

group :integration do
  cookbook 'test-kitchen'
  cookbook 'kitchen-vagrant'
end

group :development, :test do
  cookbook 'chefspec'
end
`

	berksfile, err := ParseBerksfile(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check total cookbooks
	if len(berksfile.Cookbooks) != 4 {
		t.Errorf("expected 4 cookbooks, got %d", len(berksfile.Cookbooks))
	}

	// Check groups
	if len(berksfile.Groups) != 3 {
		t.Errorf("expected 3 groups, got %d", len(berksfile.Groups))
	}

	// Check test group
	if testGroup, ok := berksfile.Groups["test"]; ok {
		if len(testGroup) != 2 { // minitest-handler and chefspec
			t.Errorf("expected 2 cookbooks in test group, got %d", len(testGroup))
		}
	} else {
		t.Error("test group not found")
	}

	// Check integration group
	if integrationGroup, ok := berksfile.Groups["integration"]; ok {
		if len(integrationGroup) != 2 {
			t.Errorf("expected 2 cookbooks in integration group, got %d", len(integrationGroup))
		}
	} else {
		t.Error("integration group not found")
	}

	// Check development group
	if devGroup, ok := berksfile.Groups["development"]; ok {
		if len(devGroup) != 1 {
			t.Errorf("expected 1 cookbook in development group, got %d", len(devGroup))
		}
	} else {
		t.Error("development group not found")
	}

	// Check that chefspec is in both development and test groups
	chefspec := berksfile.GetCookbook("chefspec")
	if chefspec == nil {
		t.Fatal("chefspec cookbook not found")
	}
	if len(chefspec.Groups) != 2 {
		t.Errorf("expected chefspec to be in 2 groups, got %d", len(chefspec.Groups))
	}
}

func TestParser_CompleteBerksfile(t *testing.T) {
	input := `# Berksfile for myapp
source 'https://supermarket.chef.io'

metadata

cookbook 'nginx', '~> 2.7.6'
cookbook 'mysql', '>= 5.0.0'
cookbook 'postgresql', '= 9.3.0'

cookbook 'private', git: 'git@github.com:user/private-cookbook.git', branch: 'master'
cookbook 'myapp', path: '../myapp'

group :integration do
  cookbook 'test-kitchen'
  cookbook 'kitchen-vagrant'
end

group :development do
  cookbook 'chefspec', '~> 4.0'
  cookbook 'rubocop'
end
`

	berksfile, err := ParseBerksfile(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check sources
	if len(berksfile.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(berksfile.Sources))
	}

	// Check metadata
	if !berksfile.HasMetadata {
		t.Error("expected HasMetadata to be true")
	}

	// Check total cookbooks
	if len(berksfile.Cookbooks) != 9 {
		t.Errorf("expected 9 cookbooks, got %d", len(berksfile.Cookbooks))
	}

	// Check specific cookbooks
	nginx := berksfile.GetCookbook("nginx")
	if nginx == nil {
		t.Fatal("nginx cookbook not found")
	}
	if nginx.Constraint == nil || nginx.Constraint.String() != "~> 2.7.6" {
		t.Error("nginx constraint mismatch")
	}

	private := berksfile.GetCookbook("private")
	if private == nil {
		t.Fatal("private cookbook not found")
	}
	if private.Source == nil {
		t.Fatal("expected private cookbook to have source")
	}
	if private.Source.Type != "git" {
		t.Error("private cookbook should have git source")
	}
	if branch, ok := private.Source.Options["branch"]; !ok || branch != "master" {
		t.Error("private cookbook should have master branch")
	}

	// Check groups
	if len(berksfile.Groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(berksfile.Groups))
	}

	// Check GetCookbooks with groups
	integrationCookbooks := berksfile.GetCookbooks("integration")
	if len(integrationCookbooks) != 2 {
		t.Errorf("expected 2 integration cookbooks, got %d", len(integrationCookbooks))
	}

	developmentCookbooks := berksfile.GetCookbooks("development")
	if len(developmentCookbooks) != 2 {
		t.Errorf("expected 2 development cookbooks, got %d", len(developmentCookbooks))
	}
}

func TestParser_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "invalid constraint",
			input:       `cookbook 'nginx', 'invalid constraint'`,
			shouldError: true,
			errorMsg:    "invalid version constraint",
		},
		{
			name:        "missing cookbook name",
			input:       `cookbook`,
			shouldError: true,
			errorMsg:    "expected cookbook name",
		},
		{
			name:        "missing source URL",
			input:       `source`,
			shouldError: true,
			errorMsg:    "expected string after 'source'",
		},
		{
			name:        "incomplete group",
			input:       `group :test`,
			shouldError: true, // Groups without 'do' or content should error
			errorMsg:    "unexpected token EOF in group",
		},
		{
			name: "unterminated group",
			input: `group :test do
		cookbook 'test'`,
			shouldError: true, // Missing 'end' should error
			errorMsg:    "unexpected token EOF in group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBerksfile(tt.input)
			if tt.shouldError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.shouldError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %v", tt.errorMsg, err)
				}
			}
		})
	}
}
