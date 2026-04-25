package lockfile_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConstraintSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lockfile Constraint Suite")
}
