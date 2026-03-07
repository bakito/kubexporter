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

	tests := []struct {
		name     string
		aesKey   string
		validate func(t *testing.T, testFile string)
	}{
		{
			name:   "should encrypt and decrypt a file",
			aesKey: "1234567890123456",
			validate: func(t *testing.T, testFile string) {
				t.Helper()
				content, err := os.ReadFile(testFile)
				if err != nil {
					t.Fatalf("failed to read test file: %v", err)
				}
				if !strings.Contains(string(content), "KUBEXPORTER_AES@") {
					t.Error("expected content to be encrypted")
				}

				cmd := &cobra.Command{}
				cmd.AddCommand(decrypt)
				cmd.SetArgs([]string{"decrypt", testFile, "--aes-key", "1234567890123456"})
				if err := cmd.Execute(); err != nil {
					t.Errorf("unexpected error during decrypt: %v", err)
				}

				content, err = os.ReadFile(testFile)
				if err != nil {
					t.Fatalf("failed to read test file: %v", err)
				}
				for _, s := range []string{"apiVersion: v1", "kind: Secret", "secret-api-key-123"} {
					if !strings.Contains(string(content), s) {
						t.Errorf("expected content to contain %q", s)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.AddCommand(encrypt)
			cmd.SetArgs([]string{"encrypt", testFile, "--aes-key", tt.aesKey})

			if err := cmd.Execute(); err != nil {
				t.Errorf("unexpected error during encrypt: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, testFile)
			}
		})
	}
}
