package berksfile_test

import (
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bdwyertech/go-berkshelf/pkg/berksfile"
)

var _ = Describe("Parse basic Berksfile", func() {
	It("should parse an empty berksfile", func() {
		b, err := berksfile.Parse("")
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Sources).To(BeEmpty())
		Expect(b.Cookbooks).To(BeEmpty())
	})

	It("should parse a source declaration", func() {
		b, err := berksfile.Parse(`source 'https://supermarket.chef.io'`)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Sources).To(HaveLen(1))
		Expect(b.Sources[0].Type).To(Equal("supermarket"))
		Expect(b.Sources[0].URL).To(Equal("https://supermarket.chef.io"))
	})

	It("should parse a metadata directive", func() {
		b, err := berksfile.Parse(`metadata`)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.HasMetadata).To(BeTrue())
	})

	It("should parse a simple cookbook", func() {
		b, err := berksfile.Parse(`cookbook 'nginx'`)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Cookbooks).To(HaveLen(1))
		Expect(b.Cookbooks[0].Name).To(Equal("nginx"))
		Expect(b.Cookbooks[0].Constraint).NotTo(BeNil())
		Expect(b.Cookbooks[0].Constraint.String()).To(Equal(">= 0.0.0"))
	})

	It("should parse a cookbook with version", func() {
		b, err := berksfile.Parse(`cookbook 'nginx', '~> 2.7.6'`)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Cookbooks).To(HaveLen(1))
		Expect(b.Cookbooks[0].Name).To(Equal("nginx"))
		Expect(b.Cookbooks[0].Constraint).NotTo(BeNil())
		Expect(b.Cookbooks[0].Constraint.String()).To(Equal("~> 2.7.6"))
	})

	It("should parse a cookbook with git source", func() {
		b, err := berksfile.Parse(`cookbook 'private', git: 'git@github.com:user/repo.git'`)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Cookbooks).To(HaveLen(1))
		cb := b.Cookbooks[0]
		Expect(cb.Name).To(Equal("private"))
		Expect(cb.Source).NotTo(BeNil())
		Expect(cb.Source.Type).To(Equal("git"))
		Expect(cb.Source.URL).To(Equal("git@github.com:user/repo.git"))
	})

	It("should parse a cookbook with github shorthand", func() {
		b, err := berksfile.Parse(`cookbook 'private', github: 'user/repo'`)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Cookbooks).To(HaveLen(1))
		cb := b.Cookbooks[0]
		Expect(cb.Source).NotTo(BeNil())
		Expect(cb.Source.Type).To(Equal("git"))
		Expect(cb.Source.URL).To(Equal("https://github.com/user/repo.git"))
	})

	It("should parse a cookbook with path source", func() {
		b, err := berksfile.Parse(`cookbook 'myapp', path: '../myapp'`)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Cookbooks).To(HaveLen(1))
		cb := b.Cookbooks[0]
		Expect(cb.Source).NotTo(BeNil())
		Expect(cb.Source.Type).To(Equal("path"))
		Expect(cb.Source.Path).To(Equal("../myapp"))
	})

	It("should parse a cookbook with git source and branch", func() {
		b, err := berksfile.Parse(`cookbook 'private', git: 'git@github.com:user/repo.git', branch: 'develop'`)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Cookbooks).To(HaveLen(1))
		cb := b.Cookbooks[0]
		Expect(cb.Source).NotTo(BeNil())
		Expect(cb.Source.Options["branch"]).To(Equal("develop"))
	})
})

var _ = Describe("Parse groups", func() {
	It("should parse groups with cookbooks", func() {
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
		b, err := berksfile.Parse(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Cookbooks).To(HaveLen(4))
		Expect(b.Groups).To(HaveLen(3))

		testGroup, ok := b.Groups["test"]
		Expect(ok).To(BeTrue())
		Expect(testGroup).To(HaveLen(2)) // minitest-handler and chefspec

		integrationGroup, ok := b.Groups["integration"]
		Expect(ok).To(BeTrue())
		Expect(integrationGroup).To(HaveLen(2))

		devGroup, ok := b.Groups["development"]
		Expect(ok).To(BeTrue())
		Expect(devGroup).To(HaveLen(1))
	})

	It("should assign multi-group membership correctly", func() {
		input := `
group :development, :test do
  cookbook 'chefspec'
end
`
		b, err := berksfile.Parse(input)
		Expect(err).NotTo(HaveOccurred())

		chefspec := b.GetCookbook("chefspec")
		Expect(chefspec).NotTo(BeNil())
		Expect(chefspec.Groups).To(HaveLen(2))
	})
})

