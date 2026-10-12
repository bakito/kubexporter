package encryption

import (
	"errors"
	"strings"
	"testing"

	"filippo.io/age"
)

// --- AES Encryptor Tests ---

func TestAesPrefix(t *testing.T) {
	if AesPrefix != "KUBEXPORTER_AES@" {
		t.Errorf("expected AesPrefix to be \"KUBEXPORTER_AES@\", got %q", AesPrefix)
	}
}

func TestNewAesEncryptor_valid_keys(t *testing.T) {
	validSizes := []int{16, 24, 32}
	for _, size := range validSizes {
		key := strings.Repeat("a", size)
		enc, err := NewAesEncryptor(key)
		if err != nil {
			t.Errorf("key size %d: expected no error, got %v", size, err)
		}
		if enc == nil {
			t.Errorf("key size %d: expected encryptor, got nil", size)
		}
	}
}

func TestNewAesEncryptor_invalid_keys(t *testing.T) {
	invalidSizes := []int{0, 15, 17, 31, 33}
	for _, size := range invalidSizes {
		key := strings.Repeat("a", size)
		enc, err := NewAesEncryptor(key)
		if err == nil {
			t.Errorf("key size %d: expected error, got nil", size)
		}
		if enc != nil {
			t.Errorf("key size %d: expected nil encryptor, got non-nil", size)
		}
	}
}

