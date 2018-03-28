package agent_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("agent_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Driver-Agent communication test suite",
		[]Reporter{junitReporter})
}