var _ = Describe("Parse complete Berksfile", func() {
	It("should parse a complete Berksfile", func() {
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
		b, err := berksfile.Parse(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Sources).To(HaveLen(1))
		Expect(b.HasMetadata).To(BeTrue())
		Expect(b.Cookbooks).To(HaveLen(9))

		nginx := b.GetCookbook("nginx")
		Expect(nginx).NotTo(BeNil())
		Expect(nginx.Constraint).NotTo(BeNil())
		Expect(nginx.Constraint.String()).To(Equal("~> 2.7.6"))

		private := b.GetCookbook("private")
		Expect(private).NotTo(BeNil())
		Expect(private.Source).NotTo(BeNil())
		Expect(private.Source.Type).To(Equal("git"))
		Expect(private.Source.Options["branch"]).To(Equal("master"))

		Expect(b.Groups).To(HaveLen(2))

		integrationCookbooks := b.GetCookbooks("integration")
		Expect(integrationCookbooks).To(HaveLen(2))

		developmentCookbooks := b.GetCookbooks("development")
		Expect(developmentCookbooks).To(HaveLen(2))
	})
})

var _ = Describe("Parse error handling", func() {
	DescribeTable("should handle parse errors",
		func(input string, shouldError bool, errorMsg string) {
			_, err := berksfile.Parse(input)
			if shouldError {
				Expect(err).To(HaveOccurred())
				if errorMsg != "" {
					Expect(err.Error()).To(ContainSubstring(errorMsg))
				}
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("invalid constraint", `cookbook 'nginx', 'invalid constraint'`, true, "invalid version constraint"),
		Entry("missing cookbook name", `cookbook`, true, "expected cookbook name"),
		Entry("missing source URL", `source`, true, "expected string after 'source'"),
		Entry("incomplete group", `group :test`, true, "unexpected token EOF in group"),
		Entry("unterminated group", "group :test do\n\t\tcookbook 'test'", true, "unexpected token EOF in group"),
	)
})

// Tests converted from berksfileparser_test.go
var _ = Describe("Parse Berksfile edge cases", func() {
	It("should parse empty input", func() {
		b, err := berksfile.Parse(``)
		Expect(err).NotTo(HaveOccurred())
		Expect(b).NotTo(BeNil())
		Expect(b.Sources).To(BeEmpty())
		Expect(b.Cookbooks).To(BeEmpty())
		Expect(b.Groups).To(BeEmpty())
	})

	It("should error on invalid syntax", func() {
		input := `cookbook "nginx" "1.2" { invalid_syntax }`
		_, err := berksfile.Parse(input)
		Expect(err).To(HaveOccurred())
	})

	It("should handle empty group", func() {
		input := `
group :web do
end
`
		b, err := berksfile.Parse(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Groups).To(HaveLen(1))
		webGroup, exists := b.Groups["web"]
		Expect(exists).To(BeTrue())
		Expect(webGroup).To(BeEmpty())
	})

	It("should handle comments and whitespace", func() {
		input := `
# This is a comment
cookbook "nginx"
   
# Another comment
cookbook "redis"
`
		b, err := berksfile.Parse(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(b).NotTo(BeNil())
		Expect(b.Cookbooks).To(HaveLen(2))
	})
})

// Tests converted from lexer_test.go — rewritten to use berksfile.Parse() public API
var _ = Describe("Lexer via Parse", func() {
	Context("basic token handling", func() {
		It("should parse a source declaration correctly", func() {
			b, err := berksfile.Parse("source 'https://supermarket.chef.io'")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Sources).To(HaveLen(1))
			Expect(b.Sources[0].URL).To(Equal("https://supermarket.chef.io"))
		})

		It("should parse a cookbook with version correctly", func() {
			b, err := berksfile.Parse("cookbook 'nginx', '~> 2.7.6'")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Cookbooks).To(HaveLen(1))
			Expect(b.Cookbooks[0].Name).To(Equal("nginx"))
			Expect(b.Cookbooks[0].Constraint.String()).To(Equal("~> 2.7.6"))
		})

		It("should parse a metadata directive correctly", func() {
			b, err := berksfile.Parse("metadata")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.HasMetadata).To(BeTrue())
		})

		It("should parse a group block correctly", func() {
			b, err := berksfile.Parse("group :test do\nend")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Groups).To(HaveKey("test"))
		})

		It("should parse a cookbook with git source correctly", func() {
			b, err := berksfile.Parse("cookbook 'private', git: 'git@github.com:user/repo.git'")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Cookbooks).To(HaveLen(1))
			Expect(b.Cookbooks[0].Source.Type).To(Equal("git"))
			Expect(b.Cookbooks[0].Source.URL).To(Equal("git@github.com:user/repo.git"))
		})

		It("should parse a cookbook with hash options correctly", func() {
			b, err := berksfile.Parse("cookbook 'test', { git: 'repo.git', branch: 'master' }")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Cookbooks).To(HaveLen(1))
			Expect(b.Cookbooks[0].Source.Type).To(Equal("git"))
			Expect(b.Cookbooks[0].Source.URL).To(Equal("repo.git"))
			Expect(b.Cookbooks[0].Source.Options["branch"]).To(Equal("master"))
		})

		It("should parse hashrocket syntax correctly", func() {
			b, err := berksfile.Parse("cookbook 'test', :git => 'repo.git'")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Cookbooks).To(HaveLen(1))
			Expect(b.Cookbooks[0].Source.Type).To(Equal("git"))
			Expect(b.Cookbooks[0].Source.URL).To(Equal("repo.git"))
		})

		It("should parse multiple lines with newlines correctly", func() {
			b, err := berksfile.Parse("source 'https://supermarket.chef.io'\n\ncookbook 'nginx'")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Sources).To(HaveLen(1))
			Expect(b.Cookbooks).To(HaveLen(1))
			Expect(b.Cookbooks[0].Name).To(Equal("nginx"))
		})
	})

	Context("string handling", func() {
		It("should handle double quoted strings", func() {
			b, err := berksfile.Parse(`cookbook "nginx"`)
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Cookbooks).To(HaveLen(1))
			Expect(b.Cookbooks[0].Name).To(Equal("nginx"))
		})

		It("should handle single quoted strings", func() {
			b, err := berksfile.Parse(`cookbook 'nginx'`)
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Cookbooks).To(HaveLen(1))
			Expect(b.Cookbooks[0].Name).To(Equal("nginx"))
		})

		It("should handle mixed quotes", func() {
			b, err := berksfile.Parse(`cookbook "nginx", '~> 2.7'`)
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Cookbooks).To(HaveLen(1))
			Expect(b.Cookbooks[0].Name).To(Equal("nginx"))
			Expect(b.Cookbooks[0].Constraint.String()).To(Equal("~> 2.7"))
		})
	})

	Context("identifiers", func() {
		It("should handle identifier-named cookbooks via parse", func() {
			// Identifiers like 'nginx', 'nginx2', 'my_cookbook' are handled as cookbook names
			b, err := berksfile.Parse("cookbook 'nginx'\ncookbook 'nginx2'\ncookbook 'my_cookbook'")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Cookbooks).To(HaveLen(3))
			Expect(b.Cookbooks[0].Name).To(Equal("nginx"))
			Expect(b.Cookbooks[1].Name).To(Equal("nginx2"))
			Expect(b.Cookbooks[2].Name).To(Equal("my_cookbook"))
		})
	})

	Context("comments", func() {
		It("should ignore line comments", func() {
			b, err := berksfile.Parse("# This is a comment\ncookbook 'test'")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Cookbooks).To(HaveLen(1))
			Expect(b.Cookbooks[0].Name).To(Equal("test"))
		})

		It("should ignore inline comments", func() {
			b, err := berksfile.Parse("cookbook 'test' # inline comment")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Cookbooks).To(HaveLen(1))
			Expect(b.Cookbooks[0].Name).To(Equal("test"))
		})

		It("should ignore multiple comments", func() {
			b, err := berksfile.Parse("# Comment 1\n# Comment 2\ncookbook 'test'")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Cookbooks).To(HaveLen(1))
			Expect(b.Cookbooks[0].Name).To(Equal("test"))
		})
	})

	Context("complex Berksfile", func() {
		It("should parse complex Berksfile with all features", func() {
			input := `# Berksfile for myapp
source 'https://supermarket.chef.io'

metadata

cookbook 'nginx', '~> 2.7.6'
cookbook 'mysql', '>= 5.0.0'

group :integration do
  cookbook 'test-kitchen'
end

cookbook 'private', git: 'git@github.com:user/repo.git', branch: 'master'
`
			b, err := berksfile.Parse(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(b.Sources).To(HaveLen(1))
			Expect(b.HasMetadata).To(BeTrue())
			Expect(len(b.Cookbooks)).To(BeNumerically(">=", 4))
			Expect(b.Groups).To(HaveKey("integration"))

			private := b.GetCookbook("private")
			Expect(private).NotTo(BeNil())
			Expect(private.Source.Type).To(Equal("git"))
		})
	})

	Context("error handling via Parse", func() {
		DescribeTable("should handle valid and invalid inputs",
			func(input string, shouldError bool) {
				_, err := berksfile.Parse(input)
				if shouldError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			},
			Entry("valid cookbook", `cookbook "test"`, false),
			Entry("valid source", `source "https://supermarket.chef.io"`, false),
			Entry("valid group", "group :test do\nend", false),
			Entry("incomplete group", `group :test`, true),
			Entry("unterminated group", "group :test do\ncookbook 'test'", true),
		)
	})
})

