package berkshelf_test

import (
	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Constraint", func() {

	// ── From constraint_test.go ──

	DescribeTable("NewConstraint",
		func(constraint string, wantErr bool) {
			_, err := berkshelf.NewConstraint(constraint)
			if wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("exact version", "= 1.0.0", false),
		Entry("greater than", "> 1.0.0", false),
		Entry("greater than or equal", ">= 1.0.0", false),
		Entry("less than", "< 2.0.0", false),
		Entry("less than or equal", "<= 2.0.0", false),
		Entry("pessimistic operator - patch", "~> 1.2.3", false),
		Entry("pessimistic operator - minor", "~> 1.2", false),
		Entry("pessimistic operator - major", "~> 1", false),
		Entry("Ruby-style equality", "== 1.0.0", false),
		Entry("empty constraint", "", false),
	)

	DescribeTable("Constraint.Check",
		func(constraintStr, versionStr string, want bool) {
			c, err := berkshelf.NewConstraint(constraintStr)
			Expect(err).NotTo(HaveOccurred())
			v, err := berkshelf.NewVersion(versionStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(v)).To(Equal(want))
		},
		Entry("exact match", "= 1.0.0", "1.0.0", true),
		Entry("exact mismatch", "= 1.0.0", "1.0.1", false),
		Entry("greater than - satisfied", "> 1.0.0", "1.0.1", true),
		Entry("greater than - not satisfied", "> 1.0.0", "1.0.0", false),
		Entry("greater than or equal - equal", ">= 1.0.0", "1.0.0", true),
		Entry("greater than or equal - greater", ">= 1.0.0", "1.0.1", true),
		Entry("less than - satisfied", "< 2.0.0", "1.9.9", true),
		Entry("less than - not satisfied", "< 2.0.0", "2.0.0", false),
	)

	DescribeTable("Pessimistic Constraint via Check",
		func(constraintStr, versionStr string, want bool) {
			c, err := berkshelf.NewConstraint(constraintStr)
			Expect(err).NotTo(HaveOccurred())
			v, err := berkshelf.NewVersion(versionStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(v)).To(Equal(want))
		},
		// ~> 1.2.3 should allow >= 1.2.3, < 1.3.0
		Entry("pessimistic patch - exact match", "~> 1.2.3", "1.2.3", true),
		Entry("pessimistic patch - patch increment", "~> 1.2.3", "1.2.4", true),
		Entry("pessimistic patch - minor increment", "~> 1.2.3", "1.3.0", false),
		Entry("pessimistic patch - major increment", "~> 1.2.3", "2.0.0", false),
		Entry("pessimistic patch - below minimum", "~> 1.2.3", "1.2.2", false),
		// ~> 1.2 should allow >= 1.2.0, < 2.0.0
		Entry("pessimistic minor - exact match", "~> 1.2", "1.2.0", true),
		Entry("pessimistic minor - patch increment", "~> 1.2", "1.2.5", true),
		Entry("pessimistic minor - minor increment", "~> 1.2", "1.3.0", true),
		Entry("pessimistic minor - major increment", "~> 1.2", "2.0.0", false),
		// ~> 1 should allow >= 1.0.0, < 2.0.0
		Entry("pessimistic major - exact match", "~> 1", "1.0.0", true),
		Entry("pessimistic major - minor increment", "~> 1", "1.5.0", true),
		Entry("pessimistic major - major increment", "~> 1", "2.0.0", false),
	)

	Describe("Constraint Conversion via String()", func() {
		DescribeTable("verifies conversion output",
			func(input, expected string) {
				c, err := berkshelf.NewConstraint(input)
				Expect(err).NotTo(HaveOccurred())
				Expect(c.String()).To(Equal(expected))
			},
			// Pessimistic constraints: String() returns the raw input
			Entry("pessimistic patch level", "~> 1.2.3", "~> 1.2.3"),
			Entry("pessimistic minor level", "~> 1.2", "~> 1.2"),
			Entry("pessimistic major level", "~> 1", "~> 1"),
			// Ruby equality: == becomes = and version is normalized
			Entry("Ruby equality operator", "== 1.0.0", "== 1.0.0"),
			// Standard constraint: already normalized
			Entry("standard constraint unchanged", ">= 1.0.0", ">= 1.0.0"),
		)
	})

	It("should panic on invalid constraint", func() {
		Expect(func() {
			berkshelf.MustConstraint("invalid constraint syntax !!!")
		}).To(Panic())
	})

	It("should return original constraint string", func() {
		c := berkshelf.MustConstraint("~> 1.2.3")
		Expect(c.String()).To(Equal("~> 1.2.3"))
	})

	// ── From constraint_pessimistic_test.go ──

	DescribeTable("Pessimistic Constraint Operator",
		func(constraintStr, versionStr string, shouldPass bool) {
			c, err := berkshelf.NewConstraint(constraintStr)
			Expect(err).NotTo(HaveOccurred())
			v, err := berkshelf.NewVersion(versionStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(v)).To(Equal(shouldPass))
		},
		// ~> 2.0 should match >= 2.0, < 3.0
		Entry("~> 2.0 matches 2.0.0", "~> 2.0", "2.0.0", true),
		Entry("~> 2.0 matches 2.1.0", "~> 2.0", "2.1.0", true),
		Entry("~> 2.0 matches 2.9.9", "~> 2.0", "2.9.9", true),
		Entry("~> 2.0 matches 2.10.0", "~> 2.0", "2.10.0", true),
		Entry("~> 2.0 does not match 3.0.0", "~> 2.0", "3.0.0", false),
		Entry("~> 2.0 does not match 1.9.9", "~> 2.0", "1.9.9", false),
		// ~> 2.0.0 should match >= 2.0.0, < 2.1.0
		Entry("~> 2.0.0 matches 2.0.0", "~> 2.0.0", "2.0.0", true),
		Entry("~> 2.0.0 matches 2.0.1", "~> 2.0.0", "2.0.1", true),
		Entry("~> 2.0.0 matches 2.0.99", "~> 2.0.0", "2.0.99", true),
		Entry("~> 2.0.0 does not match 2.1.0", "~> 2.0.0", "2.1.0", false),
		Entry("~> 2.0.0 does not match 1.9.9", "~> 2.0.0", "1.9.9", false),
		// ~> 1.2.3 should match >= 1.2.3, < 1.3.0
		Entry("~> 1.2.3 matches 1.2.3", "~> 1.2.3", "1.2.3", true),
		Entry("~> 1.2.3 matches 1.2.4", "~> 1.2.3", "1.2.4", true),
		Entry("~> 1.2.3 matches 1.2.99", "~> 1.2.3", "1.2.99", true),
		Entry("~> 1.2.3 does not match 1.3.0", "~> 1.2.3", "1.3.0", false),
		Entry("~> 1.2.3 does not match 1.2.2", "~> 1.2.3", "1.2.2", false),
		// ~> 1 should match >= 1.0.0, < 2.0.0
		Entry("~> 1 matches 1.0.0", "~> 1", "1.0.0", true),
		Entry("~> 1 matches 1.9.9", "~> 1", "1.9.9", true),
		Entry("~> 1 matches 1.99.99", "~> 1", "1.99.99", true),
		Entry("~> 1 does not match 2.0.0", "~> 1", "2.0.0", false),
		Entry("~> 1 does not match 0.9.9", "~> 1", "0.9.9", false),
		// Edge cases
		Entry("~> 0 matches 0.0.0", "~> 0", "0.0.0", true),
		Entry("~> 0 matches 0.9.9", "~> 0", "0.9.9", true),
		Entry("~> 0 does not match 1.0.0", "~> 0", "1.0.0", false),
		Entry("~> 0.0 matches 0.0.0", "~> 0.0", "0.0.0", true),
		Entry("~> 0.0 matches 0.99.99", "~> 0.0", "0.99.99", true),
		Entry("~> 0.0 does not match 1.0.0", "~> 0.0", "1.0.0", false),
		Entry("~> 0.0.0 matches 0.0.0", "~> 0.0.0", "0.0.0", true),
		Entry("~> 0.0.0 matches 0.0.99", "~> 0.0.0", "0.0.99", true),
		Entry("~> 0.0.0 does not match 0.1.0", "~> 0.0.0", "0.1.0", false),
	)

	Describe("Pessimistic Constraint Conversion via Check", func() {
		// Rewritten from TestPessimisticConstraintConversion to test through public API
		// Instead of calling convertPessimisticConstraint directly, we verify behavior
		// by checking boundary versions.

		It("~> 1 behaves as >= 1.0.0, < 2.0.0", func() {
			c, err := berkshelf.NewConstraint("~> 1")
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(berkshelf.MustVersion("1.0.0"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("1.99.99"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("2.0.0"))).To(BeFalse())
			Expect(c.Check(berkshelf.MustVersion("0.9.9"))).To(BeFalse())
		})

		It("~> 2.0 behaves as >= 2.0.0, < 3.0.0", func() {
			c, err := berkshelf.NewConstraint("~> 2.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(berkshelf.MustVersion("2.0.0"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("2.9.9"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("3.0.0"))).To(BeFalse())
			Expect(c.Check(berkshelf.MustVersion("1.9.9"))).To(BeFalse())
		})

		It("~> 2.0.0 behaves as >= 2.0.0, < 2.1.0", func() {
			c, err := berkshelf.NewConstraint("~> 2.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(berkshelf.MustVersion("2.0.0"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("2.0.99"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("2.1.0"))).To(BeFalse())
			Expect(c.Check(berkshelf.MustVersion("1.9.9"))).To(BeFalse())
		})

		It("~> 1.2.3 behaves as >= 1.2.3, < 1.3.0", func() {
			c, err := berkshelf.NewConstraint("~> 1.2.3")
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(berkshelf.MustVersion("1.2.3"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("1.2.99"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("1.3.0"))).To(BeFalse())
			Expect(c.Check(berkshelf.MustVersion("1.2.2"))).To(BeFalse())
		})

		It("~> 0 behaves as >= 0.0.0, < 1.0.0", func() {
			c, err := berkshelf.NewConstraint("~> 0")
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(berkshelf.MustVersion("0.0.0"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("0.99.99"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("1.0.0"))).To(BeFalse())
		})

		It("~> 0.0 behaves as >= 0.0.0, < 1.0.0", func() {
			c, err := berkshelf.NewConstraint("~> 0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(berkshelf.MustVersion("0.0.0"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("0.99.99"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("1.0.0"))).To(BeFalse())
		})

		It("~> 0.0.0 behaves as >= 0.0.0, < 0.1.0", func() {
			c, err := berkshelf.NewConstraint("~> 0.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Check(berkshelf.MustVersion("0.0.0"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("0.0.99"))).To(BeTrue())
			Expect(c.Check(berkshelf.MustVersion("0.1.0"))).To(BeFalse())
		})
	})
})

// ── From constraint_bugcondition_test.go ──

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

// ── From constraint_preservation_test.go ──

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
