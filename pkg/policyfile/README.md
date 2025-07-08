# Policyfile Package

This package provides parsing support for Chef Policyfile.rb files, specifically focusing on the Berkshelf-equivalent directives: `default_source` and `cookbook`.

## Overview

The policyfile package is designed to parse only the dependency management aspects of Chef Policyfile.rb files that are equivalent to Berkshelf functionality. It does not parse the full Policyfile.rb specification (run_list, named_run_list, etc.) but focuses on cookbook source and dependency declarations.

## Supported Directives

### default_source

Specifies the default location for cookbooks not explicitly sourced elsewhere.

Supported source types:
- `:supermarket` - Chef Supermarket (public or private)
- `:chef_server` - Chef Infra Server
- `:chef_repo` - Local cookbook repository/monorepo
- `:artifactory` - Artifactory server (treated as supermarket-like)

Examples:
```ruby
# Public Chef Supermarket (default)
default_source :supermarket

# Private Chef Supermarket
default_source :supermarket, "https://private.supermarket.com"

# Chef Server
default_source :chef_server, "https://chef.example.com/organizations/myorg"

# Local cookbook repository
default_source :chef_repo, "/path/to/cookbooks"

# Artifactory
default_source :artifactory, "https://artifactory.example/api/chef/my-supermarket"
```

### cookbook

Declares a cookbook dependency with optional version constraints and alternative sources.

#### Basic Cookbook Declarations

```ruby
# Simple cookbook declaration
cookbook "nginx"

# With version constraint
cookbook "nginx", "~> 2.7"
cookbook "mysql", ">= 1.0.0"
cookbook "apache2", "= 2.4.1"
```

#### Alternative Cookbook Sources

**Path Sources:**
```ruby
cookbook 'my_app', path: 'cookbooks/my_app'
```

**Git Sources:**
```ruby
# Full git URL with tag
cookbook 'chef-ingredient', git: 'https://github.com/chef-cookbooks/chef-ingredient.git', tag: 'v0.12.0'

# Git with branch
cookbook 'mysql', git: 'https://github.com/opscode-cookbooks/mysql.git', branch: 'master'

# Git with specific ref
cookbook 'test-cookbook', git: 'https://github.com/example/test-cookbook.git', ref: 'abc123'
```

**GitHub Shorthand:**
```ruby
# GitHub shorthand with branch
cookbook 'mysql', github: 'opscode-cookbooks/mysql', branch: 'master'

# GitHub shorthand with tag
cookbook 'nginx', github: 'chef-cookbooks/nginx', tag: 'v2.7.0'
```

**Chef Server Sources:**
```ruby
# Chef Server with authentication
cookbook 'windows-security-policy', chef_server: "https://chef.example.com/organizations/myorg", client_name: "myuser", client_key: "~/.chef/myorg/devops/myuser.pem"

# Chef Server with multiple options
cookbook 'enterprise-cookbook', chef_server: "https://chef.example.com/organizations/myorg", client_name: "myuser", client_key: "~/.chef/myuser.pem", node_name: "mynode"

# Chef Server with validation client
cookbook 'validation-cookbook', chef_server: "https://chef.example.com/organizations/myorg", validation_client_name: "myorg-validator", validation_key: "~/.chef/myorg-validator.pem"
```

**Private Supermarket Sources:**
```ruby
# Private Supermarket with API key
cookbook 'private-cookbook', supermarket: "https://private.supermarket.com", api_key: "my-api-key"
```

**Artifactory Sources:**
```ruby
# Artifactory with API key
cookbook 'artifactory-cookbook', artifactory: "https://artifactory.example/api/chef/my-supermarket", artifactory_api_key: "my-artifactory-key"

# Artifactory with identity token (Chef Workstation v24.11+)
cookbook 'modern-cookbook', artifactory: "https://artifactory.example/api/chef/my-supermarket", artifactory_identity_token: "my-identity-token"
```

**Combined Version Constraints and Sources:**
```ruby
# Version constraint with alternative source
cookbook 'jenkins', '~> 2.1', git: 'https://github.com/chef-cookbooks/jenkins.git'
cookbook 'windows-security-policy', '~> 1.0', chef_server: "https://chef.example.com/organizations/myorg", client_name: "myuser"
```

## Usage

### Basic Parsing

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/bdwyertech/go-berkshelf/pkg/policyfile"
)

