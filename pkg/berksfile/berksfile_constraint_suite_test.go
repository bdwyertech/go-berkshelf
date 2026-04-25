package berksfile_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConstraintSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Berksfile Constraint Suite")
}
