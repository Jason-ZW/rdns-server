package coredns_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCoredns(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "coredns suite")
}
