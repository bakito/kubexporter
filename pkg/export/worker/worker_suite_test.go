package worker_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWorker(t *testing.T) {
	RegisterFailHandler(Fail)
	t.TempDir()
	RunSpecs(t, "Worker Suite")
}