func TestAesEncryptor_Encrypt_Decrypt_Roundtrip(t *testing.T) {
	key := strings.Repeat("a", 32)
	enc, err := NewAesEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plaintext := "secret data to encrypt"
	encrypted := enc.Encrypt(plaintext)

	// Verify prefix
	if !strings.HasPrefix(encrypted, AesPrefix) {
		t.Errorf("encrypted value should have prefix %q, got %q", AesPrefix, encrypted)
	}

	// Extract ciphertext after prefix
	ciphertext := strings.TrimPrefix(encrypted, AesPrefix)

	// Create a new encryptor to decrypt
	dec, err := NewAesEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create decryptor: %v", err)
	}

	decrypted, err := dec.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestAesEncryptor_Decrypt_wrong_key(t *testing.T) {
	key1 := strings.Repeat("a", 32)
	key2 := strings.Repeat("b", 32)

	enc1, _ := NewAesEncryptor(key1)
	enc2, _ := NewAesEncryptor(key2)

	plaintext := "secret data"
	encrypted := enc1.Encrypt(plaintext)
	ciphertext := strings.TrimPrefix(encrypted, AesPrefix)

	_, err := enc2.Decrypt(ciphertext)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestAesEncryptor_Decrypt_invalid_base64(t *testing.T) {
	key := strings.Repeat("a", 32)
	enc, _ := NewAesEncryptor(key)

	_, err := enc.Decrypt("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestAesEncryptor_Decrypt_too_short(t *testing.T) {
	key := strings.Repeat("a", 32)
	enc, _ := NewAesEncryptor(key)

	// "YWJj" is base64 for "abc" (3 bytes), which is shorter than the nonce size (12 bytes)
	_, err := enc.Decrypt("YWJj")
	if err == nil {
		t.Error("expected InvalidTextSizeError for too-short ciphertext")
	}
	invalidTextSizeError := &InvalidTextSizeError{}
	ok := errors.As(err, &invalidTextSizeError)
	if !ok {
		t.Errorf("expected *InvalidTextSizeError, got %T", err)
	}
}

func TestAesKeyError_Error(t *testing.T) {
	e := &AesKeyError{Size: 15}
	msg := e.Error()
	expected := "invalid key size 15: aesKey must be 16, 24 or 32 chars long"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

func TestInvalidTextSizeError_Error(t *testing.T) {
	e := &InvalidTextSizeError{}
	msg := e.Error()
	if msg != "invalid text size" {
		t.Errorf("expected \"invalid text size\", got %q", msg)
	}
}

func Test_itoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{5, "5"},
		{42, "42"},
		{100, "100"},
		{-7, "-7"},
		{-123, "-123"},
	}
	for _, tt := range tests {
		if got := itoa(tt.input); got != tt.expected {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// --- Age Encryptor Tests ---

func TestAgePrefix(t *testing.T) {
	if AgePrefix != "KUBEXPORTER_AGE@" {
		t.Errorf("expected AgePrefix to be \"KUBEXPORTER_AGE@\", got %q", AgePrefix)
	}
}

func TestNewAgeEncryptor_valid_keys(t *testing.T) {
	secretKey, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey := secretKey.Recipient()

	enc, err := NewAgeEncryptor(publicKey.String(), secretKey.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if enc == nil {
		t.Fatal("expected encryptor, got nil")
	}
	if !enc.HasRecipients() {
		t.Error("expected HasRecipients to be true")
	}
	if !enc.HasIdentities() {
		t.Error("expected HasIdentities to be true")
	}
}

func TestNewAgeEncryptor_invalid_public_key(t *testing.T) {
	_, err := NewAgeEncryptor("not-a-valid-age-key", "")
	if err == nil {
		t.Error("expected error for invalid public key")
	}
}

func TestNewAgeEncryptor_invalid_secret_key(t *testing.T) {
	_, err := NewAgeEncryptor("", "not-a-valid-age-secret-key")
	if err == nil {
		t.Error("expected error for invalid secret key")
	}
	// Verify the error mentions the prefix requirement
	if !strings.Contains(err.Error(), "must start with") {
		t.Errorf("expected error to mention prefix, got %q", err.Error())
	}
}

func TestAgeSecretKeyPrefix(t *testing.T) {
	if AgeSecretKeyPrefix != "AGE-SECRET-KEY" {
		t.Errorf("expected AgeSecretKeyPrefix to be %q, got %q", "AGE-SECRET-KEY", AgeSecretKeyPrefix)
	}
}

func TestNewAgeEncryptor_secret_key_prefix_validation(t *testing.T) {
	// Test that a key with wrong prefix is rejected
	_, err := NewAgeEncryptor("", "WRONG-PREFIX-KEY")
	if err == nil {
		t.Error("expected error for secret key with wrong prefix")
	}

	// Test that a valid X25519 secret key passes prefix check
	secretKey, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	keyStr := secretKey.String()
	if !strings.HasPrefix(keyStr, AgeSecretKeyPrefix) {
		t.Errorf("generated secret key should have prefix %q, got %q", AgeSecretKeyPrefix, keyStr)
	}

	// The valid key should pass
	enc, err := NewAgeEncryptor("", keyStr)
	if err != nil {
		t.Fatalf("expected no error for valid secret key, got %v", err)
	}
	if !enc.HasIdentities() {
		t.Error("expected HasIdentities to be true for valid secret key")
	}
}

func TestNewAgeEncryptor_no_keys(t *testing.T) {
	enc, err := NewAgeEncryptor("", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if enc == nil {
		t.Fatal("expected encryptor, got nil")
	}
	if enc.HasRecipients() {
		t.Error("expected HasRecipients to be false")
	}
	if enc.HasIdentities() {
		t.Error("expected HasIdentities to be false")
	}
}

func TestAgeEncryptor_Encrypt_no_recipients(t *testing.T) {
	enc, _ := NewAgeEncryptor("", "")
	plaintext := "plain text"
	result := enc.Encrypt(plaintext)
	if result != plaintext {
		t.Errorf("expected plaintext unchanged, got %q", result)
	}
}

func TestAgeEncryptor_Encrypt_Decrypt_Roundtrip(t *testing.T) {
	secretKey, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey := secretKey.Recipient()

	enc, err := NewAgeEncryptor(publicKey.String(), secretKey.String())
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plaintext := "secret data to encrypt with age"
	encrypted := enc.Encrypt(plaintext)

	// Verify prefix
	if !strings.HasPrefix(encrypted, AgePrefix) {
		t.Errorf("encrypted value should have prefix %q, got %q", AgePrefix, encrypted)
	}

	// Extract armored text after prefix
	armored := strings.TrimPrefix(encrypted, AgePrefix)

	// Decrypt
	decrypted, err := enc.Decrypt(armored)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("expected %q, got %q", plaintext, decrypted)
	}
}
