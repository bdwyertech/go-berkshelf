// policyfile.y - GoYacc grammar for Policyfile.rb (Berkshelf-equivalent directives only)

%{

package policyfile

import (
    "strings"

    "github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

// CookbookDef represents a cookbook definition in a Policyfile.rb
type CookbookDef struct {
	Name       string
	Constraint *berkshelf.Constraint
	Source     *berkshelf.SourceLocation
}

// Policyfile represents a parsed Policyfile.rb (Berkshelf-equivalent parts only)
type Policyfile struct {
	DefaultSources []*berkshelf.SourceLocation   // List of default sources
	Cookbooks      []*CookbookDef                // All cookbook definitions
}

var Result *Policyfile

// GetCookbooks returns all cookbooks
func (p *Policyfile) GetCookbooks() []*CookbookDef {
	return p.Cookbooks
}

%}

%union {
    str string
    constraint *berkshelf.Constraint
    source *berkshelf.SourceLocation
    cookbook *CookbookDef
    options map[string]string
}

%token <str> IDENTIFIER STRING SYMBOL NEWLINE COMMA COLON
%token DEFAULT_SOURCE COOKBOOK

%type <source> default_source_stmt source_spec
%type <cookbook> cookbook_stmt
%type <constraint> version_constraint
%type <str> source_type
%type <options> cookbook_options cookbook_option_list
%type <str> cookbook_option_key cookbook_option_value

%start policyfile

%%

policyfile:
    statements
    {
        if Result == nil {
            Result = &Policyfile{
                DefaultSources: []*berkshelf.SourceLocation{},
                Cookbooks:      []*CookbookDef{},
            }
        }
    }

statements:
    /* empty */
    | statements statement

statement:
    default_source_stmt
    {
        if Result == nil {
            Result = &Policyfile{
                DefaultSources: []*berkshelf.SourceLocation{},
                Cookbooks:      []*CookbookDef{},
            }
        }
        if $1 != nil {
            Result.DefaultSources = append(Result.DefaultSources, $1)
        }
    }
    | cookbook_stmt
    {
        if Result == nil {
            Result = &Policyfile{
                DefaultSources: []*berkshelf.SourceLocation{},
                Cookbooks:      []*CookbookDef{},
            }
        }
        if $1 != nil {
            Result.Cookbooks = append(Result.Cookbooks, $1)
        }
    }
    | NEWLINE
    | error NEWLINE

default_source_stmt:
    DEFAULT_SOURCE source_spec
    {
        $$ = $2
    }

source_spec:
    source_type
    {
        sourceType := strings.TrimPrefix($1, ":")
        switch sourceType {
        case "supermarket":
            $$ = &berkshelf.SourceLocation{
                Type: "supermarket",
                URL:  "https://supermarket.chef.io",
            }
        case "chef_server":
            $$ = &berkshelf.SourceLocation{
                Type: "chef_server",
            }
        case "chef_repo":
            $$ = &berkshelf.SourceLocation{
                Type: "path",
            }
        case "artifactory":
            $$ = &berkshelf.SourceLocation{
                Type: "supermarket", // Treat as supermarket-like
            }
        default:
            yylex.Error("unsupported source type: " + sourceType)
            $$ = nil
        }
    }
    | source_type COMMA STRING
    {
        sourceType := strings.TrimPrefix($1, ":")
        uri := strings.Trim($3, "\"'")
        
        switch sourceType {
        case "supermarket":
            $$ = &berkshelf.SourceLocation{
                Type: "supermarket",
                URL:  uri,
            }
        case "chef_server":
            $$ = &berkshelf.SourceLocation{
                Type: "chef_server",
                URL:  uri,
            }
        case "chef_repo":
            $$ = &berkshelf.SourceLocation{
                Type: "path",
                Path: uri,
            }
        case "artifactory":
            $$ = &berkshelf.SourceLocation{
                Type: "supermarket", // Treat as supermarket-like
                URL:  uri,
            }
        default:
            yylex.Error("unsupported source type: " + sourceType)
            $$ = nil
        }
    }

source_type:
    SYMBOL
    {
        $$ = $1
    }

cookbook_stmt:
    COOKBOOK STRING
    {
        name := strings.Trim($2, "\"'")
        $$ = &CookbookDef{
            Name: name,
        }
    }
    | COOKBOOK STRING COMMA version_constraint
    {
        name := strings.Trim($2, "\"'")
        $$ = &CookbookDef{
            Name:       name,
            Constraint: $4,
        }
    }
    | COOKBOOK STRING COMMA cookbook_options
    {
        name := strings.Trim($2, "\"'")
        source := createSourceFromOptions($4)
        $$ = &CookbookDef{
            Name:   name,
            Source: source,
        }
    }
    | COOKBOOK STRING COMMA version_constraint COMMA cookbook_options
    {
        name := strings.Trim($2, "\"'")
        source := createSourceFromOptions($6)
        $$ = &CookbookDef{
            Name:       name,
            Constraint: $4,
            Source:     source,
        }
    }

cookbook_options:
    cookbook_option_list
    {
        $$ = $1
    }

cookbook_option_list:
    cookbook_option_key COLON cookbook_option_value
    {
        $$ = map[string]string{$1: $3}
    }
    | cookbook_option_list COMMA cookbook_option_key COLON cookbook_option_value
    {
        $1[$3] = $5
        $$ = $1
    }

cookbook_option_key:
    IDENTIFIER
    {
        $$ = $1
    }

cookbook_option_value:
    STRING
    {
        $$ = strings.Trim($1, "\"'")
    }

version_constraint:
    STRING
    {
        constraintStr := strings.Trim($1, "\"'")
        constraint, err := berkshelf.NewConstraint(constraintStr)
        if err != nil {
            yylex.Error("invalid version constraint: " + constraintStr)
            $$ = nil
        } else {
            $$ = constraint
        }
    }

%%

// createSourceFromOptions creates a SourceLocation from cookbook options
func createSourceFromOptions(options map[string]string) *berkshelf.SourceLocation {
    if options == nil {
        return nil
    }

    // Handle different source types
    if path, ok := options["path"]; ok {
        return &berkshelf.SourceLocation{
            Type: "path",
            Path: path,
        }
    }

    if gitURL, ok := options["git"]; ok {
        source := &berkshelf.SourceLocation{
            Type: "git",
            URL:  gitURL,
        }
        
        // Add git-specific options
        if branch, ok := options["branch"]; ok {
            source.Ref = branch
        }
        if tag, ok := options["tag"]; ok {
            source.Ref = tag
        }
        if ref, ok := options["ref"]; ok {
            source.Ref = ref
        }
        
        return source
    }

    if github, ok := options["github"]; ok {
        source := &berkshelf.SourceLocation{
            Type: "git",
            URL:  "https://github.com/" + github + ".git",
        }
        
        // Add git-specific options
        if branch, ok := options["branch"]; ok {
            source.Ref = branch
        }
        if tag, ok := options["tag"]; ok {
            source.Ref = tag
        }
        if ref, ok := options["ref"]; ok {
            source.Ref = ref
        }
        
        return source
    }

    if chefServerURL, ok := options["chef_server"]; ok {
        source := &berkshelf.SourceLocation{
            Type: "chef_server",
            URL:  chefServerURL,
        }
        
        // Add chef server specific options to Options map
        if source.Options == nil {
            source.Options = make(map[string]any)
        }
        
        if clientName, ok := options["client_name"]; ok {
            source.Options["client_name"] = clientName
        }
        if clientKey, ok := options["client_key"]; ok {
            source.Options["client_key"] = clientKey
        }
        if nodeNameOpt, ok := options["node_name"]; ok {
            source.Options["node_name"] = nodeNameOpt
        }
        if validation_client_name, ok := options["validation_client_name"]; ok {
            source.Options["validation_client_name"] = validation_client_name
        }
        if validation_key, ok := options["validation_key"]; ok {
            source.Options["validation_key"] = validation_key
        }
        
        return source
    }

    if supermarketURL, ok := options["supermarket"]; ok {
        source := &berkshelf.SourceLocation{
            Type: "supermarket",
            URL:  supermarketURL,
        }
        
        // Add supermarket-specific options
        if source.Options == nil {
            source.Options = make(map[string]any)
        }
        
        if apiKey, ok := options["api_key"]; ok {
            source.Options["api_key"] = apiKey
        }
        
        return source
    }

    if artifactoryURL, ok := options["artifactory"]; ok {
        source := &berkshelf.SourceLocation{
            Type: "supermarket", // Artifactory is treated as supermarket-like
            URL:  artifactoryURL,
        }
        
        // Add artifactory-specific options
        if source.Options == nil {
            source.Options = make(map[string]any)
        }
        
        if apiKey, ok := options["artifactory_api_key"]; ok {
            source.Options["artifactory_api_key"] = apiKey
        }
        if identityToken, ok := options["artifactory_identity_token"]; ok {
            source.Options["artifactory_identity_token"] = identityToken
        }
        
        return source
    }

    // Handle other source types as needed
    return nil
}
