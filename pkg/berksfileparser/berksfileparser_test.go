package berksfileparser_test

import (
	"testing"

	"github.com/bdwyer/go-berkshelf/pkg/berksfileparser"
)

// func TestTokens(t *testing.T) {
// 	src := `source chef_server: "https://chef.myorg.net/", client_name: "bd"`
// 	berksfile, err := berksfileparser.ParseBerksfile(src)
// 	if err != nil {
// 		t.Fatalf("ParseBerksfile failed: %v", err)
// 	}
//
// 	out, err := json.MarshalIndent(berksfile, "", "  ")
// 	if err != nil {
// 		t.Fatalf("json.MarshalIndent failed: %v", err)
// 	}
//
// 	fmt.Println(string(out))
// }

// func TestParseBerksfilePrettyJSON(t *testing.T) {
// 	input := `# source 'https://supermarket.chef.io'
// source 'https://supermarket.chef.io'
//
// source chef_server: "https://chef.myorg.net/", client_name: "bd", client_key: "~/.chef/me.pem"
//
// # source supermarket: "https://supermarket.chef.io"
// cookbook 'nginx', "~> 1.2"
//
// group :base do
//   cookbook 'aws'
// end
//
// group :production do
//   cookbook 'aws'
//   # cookbook 'wildfly', '~> 1.2'
// end
//
// cookbook 'nginx', "~> 1.2"
// cookbook 'windows-security-policy', git: 'https://github.com/bdwyertech/windows-security-policy.git', branch: 'chef-16'
// `
// 	berksfile, err := berksfileparser.ParseBerksfile(input)
// 	if err != nil {
// 		t.Fatalf("ParseBerksfile failed: %v", err)
// 	}
//
// 	// fmt.Println(berksfile.Cookbooks)
//
// 	out, err := json.MarshalIndent(berksfile, "", "  ")
// 	if err != nil {
// 		t.Fatalf("json.MarshalIndent failed: %v", err)
// 	}
//
// 	fmt.Println(string(out))
// }

// Additional tests for Berksfile parser

func TestParseBerksfile_Empty(t *testing.T) {
	input := ``
	berksfile, err := berksfileparser.ParseBerksfile(input)
	if err != nil {
		t.Fatalf("Expected success for empty input, got error: %v", err)
	}
	if berksfile == nil {
		t.Fatalf("Expected valid Berksfile for empty input, got nil")
	}
	if len(berksfile.Sources) != 0 || len(berksfile.Cookbooks) != 0 || len(berksfile.Groups) != 0 {
		t.Fatalf("Expected empty Berksfile, got sources=%d, cookbooks=%d, groups=%d",
			len(berksfile.Sources), len(berksfile.Cookbooks), len(berksfile.Groups))
	}
}

func TestParseBerksfile_InvalidSyntax(t *testing.T) {
	input := `cookbook "nginx" "1.2" { invalid_syntax }`
	_, err := berksfileparser.ParseBerksfile(input)
	if err == nil {
		t.Fatalf("Expected error for invalid syntax, got nil")
	}
}

func TestParseBerksfile_GroupEdgeCases(t *testing.T) {
	input := `
group :web do
end
`
	berksfile, err := berksfileparser.ParseBerksfile(input)
	if err != nil {
		t.Fatalf("ParseBerksfile failed: %v", err)
	}
	if len(berksfile.Groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(berksfile.Groups))
	}
	webGroup, exists := berksfile.Groups["web"]
	if !exists {
		t.Fatalf("Expected 'web' group to exist")
	}
	if len(webGroup) != 0 {
		t.Fatalf("Expected empty web group, got %d cookbooks", len(webGroup))
	}
}

func TestParseBerksfile_CommentsAndWhitespace(t *testing.T) {
	input := `
# This is a comment
cookbook "nginx"
   
# Another comment
cookbook "redis"
`
	berksfile, err := berksfileparser.ParseBerksfile(input)
	if err != nil {
		t.Fatalf("ParseBerksfile failed: %v", err)
	}
	if berksfile == nil {
		t.Fatalf("ParseBerksfile returned nil result")
	}
	if len(berksfile.Cookbooks) != 2 {
		t.Fatalf("Expected 2 cookbooks, got %d", len(berksfile.Cookbooks))
	}
}
