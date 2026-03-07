package types

import (
	"os"
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
			_ = os.Unsetenv(EnvAesKey)
			defer func() { _ = os.Unsetenv(EnvAesKey) }()

			enc := &Encrypted{
				AesKey:     tt.aesKey,
				KindFields: tt.kindFields,
			}
			if tt.envKey != "" {
				_ = os.Setenv(EnvAesKey, tt.envKey)
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
	setup := func() (*Encrypted, *Config) {
		enc := &Encrypted{
			AesKey: "1234567890123456",
			KindFields: map[string][][]string{
				"Secret": {{"data"}},
			},
		}
		_ = enc.Setup()
		config := &Config{
			Encrypted: enc,
		}
		return enc, config
	}

	t.Run("should encrypt the Secret data", func(t *testing.T) {
		_, config := setup()
		us := unstructured.Unstructured{Object: map[string]any{
			"data": map[string]any{
				"secret": "don't tell anyone!",
			},
		}}
		config.EncryptFields(&GroupResource{APIResource: metav1.APIResource{Kind: "Secret"}}, us)
		secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
		if !strings.HasPrefix(secret, prefix) {
			t.Errorf("expected secret to have prefix %q, but got %q", prefix, secret)
		}
	})

	t.Run("should return an empty string if no key is set", func(t *testing.T) {
		configNoKey := &Config{
			Encrypted: &Encrypted{
				AesKey: "",
				KindFields: map[string][][]string{
					"Secret": {{"data"}},
				},
			},
		}
		us := unstructured.Unstructured{Object: map[string]any{
			"data": map[string]any{
				"secret": "don't tell anyone!",
			},
		}}
		configNoKey.EncryptFields(&GroupResource{APIResource: metav1.APIResource{Kind: "Secret"}}, us)
		secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
		if secret != "" {
			t.Errorf("expected empty secret, but got %q", secret)
		}
	})

	t.Run("should not encrypt if already encrypted", func(t *testing.T) {
		_, config := setup()
		val := "KUBEXPORTER_AES@wKCCGma3NhnvzLMbMCrPK7nq7cQV6hF385YuqLjSk+UXCRgaQATO3PPUsfoheg=="
		us := unstructured.Unstructured{Object: map[string]any{
			"data": map[string]any{
				"secret": val,
			},
		}}
		res := &GroupResource{APIResource: metav1.APIResource{Kind: "Secret"}}
		config.EncryptFields(res, us)
		secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
		if secret != val {
			t.Errorf("expected secret %q, but got %q", val, secret)
		}
	})

	t.Run("should not encrypt empty strings", func(t *testing.T) {
		_, config := setup()
		us := unstructured.Unstructured{Object: map[string]any{
			"data": map[string]any{
				"secret": "",
			},
		}}
		res := &GroupResource{APIResource: metav1.APIResource{Kind: "Secret"}}
		config.EncryptFields(res, us)
		secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
		if secret != "" {
			t.Errorf("expected empty secret, but got %q", secret)
		}
	})
}

func TestDecryptFields(t *testing.T) {
	enc := &Encrypted{
		AesKey: "1234567890123456",
	}
	_ = enc.Setup()

	t.Run("should decrypt the value correctly", func(t *testing.T) {
		us := unstructured.Unstructured{Object: map[string]any{
			"data": map[string]any{
				"secret": "KUBEXPORTER_AES@wKCCGma3NhnvzLMbMCrPK7nq7cQV6hF385YuqLjSk+UXCRgaQATO3PPUsfoheg==",
			},
		}}
		cnt, err := decryptFields(us.Object, enc.gcm, len(enc.nonce))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cnt != 1 {
			t.Errorf("expected 1 decrypted field, but got %d", cnt)
		}
		secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
		if secret != "don't tell anyone!" {
			t.Errorf("expected decrypted secret \"don't tell anyone!\", but got %q", secret)
		}
	})

	t.Run("should not decrypt if not encrypted", func(t *testing.T) {
		us := unstructured.Unstructured{Object: map[string]any{
			"data": map[string]any{
				"secret": "don't tell anyone!",
			},
		}}
		cnt, err := decryptFields(us.Object, enc.gcm, len(enc.nonce))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cnt != 0 {
			t.Errorf("expected 0 decrypted fields, but got %d", cnt)
		}
		secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
		if secret != "don't tell anyone!" {
			t.Errorf("expected secret \"don't tell anyone!\", but got %q", secret)
		}
	})
}
