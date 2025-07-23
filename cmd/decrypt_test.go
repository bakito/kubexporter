package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptCommand(t *testing.T) {
	// Create a temporary test file
	testFile := "temp-secret.yaml"
	defer os.Remove(testFile)

	// Create test content with encrypted data
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
	require.NoError(t, err)

	// Test decrypt command
	cmd := &cobra.Command{}
	cmd.AddCommand(decrypt)
	cmd.SetArgs([]string{"decrypt", testFile, "--aes-key", "1234567890123456"})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify file was decrypted
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)

	// Check that encrypted fields were decrypted
	assert.Contains(t, string(content), "dXNlcm5hbWU=") // base64 encoded "username"
	assert.Contains(t, string(content), "cGFzc3dvcmQ=") // base64 encoded "password"
	assert.Contains(t, string(content), "secret-api-key-123")
	assert.Contains(t, string(content), "my-secret-token")
	assert.NotContains(t, string(content), "KUBEXPORTER_AES@")
}
