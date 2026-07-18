package types

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestEncrypted_Setup(t *testing.T) {
	tests := []struct {
		name       string
		aesKey     string
		envKey     string
		kindFields KindFields
		wantErr    bool
	}{
		{
			name:   "16 should be ok",
			aesKey: "1234567890123456",
		},
		{
			name:   "24 should be ok",
			aesKey: "123456789012345678901234",
		},
		{
			name:   "32 should be ok",
			aesKey: "12345678901234567890123456789012",
		},
		{
			name:       "fail if no key is set and kind fields are set",
			aesKey:     "",
			kindFields: KindFields{"Secret": {{"data"}}},
			wantErr:    true,
		},
		{
			name:   "use key from env",
			aesKey: "",
			envKey: "1234567890123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := &Encrypted{
				AesKey:     tt.aesKey,
				KindFields: tt.kindFields,
			}
			if tt.envKey != "" {
				t.Setenv(EnvAesKey, tt.envKey)
			}

			err := enc.Setup()
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypted.Setup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if enc.gcm == nil {
					t.Error("Encrypted.gcm is nil")
				}
				if enc.nonce == nil {
					t.Error("Encrypted.nonce is nil")
				}
			} else {
				if enc.gcm != nil {
					t.Error("Encrypted.gcm is not nil")
				}
				if enc.nonce != nil {
					t.Error("Encrypted.nonce is not nil")
				}
			}
		})
	}
}

func TestConfig_EncryptFields(t *testing.T) {
	tests := []struct {
		name     string
		aesKey   string
		input    string
		validate func(t *testing.T, got string)
	}{
		{
			name:   "should encrypt the Secret data",
			aesKey: "1234567890123456",
			input:  "don't tell anyone!",
			validate: func(t *testing.T, got string) {
				t.Helper()
				if !strings.HasPrefix(got, prefix) {
					t.Errorf("expected secret to have prefix %q, but got %q", prefix, got)
				}
			},
		},
		{
			name:   "should return an empty string if no key is set",
			aesKey: "",
			input:  "don't tell anyone!",
			validate: func(t *testing.T, got string) {
				t.Helper()
				if got != "" {
					t.Errorf("expected empty secret, but got %q", got)
				}
			},
		},
		{
			name:   "should not encrypt if already encrypted",
			aesKey: "1234567890123456",
			input:  "KUBEXPORTER_AES@wKCCGma3NhnvzLMbMCrPK7nq7cQV6hF385YuqLjSk+UXCRgaQATO3PPUsfoheg==",
			validate: func(t *testing.T, got string) {
				t.Helper()
				expected := "KUBEXPORTER_AES@wKCCGma3NhnvzLMbMCrPK7nq7cQV6hF385YuqLjSk+UXCRgaQATO3PPUsfoheg=="
				if got != expected {
					t.Errorf("expected %q, but got %q", expected, got)
				}
			},
		},
		{
			name:   "should not encrypt empty strings",
			aesKey: "1234567890123456",
			input:  "",
			validate: func(t *testing.T, got string) {
				t.Helper()
				if got != "" {
					t.Errorf("expected empty secret, but got %q", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := &Encrypted{
				AesKey: tt.aesKey,
				KindFields: map[string][][]string{
					"Secret": {{"data"}},
				},
			}
			_ = enc.Setup()
			config := &Config{Encrypted: enc}

			us := unstructured.Unstructured{Object: map[string]any{
				"data": map[string]any{
					"secret": tt.input,
				},
			}}
			config.EncryptFields(&GroupResource{APIResource: metav1.APIResource{Kind: "Secret"}}, us)
			secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
			tt.validate(t, secret)
		})
	}
}

func TestDecryptFields(t *testing.T) {
	enc := &Encrypted{
		AesKey: "1234567890123456",
	}
	_ = enc.Setup()

	tests := []struct {
		name          string
		input         string
		expected      string
		expectedCount int
	}{
		{
			name:          "should decrypt the value correctly",
			input:         "KUBEXPORTER_AES@wKCCGma3NhnvzLMbMCrPK7nq7cQV6hF385YuqLjSk+UXCRgaQATO3PPUsfoheg==",
			expected:      "don't tell anyone!",
			expectedCount: 1,
		},
		{
			name:          "should not decrypt if not encrypted",
			input:         "don't tell anyone!",
			expected:      "don't tell anyone!",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			us := unstructured.Unstructured{Object: map[string]any{
				"data": map[string]any{
					"secret": tt.input,
				},
			}}
			cnt, err := decryptFields(us.Object, enc.gcm, len(enc.nonce))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cnt != tt.expectedCount {
				t.Errorf("expected %d decrypted field, but got %d", tt.expectedCount, cnt)
			}
			secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
			if secret != tt.expected {
				t.Errorf("expected %q, but got %q", tt.expected, secret)
			}
		})
	}
}
