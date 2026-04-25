package berksfile_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bdwyertech/go-berkshelf/pkg/berksfile"
)

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
