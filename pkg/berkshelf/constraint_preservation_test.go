package berkshelf_test

import (
	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7**
var _ = Describe("Preservation: Non-Buggy Constraint Behavior", func() {

	// Helper to check a constraint against a version
	checkConstraint := func(constraintStr, versionStr string) bool {
		c, err := berkshelf.NewConstraint(constraintStr)
		Expect(err).NotTo(HaveOccurred())
		v, err := berkshelf.NewVersion(versionStr)
		Expect(err).NotTo(HaveOccurred())
		return c.Check(v)
	}

	Context("three-part non-pessimistic constraints evaluate correctly", func() {

		It("<= 12.0.0 against 12.0.0 returns true", func() {
			Expect(checkConstraint("<= 12.0.0", "12.0.0")).To(BeTrue())
		})

		It(">= 1.0.0 against 1.0.1 returns true", func() {
			Expect(checkConstraint(">= 1.0.0", "1.0.1")).To(BeTrue())
		})

		It("= 5.0.0 against 5.0.1 returns false", func() {
			Expect(checkConstraint("= 5.0.0", "5.0.1")).To(BeFalse())
		})

		It("< 2.0.0 against 1.9.9 returns true", func() {
			Expect(checkConstraint("< 2.0.0", "1.9.9")).To(BeTrue())
		})

		It("> 1.0.0 against 1.0.1 returns true", func() {
			Expect(checkConstraint("> 1.0.0", "1.0.1")).To(BeTrue())
		})
	})

	Context("pessimistic constraints work correctly", func() {

		It("~> 1.2 allows 1.5.0", func() {
			Expect(checkConstraint("~> 1.2", "1.5.0")).To(BeTrue())
		})

		It("~> 1.2 rejects 2.0.0", func() {
			Expect(checkConstraint("~> 1.2", "2.0.0")).To(BeFalse())
		})

		It("~> 1.2.3 allows 1.2.5", func() {
			Expect(checkConstraint("~> 1.2.3", "1.2.5")).To(BeTrue())
		})

		It("~> 1.2.3 rejects 1.3.0", func() {
			Expect(checkConstraint("~> 1.2.3", "1.3.0")).To(BeFalse())
		})
	})

	Context("empty constraint defaults to >= 0.0.0", func() {

		It("empty constraint String() returns >= 0.0.0", func() {
			c, err := berkshelf.NewConstraint("")
			Expect(err).NotTo(HaveOccurred())
			Expect(c.String()).To(Equal(">= 0.0.0"))
		})
	})

	Context("== operator converts to =", func() {

		It("NewConstraint with == 1.0.0 works without error", func() {
			c, err := berkshelf.NewConstraint("== 1.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
		})

		It("== 1.0.0 constraint evaluates correctly (matches 1.0.0)", func() {
			c, err := berkshelf.NewConstraint("== 1.0.0")
			Expect(err).NotTo(HaveOccurred())
			v, err := berkshelf.NewVersion("1.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(v)).To(BeTrue())
		})

		It("== 1.0.0 constraint rejects non-matching version", func() {
			c, err := berkshelf.NewConstraint("== 1.0.0")
			Expect(err).NotTo(HaveOccurred())
			v, err := berkshelf.NewVersion("1.0.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(v)).To(BeFalse())
		})
	})
})
