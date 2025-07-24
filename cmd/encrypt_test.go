package cmd

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Encrypt Command", func() {
	var testFile string

	BeforeEach(func() {
		testFile = "temp-secret.yaml"
		testContent := `apiVersion: v1
kind: Secret
metadata:
  name: test-secret 
  namespace: default
type: Opaque
data:
  username: dXNlcm5hbWU=
  password: cGFzc3dvcmQ=
stringData:
  api-key: "secret-api-key-123"
  token: "my-secret-token"`

		err := os.WriteFile(testFile, []byte(testContent), 0o644)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.Remove(testFile)
	})

	It("should encrypt and decrypt a file", func() {
		cmd := &cobra.Command{}
		cmd.AddCommand(encrypt)
		cmd.AddCommand(decrypt)
		cmd.SetArgs([]string{"encrypt", testFile, "--aes-key", "1234567890123456"})

		err := cmd.Execute()
		Expect(err).NotTo(HaveOccurred())

		content, err := os.ReadFile(testFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("KUBEXPORTER_AES@"))

		cmd.SetArgs([]string{"decrypt", testFile, "--aes-key", "1234567890123456"})
		err = cmd.Execute()
		Expect(err).NotTo(HaveOccurred())

		content, err = os.ReadFile(testFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("apiVersion: v1"))
		Expect(string(content)).To(ContainSubstring("kind: Secret"))
		Expect(string(content)).To(ContainSubstring("secret-api-key-123"))
	})
})
