package lockfile_test

import (
	"os"
	"path/filepath"
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bdwyertech/go-berkshelf/pkg/lockfile"
)

var _ = Describe("ExtractDirectDependencies preservation for unconstrained cookbooks", func() {
	// Validates: Requirements 3.1, 3.2, 3.3
	var (
		tmpDir        string
		berksfilePath string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "lockfile-preservation-test")
		Expect(err).NotTo(HaveOccurred())
		berksfilePath = filepath.Join(tmpDir, "Berksfile")
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	It("should produce just the cookbook name for unconstrained cookbooks (no version specified)", func() {
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'redis'
cookbook 'apt'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
		Expect(err).NotTo(HaveOccurred())

		// Unconstrained cookbooks should appear as just the name
		Expect(deps).To(ConsistOf("apt", "redis"))
		// Each entry should be exactly the cookbook name with no parentheses
		for _, dep := range deps {
			Expect(dep).NotTo(ContainSubstring("("))
			Expect(dep).NotTo(ContainSubstring(")"))
		}
	})

	It("should produce alphabetically sorted output for multiple unconstrained cookbooks", func() {
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'zookeeper'
cookbook 'mysql'
cookbook 'apache'
cookbook 'nginx'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(deps).To(HaveLen(4))
		Expect(sort.StringsAreSorted(deps)).To(BeTrue(), "dependencies should be alphabetically sorted")
		Expect(deps).To(Equal([]string{"apache", "mysql", "nginx", "zookeeper"}))
	})

	It("should preserve unconstrained cookbooks as just the name in a mix of constrained and unconstrained", func() {
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'apt', '~> 7.5'
cookbook 'redis'
cookbook 'nginx', '>= 12.0'
cookbook 'mysql'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
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
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'test', path: 'test/fixtures/cookbook'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(deps).To(ConsistOf("test"))
		Expect(deps[0]).To(Equal("test"))
	})
})