// Merged from utils_constraint_test.go
var _ = Describe("Berksfile.ExtractDirectDependencies with constraints", func() {
	// Validates: Requirements 1.1, 1.2
	It("should include constraint annotations for constrained cookbooks", func() {
		input := `source 'https://supermarket.chef.io'

cookbook 'apt', '~> 7.5'
cookbook 'nginx', '>= 12.0'
`
		b, err := berksfile.Parse(input)
		Expect(err).NotTo(HaveOccurred())

		deps, err := b.ExtractDirectDependencies(nil)
		Expect(err).NotTo(HaveOccurred())

		// The bug condition: constrained cookbooks should include their constraint
		// in the dependency string, e.g. "apt (~> 7.5)" not just "apt"
		Expect(deps).To(ContainElement("apt (~> 7.5)"))
		Expect(deps).To(ContainElement("nginx (>= 12.0.0)"))
	})

	It("should include exact version constraint annotation", func() {
		input := `source 'https://supermarket.chef.io'

cookbook 'mysql', '= 5.0.0'
`
		b, err := berksfile.Parse(input)
		Expect(err).NotTo(HaveOccurred())

		deps, err := b.ExtractDirectDependencies(nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(deps).To(ContainElement("mysql (= 5.0.0)"))
	})
})

// Merged from utils_preservation_test.go
var _ = Describe("Berksfile.ExtractDirectDependencies preservation for unconstrained cookbooks", func() {
	// Validates: Requirements 3.1, 3.2, 3.3
	It("should produce just the cookbook name for unconstrained cookbooks (no version specified)", func() {
		input := `source 'https://supermarket.chef.io'

cookbook 'redis'
cookbook 'apt'
`
		b, err := berksfile.Parse(input)
		Expect(err).NotTo(HaveOccurred())

		deps, err := b.ExtractDirectDependencies(nil)
		Expect(err).NotTo(HaveOccurred())

		// Unconstrained cookbooks should appear as just the name
		Expect(deps).To(ConsistOf("apt", "redis"))
		for _, dep := range deps {
			Expect(dep).NotTo(ContainSubstring("("))
			Expect(dep).NotTo(ContainSubstring(")"))
		}
	})

	It("should produce alphabetically sorted output for multiple unconstrained cookbooks", func() {
		input := `source 'https://supermarket.chef.io'

cookbook 'zookeeper'
cookbook 'mysql'
cookbook 'apache'
cookbook 'nginx'
`
		b, err := berksfile.Parse(input)
		Expect(err).NotTo(HaveOccurred())

		deps, err := b.ExtractDirectDependencies(nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(deps).To(HaveLen(4))
		Expect(sort.StringsAreSorted(deps)).To(BeTrue(), "dependencies should be alphabetically sorted")
		Expect(deps).To(Equal([]string{"apache", "mysql", "nginx", "zookeeper"}))
	})

	It("should preserve unconstrained cookbooks as just the name in a mix of constrained and unconstrained", func() {
		input := `source 'https://supermarket.chef.io'

cookbook 'apt', '~> 7.5'
cookbook 'redis'
cookbook 'nginx', '>= 12.0'
cookbook 'mysql'
`
		b, err := berksfile.Parse(input)
		Expect(err).NotTo(HaveOccurred())

		deps, err := b.ExtractDirectDependencies(nil)
		Expect(err).NotTo(HaveOccurred())

		// On unfixed code, ALL cookbooks appear as just the name (the bug drops constraints).
		// The unconstrained ones (redis, mysql) should be just the name on both unfixed and fixed code.
		Expect(deps).To(ContainElement("redis"))
		Expect(deps).To(ContainElement("mysql"))

		// Verify unconstrained entries have no parentheses
		for _, dep := range deps {
			if dep == "redis" || dep == "mysql" {
				Expect(dep).NotTo(ContainSubstring("("))
			}
		}
	})

	It("should produce just the cookbook name for a path-sourced cookbook with no constraint", func() {
		input := `source 'https://supermarket.chef.io'

cookbook 'test', path: 'test/fixtures/cookbook'
`
		b, err := berksfile.Parse(input)
		Expect(err).NotTo(HaveOccurred())

		deps, err := b.ExtractDirectDependencies(nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(deps).To(ConsistOf("test"))
		Expect(deps[0]).To(Equal("test"))
	})
})
