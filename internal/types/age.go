package types

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
)

const agePrefix = "KUBEXPORTER_AGE@"

// AgeEncryptor handles age-based encryption and decryption.
type AgeEncryptor struct {
	recipients []age.Recipient
	identities []age.Identity
}

// NewAgeEncryptor creates a new age encryptor with the given public key and/or identity.
func NewAgeEncryptor(publicKey, identity string) (*AgeEncryptor, error) {
	e := &AgeEncryptor{}

	if publicKey != "" {
		recipient, err := age.ParseX25519Recipient(publicKey)
		if err != nil {
			return nil, fmt.Errorf("invalid age public key: %w", err)
		}
		e.recipients = []age.Recipient{recipient}
	}

	if identity != "" {
		identities, err := age.ParseIdentities(strings.NewReader(identity))
		if err != nil {
			return nil, fmt.Errorf("invalid age identity: %w", err)
		}
		e.identities = identities
	}

	return e, nil
}

// Encrypt encrypts the given plaintext value using age encryption with armor encoding.
func (a *AgeEncryptor) Encrypt(plaintext string) string {
	if len(a.recipients) == 0 {
		return plaintext
	}

	var buf bytes.Buffer
	w := armor.NewWriter(&buf)
	wc, err := age.Encrypt(w, a.recipients...)
	if err != nil {
		return plaintext
	}
	if _, err := io.WriteString(wc, plaintext); err != nil {
		return plaintext
	}
	if err := wc.Close(); err != nil {
		return plaintext
	}
	if err := w.Close(); err != nil {
		return plaintext
	}

	return agePrefix + buf.String()
}

// Decrypt decrypts the given armored age-encrypted value.
func (a *AgeEncryptor) Decrypt(encryptedData string) (string, error) {
	reader := armor.NewReader(strings.NewReader(encryptedData))
	decrypted, err := age.Decrypt(reader, a.identities...)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, decrypted); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// HasRecipients returns true if the encryptor has been configured with recipients for encryption.
func (a *AgeEncryptor) HasRecipients() bool {
	return len(a.recipients) > 0
}

// HasIdentities returns true if the encryptor has been configured with identities for decryption.
func (a *AgeEncryptor) HasIdentities() bool {
	return len(a.identities) > 0
}
