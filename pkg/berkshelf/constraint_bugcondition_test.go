package berkshelf_test

import (
	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7**
var _ = Describe("BugCondition: Two-Part Constraint Permissive Evaluation", func() {

	// Helper to check a constraint against a version
	checkConstraint := func(constraintStr, versionStr string) bool {
		c, err := berkshelf.NewConstraint(constraintStr)
		Expect(err).NotTo(HaveOccurred())
		v, err := berkshelf.NewVersion(versionStr)
		Expect(err).NotTo(HaveOccurred())
		return c.Check(v)
	}

	Context("two-part version constraints that should reject versions", func() {

		It("<= 12.0 should reject 12.0.12 (12.0.12 > 12.0.0)", func() {
			// Bug: unfixed code returns true because semver treats <= 12.0 permissively
			Expect(checkConstraint("<= 12.0", "12.0.12")).To(BeFalse())
		})

		It(">= 7.0 should reject 6.9.9 (6.9.9 < 7.0.0)", func() {
			// Bug: unfixed code may return true due to permissive two-part parsing
			Expect(checkConstraint(">= 7.0", "6.9.9")).To(BeFalse())
		})

		It("= 5.0 should reject 5.0.1 (5.0.1 != 5.0.0)", func() {
			// Bug: unfixed code may return true because = 5.0 matches 5.0.x
			Expect(checkConstraint("= 5.0", "5.0.1")).To(BeFalse())
		})

		It("< 2.0 should reject 2.0.0 (2.0.0 is not < 2.0.0)", func() {
			// Bug: unfixed code may return true due to permissive two-part parsing
			Expect(checkConstraint("< 2.0", "2.0.0")).To(BeFalse())
		})

		It("> 12.0 should reject 12.0.0 (12.0.0 is not > 12.0.0)", func() {
			// Bug: unfixed code may return true due to permissive two-part parsing
			Expect(checkConstraint("> 12.0", "12.0.0")).To(BeFalse())
		})
	})

	Context("single-segment version constraints that should reject versions", func() {

		It("<= 12 should reject 12.0.1 (12.0.1 > 12.0.0)", func() {
			// Bug: unfixed code returns true because semver treats <= 12 permissively
			Expect(checkConstraint("<= 12", "12.0.1")).To(BeFalse())
		})
	})
})
