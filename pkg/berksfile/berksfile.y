// berksfile.y - GoYacc grammar for Berksfile with groups

%{

package berksfile

import (
    "strings"

    "github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

// CookbookDef represents a cookbook definition in a Berksfile
type CookbookDef struct {
	Name       string
	Constraint *berkshelf.Constraint
	Source     *berkshelf.SourceLocation
	Groups     []string
}

// Berksfile represents a parsed Berksfile
type Berksfile struct {
	Sources     []string                  // List of default sources
	Cookbooks   []*CookbookDef            // All cookbook definitions
	Groups      map[string][]*CookbookDef // Grouped cookbooks
	HasMetadata bool                      // Whether metadata directive is present
}

var Result *Berksfile

// GetCookbooks returns all cookbooks, optionally filtered by groups
func (b *Berksfile) GetCookbooks(groups ...string) []*CookbookDef {
	if len(groups) == 0 {
		return b.Cookbooks
	}

	// Create a map to avoid duplicates
	cookbookMap := make(map[string]*CookbookDef)

	for _, group := range groups {
		if groupCookbooks, ok := b.Groups[group]; ok {
			for _, cookbook := range groupCookbooks {
				cookbookMap[cookbook.Name] = cookbook
			}
		}
	}

	// Convert map to slice
	result := make([]*CookbookDef, 0, len(cookbookMap))
	for _, cookbook := range cookbookMap {
		result = append(result, cookbook)
	}

	return result
}

// GetCookbook returns a specific cookbook by name
func (b *Berksfile) GetCookbook(name string) *CookbookDef {
	for _, cookbook := range b.Cookbooks {
		if cookbook.Name == name {
			return cookbook
		}
	}
	return nil
}

// HasGroup checks if a group exists
func (b *Berksfile) HasGroup(name string) bool {
	_, ok := b.Groups[name]
	return ok
}

// GetGroups returns all group names
func (b *Berksfile) GetGroups() []string {
	groups := make([]string, 0, len(b.Groups))
	for group := range b.Groups {
		groups = append(groups, group)
	}
	return groups
}

// ParseConstraint parses a version constraint string
func ParseConstraint(constraintStr string) (*berkshelf.Constraint, error) {
	return berkshelf.NewConstraint(constraintStr)
}

func trimQuotes(s string) string {
    return strings.Trim(s, `"'`)
}

// Intermediate types for semantic values
type sourceArgs struct {
    typ  string
    url  string
    opts map[string]string
}

type cbTail struct {
    version string
    options map[string]string
}

type kv struct {
    key   string
    value string
}

// Source represents a source definition in a Berksfile
type Source struct {
    Type    string
    URL     string
    Options map[string]string
}

// Group represents a group definition in a Berksfile
type Group struct {
    Name      string
    Cookbooks []*CookbookDef
}

// Collections type to hold multiple items with metadata flag
type collections struct {
    sources   []*Source
    cookbooks []*CookbookDef
    groups    []*Group
    metadata  bool
}

// Statement result type
type stmtResult struct {
    source   *Source
    cookbook *CookbookDef
    group    *Group
    metadata bool
}

%}

%union {
    str        string
    source     *Source
    cookbook   *CookbookDef
    group      *Group
    sources    []*Source
    cookbooks  []*CookbookDef
    groups     []*Group
    opts       map[string]string
    sa         sourceArgs
    cbTail     cbTail
    kv         kv
    boolVal    bool
    collections collections
    stmt       stmtResult
}

// Tokens
%token <str> SOURCE METADATA COOKBOOK GROUP DO END IDENT STRING COLON COMMA LBRACE RBRACE HASHROCKET NEWLINE

// Type declarations for non-terminals
%type <collections> berksfile statement_list non_empty_statement_list
%type <stmt> statement
%type <source> source_stmt
%type <sa> source_args
%type <boolVal> metadata_stmt
%type <cookbook> cookbook_stmt
%type <str> cookbook_name
%type <cbTail> cookbook_tail
%type <group> group_stmt
%type <cookbooks> group_body group_content
%type <opts> hash_pairs hash_pairs_tail
%type <kv> hash_pair
%type <sources> group_names

%%

berksfile:
    statement_list {
        // Convert sources from []*Source to []string
        sources := make([]string, len($1.sources))
        for i, src := range $1.sources {
            sources[i] = src.URL
        }
        
        // Collect all cookbooks (both standalone and from groups)
        allCookbooks := make([]*CookbookDef, 0, len($1.cookbooks))
        allCookbooks = append(allCookbooks, $1.cookbooks...)
        
        // Convert groups from []*Group to map[string][]*CookbookDef
        groups := make(map[string][]*CookbookDef)
        for _, group := range $1.groups {
            // Add group cookbooks to the main cookbook list
            allCookbooks = append(allCookbooks, group.Cookbooks...)
            
            // Handle group names (could be comma-separated for multiple groups)
            groupNames := strings.Split(group.Name, ",")
            
            for _, groupName := range groupNames {
                groupName = strings.TrimSpace(groupName)
                if groups[groupName] == nil {
                    groups[groupName] = []*CookbookDef{}
                }
                
                // Add cookbooks to this group
                for _, cb := range group.Cookbooks {
                    // Check if cookbook already exists in this group
                    found := false
                    for _, existing := range groups[groupName] {
                        if existing.Name == cb.Name {
                            found = true
                            break
                        }
                    }
                    if !found {
                        groups[groupName] = append(groups[groupName], cb)
                    }
                }
            }
        }
        
        Result = &Berksfile{
            Sources:     sources,
            Cookbooks:   allCookbooks,
            Groups:      groups,
            HasMetadata: $1.metadata,
        }
        $$ = $1
    }
    ;

statement_list:
    non_empty_statement_list {
        $$ = $1
    }
    | /* empty */ {
        $$.sources = []*Source{}
        $$.cookbooks = []*CookbookDef{}
        $$.groups = []*Group{}
        $$.metadata = false
    }
    ;

non_empty_statement_list:
    non_empty_statement_list statement {
        $$.sources = $1.sources
        $$.cookbooks = $1.cookbooks
        $$.groups = $1.groups
        $$.metadata = $1.metadata
        
        // Add new statement
        if $2.source != nil {
            $$.sources = append($$.sources, $2.source)
        }
        if $2.cookbook != nil {
            $$.cookbooks = append($$.cookbooks, $2.cookbook)
        }
        if $2.group != nil {
            $$.groups = append($$.groups, $2.group)
        }
        if $2.metadata {
            $$.metadata = true
        }
    }
    | non_empty_statement_list NEWLINE {
        $$ = $1
    }
    | statement {
        $$.sources = []*Source{}
        $$.cookbooks = []*CookbookDef{}
        $$.groups = []*Group{}
        $$.metadata = false
        
        // Add the statement
        if $1.source != nil {
            $$.sources = append($$.sources, $1.source)
        }
        if $1.cookbook != nil {
            $$.cookbooks = append($$.cookbooks, $1.cookbook)
        }
        if $1.group != nil {
            $$.groups = append($$.groups, $1.group)
        }
        if $1.metadata {
            $$.metadata = true
        }
    }
    | NEWLINE {
        $$.sources = []*Source{}
        $$.cookbooks = []*CookbookDef{}
        $$.groups = []*Group{}
        $$.metadata = false
    }
    ;

statement:
    source_stmt {
        $$.source = $1
        $$.cookbook = nil
        $$.group = nil
        $$.metadata = false
    }
    | metadata_stmt {
        $$.source = nil
        $$.cookbook = nil
        $$.group = nil
        $$.metadata = $1
    }
    | cookbook_stmt {
        $$.source = nil
        $$.cookbook = $1
        $$.group = nil
        $$.metadata = false
    }
    | group_stmt {
        $$.source = nil
        $$.cookbook = nil
        $$.group = $1
        $$.metadata = false
    }
    ;

source_stmt:
    SOURCE source_args {
        $$ = &Source{
            Type:    $2.typ,
            URL:     $2.url,
            Options: $2.opts,
        }
    }
    ;

source_args:
    STRING {
        $$.typ = "supermarket"
        $$.url = trimQuotes($1)
        $$.opts = nil
    }
    | IDENT COLON STRING {
        $$.typ = $1
        $$.url = trimQuotes($3)
        $$.opts = nil
    }
    | IDENT COLON STRING COMMA hash_pairs {
        $$.typ = $1
        $$.url = trimQuotes($3)
        $$.opts = $5
    }
    ;

metadata_stmt:
    METADATA {
        $$ = true
    }
    ;

cookbook_stmt:
    COOKBOOK cookbook_name cookbook_tail {
        constraint, _ := ParseConstraint(">= 0.0.0")
        if $3.version != "" {
            if c, err := ParseConstraint($3.version); err != nil {
                yylex.Error("invalid version constraint: " + $3.version)
                return 1
            } else {
                constraint = c
            }
        }
        
        source := &berkshelf.SourceLocation{}
        if $3.options != nil {
            source.Options = make(map[string]any)
            if gitUrl, ok := $3.options["git"]; ok {
                source.Type = "git"
                source.URL = gitUrl
                if branch, ok := $3.options["branch"]; ok {
                    source.Ref = branch
                    source.Options["branch"] = branch
                }
                if ref, ok := $3.options["ref"]; ok {
                    source.Ref = ref
                    source.Options["ref"] = ref
                }
            } else if github, ok := $3.options["github"]; ok {
                source.Type = "git"
                source.URL = "https://github.com/" + github + ".git"
            } else if path, ok := $3.options["path"]; ok {
                source.Type = "path"
                source.Path = path
            }
        }
        
        $$ = &CookbookDef{
            Name:       $2,
            Constraint: constraint,
            Source:     source,
            Groups:     []string{},
        }
    }
    ;

cookbook_name:
    STRING { $$ = trimQuotes($1) }
    | IDENT { $$ = $1 }
    ;

cookbook_tail:
    COMMA STRING {
        $$.version = trimQuotes($2)
        $$.options = nil
    }
    | COMMA LBRACE hash_pairs RBRACE {
        $$.version = ""
        $$.options = $3
    }
    | COMMA STRING COMMA LBRACE hash_pairs RBRACE {
        $$.version = trimQuotes($2)
        $$.options = $5
    }
    | COMMA hash_pairs {
        $$.version = ""
        $$.options = $2
    }
    | COMMA STRING COMMA hash_pairs {
        $$.version = trimQuotes($2)
        $$.options = $4
    }
    | /* empty */ {
        $$.version = ""
        $$.options = nil
    }
    ;

group_stmt:
    GROUP group_names DO group_body END {
        // For multiple groups, we need to create separate Group entries
        // but the cookbooks will be shared across groups
        groupNames := make([]string, len($2))
        for i, src := range $2 {
            groupNames[i] = src.URL // We're reusing Source.URL to store group names
        }
        
        // Add group names to each cookbook
        for _, cb := range $4 {
            cb.Groups = append(cb.Groups, groupNames...)
        }
        
        // For empty groups, we still need to return a Group object with the first name
        // The berksfile rule will handle creating entries for all group names
        groupName := groupNames[0]
        if len(groupNames) > 1 {
            // For multiple groups, we need to create a special marker
            // The berksfile rule will expand this into multiple group entries
            groupName = strings.Join(groupNames, ",")
        }
        
        $$ = &Group{
            Name:      groupName,
            Cookbooks: $4,
        }
    }
    ;

group_names:
    group_names COMMA COLON IDENT {
        $$ = append($1, &Source{URL: $4})
    }
    | group_names COMMA COLON STRING {
        $$ = append($1, &Source{URL: trimQuotes($4)})
    }
    | IDENT {
        $$ = []*Source{{URL: $1}}
    }
    | STRING {
        $$ = []*Source{{URL: trimQuotes($1)}}
    }
    | COLON IDENT {
        $$ = []*Source{{URL: $2}}
    }
    | COLON STRING {
        $$ = []*Source{{URL: trimQuotes($2)}}
    }
    ;

group_body:
    group_content {
        $$ = $1
    }
    | /* empty */ {
        $$ = []*CookbookDef{}
    }
    ;

group_content:
    group_content cookbook_stmt {
        $$ = append($1, $2)
    }
    | group_content NEWLINE {
        $$ = $1
    }
    | cookbook_stmt {
        $$ = []*CookbookDef{$1}
    }
    | NEWLINE {
        $$ = []*CookbookDef{}
    }
    ;

hash_pairs:
    hash_pair hash_pairs_tail {
        m := map[string]string{$1.key: $1.value}
        for k, v := range $2 {
            m[k] = v
        }
        $$ = m
    }
    ;

hash_pairs_tail:
    COMMA hash_pair hash_pairs_tail {
        m := map[string]string{$2.key: $2.value}
        for k, v := range $3 {
            m[k] = v
        }
        $$ = m
    }
    | /* empty */ {
        $$ = map[string]string{}
    }
    ;

hash_pair:
    IDENT COLON STRING {
        $$.key = $1
        $$.value = trimQuotes($3)
    }
    | COLON IDENT HASHROCKET STRING {
        $$.key = $2
        $$.value = trimQuotes($4)
    }
    | STRING HASHROCKET STRING {
        $$.key = trimQuotes($1)
        $$.value = trimQuotes($3)
    }
    ;

%%