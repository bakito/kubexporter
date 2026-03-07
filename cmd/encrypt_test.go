package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestEncryptCommand(t *testing.T) {
	testFile := "temp-secret-encrypt.yaml"
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
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	defer os.Remove(testFile)

	t.Run("should encrypt and decrypt a file", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.AddCommand(encrypt)
		cmd.AddCommand(decrypt)
		cmd.SetArgs([]string{"encrypt", testFile, "--aes-key", "1234567890123456"})

		err := cmd.Execute()
		if err != nil {
			t.Errorf("unexpected error during encrypt: %v", err)
		}

		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("failed to read test file: %v", err)
		}
		if !strings.Contains(string(content), "KUBEXPORTER_AES@") {
			t.Error("expected content to be encrypted")
		}

		cmd.SetArgs([]string{"decrypt", testFile, "--aes-key", "1234567890123456"})
		err = cmd.Execute()
		if err != nil {
			t.Errorf("unexpected error during decrypt: %v", err)
		}

		content, err = os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("failed to read test file: %v", err)
		}
		if !strings.Contains(string(content), "apiVersion: v1") {
			t.Error("expected apiVersion: v1")
		}
		if !strings.Contains(string(content), "kind: Secret") {
			t.Error("expected kind: Secret")
		}
		if !strings.Contains(string(content), "secret-api-key-123") {
			t.Error("expected secret-api-key-123")
		}
	})
}
