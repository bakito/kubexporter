package worker_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestWorker(t *testing.T) {
	RegisterFailHandler(Fail)
	t.TempDir()
	RunSpecs(t, "Worker Suite")
}
