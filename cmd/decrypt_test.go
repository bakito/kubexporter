package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestDecryptCommand(t *testing.T) {
	testFile := "temp-secret-decrypt.yaml"
	testContent := `apiVersion: v1
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
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	defer os.Remove(testFile)

	t.Run("should decrypt the file successfully", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.AddCommand(decrypt)
		cmd.SetArgs([]string{"decrypt", testFile, "--aes-key", "1234567890123456"})

		err := cmd.Execute()
		if err != nil {
			t.Errorf("unexpected error during decrypt: %v", err)
		}

		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("failed to read test file: %v", err)
		}

		if !strings.Contains(string(content), "dXNlcm5hbWU=") {
			t.Error("expected dXNlcm5hbWU= (username)")
		}
		if !strings.Contains(string(content), "cGFzc3dvcmQ=") {
			t.Error("expected cGFzc3dvcmQ= (password)")
		}
		if !strings.Contains(string(content), "secret-api-key-123") {
			t.Error("expected secret-api-key-123")
		}
		if !strings.Contains(string(content), "my-secret-token") {
			t.Error("expected my-secret-token")
		}
		if strings.Contains(string(content), "KUBEXPORTER_AES@") {
			t.Error("expected no KUBEXPORTER_AES@ prefix")
		}
	})
}
