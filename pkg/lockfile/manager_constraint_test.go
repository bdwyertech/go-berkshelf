package lockfile_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bdwyertech/go-berkshelf/pkg/lockfile"
)

var _ = Describe("ExtractDirectDependencies with constraints", func() {
	var (
		tmpDir        string
		berksfilePath string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "lockfile-constraint-test")
		Expect(err).NotTo(HaveOccurred())
		berksfilePath = filepath.Join(tmpDir, "Berksfile")
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	// Validates: Requirements 1.1, 1.2
	It("should include constraint annotations for constrained cookbooks", func() {
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'apt', '~> 7.5'
cookbook 'nginx', '>= 12.0'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
		Expect(err).NotTo(HaveOccurred())

		// The bug condition: constrained cookbooks should include their constraint
		// in the dependency string, e.g. "apt (~> 7.5)" not just "apt"
		Expect(deps).To(ContainElement("apt (~> 7.5)"))
		Expect(deps).To(ContainElement("nginx (>= 12.0.0)"))
	})

	It("should include exact version constraint annotation", func() {
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'mysql', '= 5.0.0'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(deps).To(ContainElement("mysql (= 5.0.0)"))
	})
})