func main() {
    input := `
    default_source :supermarket
    cookbook "nginx", "~> 2.7"
    cookbook "my_app", path: "cookbooks/my_app"
    cookbook "windows-security-policy", chef_server: "https://chef.example.com/", client_name: "myuser", client_key: "~/.chef/myuser.pem"
    `
    
    policyfile, err := policyfile.ParsePolicyfile(input)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Sources: %d\n", len(policyfile.DefaultSources))
    fmt.Printf("Cookbooks: %d\n", len(policyfile.Cookbooks))
}
```

### Loading from File

```go
policyfile, err := policyfile.LoadPolicyfile("./Policyfile.rb")
if err != nil {
    log.Fatal(err)
}
```

### Finding Policyfile.rb

```go
path, err := policyfile.FindPolicyfile(".")
if err != nil {
    log.Fatal(err)
}

policyfile, err := policyfile.LoadPolicyfile(path)
if err != nil {
    log.Fatal(err)
}
```

### Converting to Berkshelf-Compatible Format

```go
equivalent, err := policyfile.ToBerksfileEquivalent()
if err != nil {
    log.Fatal(err)
}

// Use equivalent.Sources and equivalent.Cookbooks with existing Berkshelf resolver
```

## Data Structures

### Policyfile

```go
type Policyfile struct {
    DefaultSources []*berkshelf.SourceLocation   // List of default sources
    Cookbooks      []*CookbookDef                // All cookbook definitions
}
```

### CookbookDef

```go
type CookbookDef struct {
    Name       string                    // Cookbook name
    Constraint *berkshelf.Constraint     // Version constraint (optional)
    Source     *berkshelf.SourceLocation // Specific source (optional)
}
```

### SourceLocation (from berkshelf package)

```go
type SourceLocation struct {
    Type    string         `json:"type"`    // "supermarket", "git", "path", "chef_server"
    URL     string         `json:"url,omitempty"`
    Ref     string         `json:"ref,omitempty"`  // git branch/tag/commit
    Path    string         `json:"path,omitempty"` // local path
    Options map[string]any `json:"options,omitempty"` // source-specific options
}
```

## Supported Source Options

### Chef Server Options
- `client_name` - Chef client name for authentication
- `client_key` - Path to Chef client key file
- `node_name` - Chef node name
- `validation_client_name` - Validation client name
- `validation_key` - Path to validation key file

### Git Options
- `branch` - Git branch name
- `tag` - Git tag name
- `ref` - Specific git reference (commit hash, etc.)

### Supermarket Options
- `api_key` - API key for private supermarket authentication

### Artifactory Options
- `artifactory_api_key` - Artifactory API key
- `artifactory_identity_token` - Artifactory identity token (Chef Workstation v24.11+)

## Integration with go-berkshelf

The policyfile package is designed to integrate seamlessly with the existing go-berkshelf infrastructure:

1. **Source Compatibility**: Uses the same `berkshelf.SourceLocation` type as Berksfile parsing
2. **Constraint Compatibility**: Uses the same `berkshelf.Constraint` type for version constraints
3. **Resolver Integration**: Output can be used directly with the existing dependency resolver

## Limitations

This implementation focuses only on the Berkshelf-equivalent aspects of Policyfile.rb:

- **Not Supported**: `run_list`, `named_run_list`, policy settings, attributes
- **Fully Supported**: `default_source` and `cookbook` directives with all source types and options
- **Source Types**: All major source types supported (supermarket, chef_server, git, path, artifactory)

## Testing

The package includes comprehensive tests:

```bash
go test ./pkg/policyfile -v
```

Test coverage includes:
- All supported source types and options
- Version constraint parsing
- Comment handling
- Lexer functionality
- Error cases
- Integration examples

## Architecture

The package follows the same architecture as the berksfile package:

- **Lexer** (`lexer.go`): Tokenizes Policyfile.rb input with support for symbols and key-value pairs
- **Parser** (`policyfile.y`, generated `parser.go`): Yacc-based grammar parser
- **Types**: Reuses `berkshelf.SourceLocation` and `berkshelf.Constraint`
- **Utilities** (`utils.go`): File loading and conversion helpers

## Examples

See `example_test.go` for comprehensive usage examples and `*_test.go` files for detailed test cases demonstrating all supported functionality, including the complex Chef Server authentication example you provided.
