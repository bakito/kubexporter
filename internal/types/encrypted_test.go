package types

import (
	"strings"
	"testing"

	"filippo.io/age"
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
				if enc.aes == nil {
					t.Error("Encrypted.aes is nil")
				}
			} else {
				if enc.aes != nil {
					t.Error("Encrypted.aes is not nil")
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
				if !strings.HasPrefix(got, aesPrefix) {
					t.Errorf("expected secret to have prefix %q, but got %q", aesPrefix, got)
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
			name:   "should not encrypt if already age encrypted",
			aesKey: "1234567890123456",
			input:  "KUBEXPORTER_AGE@-----BEGIN AGE ENCRYPTED FILE-----",
			validate: func(t *testing.T, got string) {
				t.Helper()
				expected := "KUBEXPORTER_AGE@-----BEGIN AGE ENCRYPTED FILE-----"
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
			res, err := decryptFields(us.Object, enc.aes, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.count != tt.expectedCount {
				t.Errorf("expected %d decrypted field, but got %d", tt.expectedCount, res.count)
			}
			secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
			if secret != tt.expected {
				t.Errorf("expected %q, but got %q", tt.expected, secret)
			}
		})
	}
}

// TestEncrypted_Setup_Age tests age encryption setup.
func TestEncrypted_Setup_Age(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate age identity: %v", err)
	}

	tests := []struct {
		name         string
		agePublicKey string
		ageIdentity  string
		kindFields   KindFields
		wantErr      bool
	}{
		{
			name:         "age public key setup should succeed",
			agePublicKey: identity.Recipient().String(),
		},
		{
			name:        "age identity setup should succeed",
			ageIdentity: identity.String(),
		},
		{
			name:         "invalid age public key should fail",
			agePublicKey: "age1invalidkey123",
			wantErr:      true,
		},
		{
			name:         "no key with kind fields should fail",
			agePublicKey: "",
			kindFields:   KindFields{"Secret": {{"data"}}},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := &Encrypted{
				AgePublicKey: tt.agePublicKey,
				AgeIdentity:  tt.ageIdentity,
				KindFields:   tt.kindFields,
			}

			err := enc.Setup()
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypted.Setup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.agePublicKey != "" {
				if enc.age == nil || !enc.age.HasRecipients() {
					t.Error("expected age recipients to be set")
				}
			}
			if !tt.wantErr && tt.ageIdentity != "" {
				if enc.age == nil || !enc.age.HasIdentities() {
					t.Error("expected age identities to be set")
				}
			}
		})
	}
}

// TestEncrypted_Age_RoundTrip tests age encryption and decryption roundtrip.
func TestEncrypted_Age_RoundTrip(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate age identity: %v", err)
	}
	publicKey := identity.Recipient().String()

	enc := &Encrypted{
		AgePublicKey: publicKey,
		KindFields: KindFields{
			"Secret": {{"data"}, {"stringData"}},
		},
	}
	if err := enc.Setup(); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	plaintext := "super secret password"
	encrypted := enc.doEncrypt(plaintext)

	if !strings.HasPrefix(encrypted, agePrefix) {
		t.Errorf("expected encrypted value to have prefix %q, but got %q", agePrefix, encrypted)
	}

	afterPrefix, _ := strings.CutPrefix(encrypted, agePrefix)
	ageEnc, err := NewAgeEncryptor("", identity.String())
	if err != nil {
		t.Fatalf("NewAgeEncryptor failed: %v", err)
	}
	decrypted, err := ageEnc.Decrypt(afterPrefix)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("expected %q, but got %q", plaintext, decrypted)
	}
}

// TestDecryptFields_Age tests decryptFields with age encrypted values.
func TestDecryptFields_Age(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate age identity: %v", err)
	}
	publicKey := identity.Recipient().String()

	enc := &Encrypted{
		AgePublicKey: publicKey,
	}
	if err := enc.Setup(); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	plaintext := "my secret value"
	encrypted := enc.doEncrypt(plaintext)

	ageEnc, err := NewAgeEncryptor("", identity.String())
	if err != nil {
		t.Fatalf("NewAgeEncryptor failed: %v", err)
	}

	tests := []struct {
		name          string
		input         string
		expected      string
		expectedCount int
	}{
		{
			name:          "should decrypt age encrypted value",
			input:         encrypted,
			expected:      plaintext,
			expectedCount: 1,
		},
		{
			name:          "should not decrypt plaintext",
			input:         "just plain text",
			expected:      "just plain text",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := map[string]any{
				"data": map[string]any{
					"secret": tt.input,
				},
			}
			res, err := decryptFields(obj, nil, ageEnc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.count != tt.expectedCount {
				t.Errorf("expected %d decrypted field, but got %d", tt.expectedCount, res.count)
			}
			secret, _, _ := unstructured.NestedString(obj, "data", "secret")
			if secret != tt.expected {
				t.Errorf("expected %q, but got %q", tt.expected, secret)
			}
		})
	}
}

// TestDecryptFields_Mixed tests decryptFields with both AES and age encrypted values.
func TestDecryptFields_Mixed(t *testing.T) {
	aesEncrypted := &Encrypted{
		AesKey: "1234567890123456",
	}
	if err := aesEncrypted.Setup(); err != nil {
		t.Fatalf("AES Setup failed: %v", err)
	}

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate age identity: %v", err)
	}
	ageEncrypted := &Encrypted{
		AgePublicKey: identity.Recipient().String(),
	}
	if err := ageEncrypted.Setup(); err != nil {
		t.Fatalf("Age Setup failed: %v", err)
	}

	aesPlaintext := "aes secret"
	aesEncryptedVal := aesEncrypted.doEncrypt(aesPlaintext)

	agePlaintext := "age secret"
	ageEncryptedVal := ageEncrypted.doEncrypt(agePlaintext)

	obj := map[string]any{
		"data": map[string]any{
			"aes-field": aesEncryptedVal,
			"age-field": ageEncryptedVal,
		},
	}

	ageDec, err := NewAgeEncryptor("", identity.String())
	if err != nil {
		t.Fatalf("NewAgeEncryptor failed: %v", err)
	}

	res, err := decryptFields(obj, aesEncrypted.aes, ageDec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.count != 2 {
		t.Errorf("expected 2 decrypted fields, but got %d", res.count)
	}

	aesDecrypted, _, _ := unstructured.NestedString(obj, "data", "aes-field")
	if aesDecrypted != aesPlaintext {
		t.Errorf("expected AES decrypted value %q, but got %q", aesPlaintext, aesDecrypted)
	}

	ageDecrypted, _, _ := unstructured.NestedString(obj, "data", "age-field")
	if ageDecrypted != agePlaintext {
		t.Errorf("expected age decrypted value %q, but got %q", agePlaintext, ageDecrypted)
	}
}

// TestCountEncryptedFields_Age tests countEncryptedFields with age prefix.
func TestCountEncryptedFields_Age(t *testing.T) {
	obj := map[string]any{
		"data": map[string]any{
			"aes-field": "KUBEXPORTER_AES@somebase64data==",
			"age-field": "KUBEXPORTER_AGE@-----BEGIN AGE ENCRYPTED FILE-----\nsome data",
			"plain":     "not encrypted",
		},
	}

	count := countEncryptedFields(obj)
	if count != 2 {
		t.Errorf("expected 2 encrypted fields, but got %d", count)
	}
}
