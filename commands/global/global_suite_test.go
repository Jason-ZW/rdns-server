package global_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGlobal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "global suite")
}
