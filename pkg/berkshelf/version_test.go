package berkshelf_test

import (
	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Version", func() {
	DescribeTable("NewVersion",
		func(version string, want string, wantErr bool) {
			got, err := berkshelf.NewVersion(version)
			if wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(got.String()).To(Equal(want))
			}
		},
		Entry("simple version", "1.0.0", "1.0.0", false),
		Entry("version with v prefix", "v2.1.3", "2.1.3", false),
		Entry("version with prerelease", "1.0.0-alpha.1", "1.0.0-alpha.1", false),
		Entry("version with build metadata", "1.0.0+20130313144700", "1.0.0+20130313144700", false),
		Entry("invalid version", "not.a.version", "", true),
		Entry("empty version", "", "", true),
	)

	DescribeTable("Version.Compare",
		func(v1Str string, v2Str string, want int) {
			v1, err := berkshelf.NewVersion(v1Str)
			Expect(err).NotTo(HaveOccurred())
			v2, err := berkshelf.NewVersion(v2Str)
			Expect(err).NotTo(HaveOccurred())

			Expect(v1.Compare(v2)).To(Equal(want))
		},
		Entry("equal versions", "1.0.0", "1.0.0", 0),
		Entry("v1 less than v2", "1.0.0", "2.0.0", -1),
		Entry("v1 greater than v2", "2.0.0", "1.0.0", 1),
		Entry("patch version difference", "1.0.1", "1.0.2", -1),
		Entry("prerelease vs release", "1.0.0-alpha", "1.0.0", -1),
	)

	Describe("Version helpers", func() {
		var v1, v2 *berkshelf.Version

		BeforeEach(func() {
			v1 = berkshelf.MustVersion("1.0.0")
			v2 = berkshelf.MustVersion("2.0.0")
		})

		It("should report LessThan correctly", func() {
			Expect(v1.LessThan(v2)).To(BeTrue())
		})

		It("should report GreaterThan correctly", func() {
			Expect(v2.GreaterThan(v1)).To(BeTrue())
		})

		It("should report Equal correctly", func() {
			Expect(v1.Equal(berkshelf.MustVersion("1.0.0"))).To(BeTrue())
		})
	})

	It("should panic on invalid version", func() {
		Expect(func() {
			berkshelf.MustVersion("invalid.version")
		}).To(Panic())
	})
})
