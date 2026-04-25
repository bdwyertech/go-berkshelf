package berkshelf_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBerkshelfSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Berkshelf Suite")
}
