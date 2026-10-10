package types

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

const aesPrefix = "KUBEXPORTER_AES@"

// AesEncryptor handles AES-based encryption and decryption.
type AesEncryptor struct {
	gcm   cipher.AEAD
	nonce []byte
	key   string
}

// NewAesEncryptor creates a new AES encryptor with the given key.
func NewAesEncryptor(key string) (*AesEncryptor, error) {
	gcm, err := setupAES(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return &AesEncryptor{
		gcm:   gcm,
		nonce: nonce,
		key:   key,
	}, nil
}

// setupAES creates a new GCM cipher from the given key.
func setupAES(key string) (cipher.AEAD, error) {
	k := len(key)
	switch k {
	case 16, 24, 32:
	default:
		return nil, invalidAesKeyError(k)
	}

	c, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	return gcm, nil
}

// invalidAesKeyError returns an error for invalid AES key sizes.
func invalidAesKeyError(size int) error {
	return &AesKeyError{Size: size}
}

// AesKeyError is returned when an AES key has an invalid size.
type AesKeyError struct {
	Size int
}

func (e *AesKeyError) Error() string {
	return "invalid key size " + itoa(e.Size) + ": aesKey must be 16, 24 or 32 chars long"
}

// itoa converts an int to a string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	buf := make([]byte, 0, 20)
	for i > 0 {
		buf = append(buf, byte('0'+i%10))
		i /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	for left, right := 0, len(buf)-1; left < right; left, right = left+1, right-1 {
		buf[left], buf[right] = buf[right], buf[left]
	}
	return string(buf)
}

// Encrypt encrypts the given plaintext value using AES-GCM.
func (a *AesEncryptor) Encrypt(plaintext string) string {
	data := []byte(plaintext)
	return aesPrefix + base64.StdEncoding.EncodeToString(a.gcm.Seal(a.nonce, a.nonce, data, nil))
}

// Decrypt decrypts the given base64-encoded ciphertext value using AES-GCM.
// The input should be the base64-encoded ciphertext after the prefix is removed.
func (a *AesEncryptor) Decrypt(ciphertext string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	nonceSize := a.gcm.NonceSize()
	if len(decoded) < nonceSize {
		return "", invalidTextSizeError()
	}
	nonce, data := decoded[:nonceSize], decoded[nonceSize:]
	plaintext, err := a.gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// invalidTextSizeError returns an error when the ciphertext is too short.
func invalidTextSizeError() error {
	return &InvalidTextSizeError{}
}

// InvalidTextSizeError is returned when the ciphertext is too short to decrypt.
type InvalidTextSizeError struct{}

func (*InvalidTextSizeError) Error() string {
	return "invalid text size"
}
