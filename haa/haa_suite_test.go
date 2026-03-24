package haa_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHybridAssetAllocation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hybrid Asset Allocation Suite")
}
