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

	tests := []struct {
		name     string
		aesKey   string
		validate func(t *testing.T, testFile string)
	}{
		{
			name:   "should decrypt the file successfully",
			aesKey: "1234567890123456",
			validate: func(t *testing.T, testFile string) {
				t.Helper()
				content, err := os.ReadFile(testFile)
				if err != nil {
					t.Fatalf("failed to read test file: %v", err)
				}

				for _, s := range []string{"dXNlcm5hbWU=", "cGFzc3dvcmQ=", "secret-api-key-123", "my-secret-token"} {
					if !strings.Contains(string(content), s) {
						t.Errorf("expected content to contain %q", s)
					}
				}
				if strings.Contains(string(content), "KUBEXPORTER_AES@") {
					t.Error("expected no KUBEXPORTER_AES@ prefix")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.AddCommand(decrypt)
			cmd.SetArgs([]string{"decrypt", testFile, "--aes-key", tt.aesKey})

			if err := cmd.Execute(); err != nil {
				t.Errorf("unexpected error during decrypt: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, testFile)
			}
		})
	}
}
