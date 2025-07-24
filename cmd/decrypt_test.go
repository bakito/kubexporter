package cmd

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Decrypt Command", func() {
	var testFile string
	var testContent string

	BeforeEach(func() {
		// Create a temporary test file
		testFile = "temp-secret.yaml"

		// Create test content with encrypted data
		testContent = `apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
type: Opaque
data:
  username: KUBEXPORTER_AES@f6MMcnyTt4Tm4zGotyhAzPLjWSeV42ke8hu93AJG251W4Ew17RI6pA==
  password: KUBEXPORTER_AES@f6MMcnyTt4Tm4zGosDdI2vK9CDmU2W0eNvcVMbvee0S/vlFO6+Vf+w==
stringData:
  api-key: KUBEXPORTER_AES@f6MMcnyTt4Tm4zGooBVt0vT6QS6H3RFI6AR8sM7D8/ElvOzygKsuM3ZT5nCfmg==
  token: KUBEXPORTER_AES@f6MMcnyTt4Tm4zGovgkj0/TtHiqDmUhM5hg/X5Nsgw07ZtUsFnYpG4OnGg==`

		err := os.WriteFile(testFile, []byte(testContent), 0o644)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.Remove(testFile)
	})

	It("should decrypt the file successfully", func() {
		cmd := &cobra.Command{}
		cmd.AddCommand(decrypt)
		cmd.SetArgs([]string{"decrypt", testFile, "--aes-key", "1234567890123456"})

		err := cmd.Execute()
		Expect(err).NotTo(HaveOccurred())

		// Verify the file was decrypted
		content, err := os.ReadFile(testFile)
		Expect(err).NotTo(HaveOccurred())

		// Check that encrypted fields were decrypted
		Expect(string(content)).To(ContainSubstring("dXNlcm5hbWU=")) // base64 encoded "username"
		Expect(string(content)).To(ContainSubstring("cGFzc3dvcmQ=")) // base64 encoded "password"
		Expect(string(content)).To(ContainSubstring("secret-api-key-123"))
		Expect(string(content)).To(ContainSubstring("my-secret-token"))
		Expect(string(content)).NotTo(ContainSubstring("KUBEXPORTER_AES@"))
	})
})
