package policyfile

import (
	"fmt"
	"log"
)

func ExampleParsePolicyfile() {
	input := `
# Example Policyfile.rb with all supported Berkshelf-equivalent directives
default_source :supermarket
default_source :chef_repo, "/path/to/local/cookbooks"

# Basic cookbook declarations
cookbook "nginx", "~> 2.7"
cookbook "mysql"
cookbook "apache2", ">= 1.0.0"

# Path source
cookbook "my_app", path: "cookbooks/my_app"

# Git sources
cookbook "chef-ingredient", git: "https://github.com/chef-cookbooks/chef-ingredient.git", tag: "v0.12.0"
cookbook "mysql-custom", github: "opscode-cookbooks/mysql", branch: "master"

# Chef Server source with authentication
cookbook "windows-security-policy", chef_server: "https://chef.my.org/", client_name: "myself", client_key: "~/.chef/key.pem"

# Private Supermarket with API key
cookbook "private-cookbook", supermarket: "https://private.supermarket.com", api_key: "my-api-key"

# Artifactory source
cookbook "artifactory-cookbook", artifactory: "https://artifactory.example/api/chef/my-supermarket", artifactory_api_key: "my-artifactory-key"
`

	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Default Sources: %d\n", len(policyfile.DefaultSources))
	for i, source := range policyfile.DefaultSources {
		fmt.Printf("  Source %d: %s", i+1, source.Type)
		if source.URL != "" {
			fmt.Printf(" (%s)", source.URL)
		}
		if source.Path != "" {
			fmt.Printf(" (%s)", source.Path)
		}
		fmt.Println()
	}

	fmt.Printf("Cookbooks: %d\n", len(policyfile.Cookbooks))
	for i, cookbook := range policyfile.Cookbooks {
		fmt.Printf("  Cookbook %d: %s", i+1, cookbook.Name)
		if cookbook.Constraint != nil {
			fmt.Printf(" %s", cookbook.Constraint.String())
		}
		if cookbook.Source != nil {
			fmt.Printf(" [%s", cookbook.Source.Type)
			if cookbook.Source.URL != "" {
				fmt.Printf(": %s", cookbook.Source.URL)
			}
			if cookbook.Source.Path != "" {
				fmt.Printf(": %s", cookbook.Source.Path)
			}
			if cookbook.Source.Ref != "" {
				fmt.Printf(" @ %s", cookbook.Source.Ref)
			}
			if len(cookbook.Source.Options) > 0 {
				fmt.Printf(" +opts")
			}
			fmt.Printf("]")
		}
		fmt.Println()
	}

	// Output:
	// Default Sources: 2
	//   Source 1: supermarket (https://supermarket.chef.io)
	//   Source 2: path (/path/to/local/cookbooks)
	// Cookbooks: 9
	//   Cookbook 1: nginx ~> 2.7
	//   Cookbook 2: mysql
	//   Cookbook 3: apache2 >= 1.0.0
	//   Cookbook 4: my_app [path: cookbooks/my_app]
	//   Cookbook 5: chef-ingredient [git: https://github.com/chef-cookbooks/chef-ingredient.git @ v0.12.0]
	//   Cookbook 6: mysql-custom [git: https://github.com/opscode-cookbooks/mysql.git @ master]
	//   Cookbook 7: windows-security-policy [chef_server: https://chef.my.org/ +opts]
	//   Cookbook 8: private-cookbook [supermarket: https://private.supermarket.com +opts]
	//   Cookbook 9: artifactory-cookbook [supermarket: https://artifactory.example/api/chef/my-supermarket +opts]
}

func ExampleLoadPolicyfile() {
	// This example shows how to load a Policyfile.rb from disk
	// Note: This would require an actual file to exist

	// policyfile, err := LoadPolicyfile("./Policyfile.rb")
	// if err != nil {
	//     log.Fatal(err)
	// }

	// fmt.Printf("Loaded %d default sources and %d cookbooks\n",
	//     len(policyfile.DefaultSources), len(policyfile.Cookbooks))

	fmt.Println("Example of loading Policyfile.rb from disk")
	// Output: Example of loading Policyfile.rb from disk
}

func ExamplePolicyfile_ToBerksfileEquivalent() {
	input := `
default_source :supermarket
cookbook "nginx", "~> 2.7"
cookbook "my_app", path: "cookbooks/my_app"
`

	policyfile, err := ParsePolicyfile(input)
	if err != nil {
		log.Fatal(err)
	}

	equivalent, err := policyfile.ToBerksfileEquivalent()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Berkshelf-equivalent sources: %d\n", len(equivalent.Sources))
	fmt.Printf("Berkshelf-equivalent cookbooks: %d\n", len(equivalent.Cookbooks))

	// Output:
	// Berkshelf-equivalent sources: 1
	// Berkshelf-equivalent cookbooks: 2
}
