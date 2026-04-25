package berksfile_test

import (
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bdwyertech/go-berkshelf/pkg/berksfile"
)

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
