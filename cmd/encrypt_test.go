package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptCommand(t *testing.T) {
	// Create a temporary test file
	testFile := "temp-secret.yaml"
	defer os.Remove(testFile)

	// Create test content
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
	require.NoError(t, err)

	// Test encrypt command
	cmd := &cobra.Command{}
	cmd.AddCommand(encrypt)
	cmd.AddCommand(decrypt)
	cmd.SetArgs([]string{"encrypt", testFile, "--aes-key", "1234567890123456"})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify file was encrypted
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)

	// Check that some fields are encrypted (have the prefix)
	assert.Contains(t, string(content), "KUBEXPORTER_AES@")

	// Test decrypt to verify it works
	cmd.SetArgs([]string{"decrypt", testFile, "--aes-key", "1234567890123456"})
	err = cmd.Execute()
	require.NoError(t, err)

	// Verify file was decrypted back to original
	content, err = os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "apiVersion: v1")
	assert.Contains(t, string(content), "kind: Secret")
	assert.Contains(t, string(content), "secret-api-key-123")
}
