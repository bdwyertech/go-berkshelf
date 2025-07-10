package berksfile_test

import (
	"fmt"
	"log"

	"github.com/bdwyertech/go-berkshelf/pkg/berksfile"
)

func Example_Parse() {
	// Example Berksfile content
	content := `
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

	// Parse the Berksfile
	berks, err := berksfile.Parse(content)
	if err != nil {
		log.Fatal(err)
	}

	// Print sources
	fmt.Println("Sources:")
	for _, source := range berks.Sources {
		fmt.Printf("  - %s\n", source)
	}

	// Print metadata directive
	fmt.Printf("\nHas metadata: %v\n", berks.HasMetadata)

	// Print cookbooks
	fmt.Println("\nCookbooks:")
	for _, cookbook := range berks.Cookbooks {
		fmt.Printf("  - %s", cookbook.Name)
		if cookbook.Constraint != nil {
			fmt.Printf(" (%s)", cookbook.Constraint)
		}
		if cookbook.Source != nil && cookbook.Source.Type != "" {
			sourceInfo := fmt.Sprintf(" [%s", cookbook.Source.Type)
			if cookbook.Source.URL != "" {
				sourceInfo += fmt.Sprintf(": %s", cookbook.Source.URL)
			} else if cookbook.Source.Path != "" {
				sourceInfo += fmt.Sprintf(": %s", cookbook.Source.Path)
			}
			sourceInfo += "]"
			fmt.Printf("%s", sourceInfo)
		}
		if len(cookbook.Groups) > 0 {
			fmt.Printf(" groups: %v", cookbook.Groups)
		}
		fmt.Println()
	}

	// Print groups (in sorted order for consistent output)
	fmt.Println("\nGroups:")
	groups := []string{"integration", "development"}
	for _, group := range groups {
		if cookbooks, ok := berks.Groups[group]; ok {
			fmt.Printf("  %s:\n", group)
			for _, cookbook := range cookbooks {
				fmt.Printf("    - %s\n", cookbook.Name)
			}
		}
	}

	// Output:
	// Sources:
	//   - https://supermarket.chef.io
	//
	// Has metadata: true
	//
	// Cookbooks:
	//   - nginx (~> 2.7.6)
	//   - mysql (>= 5.0.0)
	//   - postgresql (= 9.3.0)
	//   - private (>= 0.0.0) [git: git@github.com:user/private-cookbook.git]
	//   - myapp (>= 0.0.0) [path: ../myapp]
	//   - test-kitchen (>= 0.0.0) groups: [integration]
	//   - kitchen-vagrant (>= 0.0.0) groups: [integration]
	//   - chefspec (~> 4.0) groups: [development]
	//   - rubocop (>= 0.0.0) groups: [development]
	//
	// Groups:
	//   integration:
	//     - test-kitchen
	//     - kitchen-vagrant
	//   development:
	//     - chefspec
	//     - rubocop
}
