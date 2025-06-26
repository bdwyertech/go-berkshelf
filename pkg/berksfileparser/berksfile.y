// berksfile.y - GoYacc grammar for Berksfile with groups

%{

package berksfileparser

import (
    "strings"
)

// Berksfile represents a parsed Berksfile DSL root.
type Berksfile struct {
    Sources   []*Source
    Metadata  bool
    Cookbooks []*Cookbook
    Groups    []*Group
}

// Source defines a cookbook source (supermarket, chef_server, artifactory, etc).
type Source struct {
    Type    string            // e.g. "chef_server", "supermarket"
    URL     string            // e.g. "https://chef.example.com"
    Options map[string]string // e.g. user, client_key, api_key
}

// Cookbook represents a cookbook declaration.
type Cookbook struct {
    Name    string            // cookbook name
    Version string            // version constraint, e.g. "~> 1.2"
    Options map[string]string // git, branch, path, etc.
    Groups  []string          // group names, empty if none
}

// Group represents a named group block with multiple cookbooks.
type Group struct {
    Name      string
    Cookbooks []*Cookbook
}

var Result *Berksfile

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

// Collections type to hold multiple items with metadata flag
type collections struct {
    sources   []*Source
    cookbooks []*Cookbook
    groups    []*Group
    metadata  bool
}

// Statement result type
type stmtResult struct {
    source   *Source
    cookbook *Cookbook
    group    *Group
    metadata bool
}

%}

%union {
    str        string
    source     *Source
    cookbook   *Cookbook
    group      *Group
    sources    []*Source
    cookbooks  []*Cookbook
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

%%

berksfile:
    statement_list {
        Result = &Berksfile{
            Sources:   $1.sources,
            Metadata:  $1.metadata,
            Cookbooks: $1.cookbooks,
            Groups:    $1.groups,
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
        $$.cookbooks = []*Cookbook{}
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
        $$.cookbooks = []*Cookbook{}
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
        $$.cookbooks = []*Cookbook{}
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
        $$ = &Cookbook{
            Name:    $2,
            Version: $3.version,
            Options: $3.options,
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
    GROUP COLON IDENT DO group_body END {
        groupName := $3
        for _, cb := range $5 {
            cb.Groups = append(cb.Groups, groupName)
        }
        $$ = &Group{
            Name:      groupName,
            Cookbooks: $5,
        }
    }
    | GROUP COLON STRING DO group_body END {
        groupName := trimQuotes($3)
        for _, cb := range $5 {
            cb.Groups = append(cb.Groups, groupName)
        }
        $$ = &Group{
            Name:      groupName,
            Cookbooks: $5,
        }
    }
    ;

group_body:
    group_content {
        $$ = $1
    }
    | /* empty */ {
        $$ = []*Cookbook{}
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
        $$ = []*Cookbook{$1}
    }
    | NEWLINE {
        $$ = []*Cookbook{}
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