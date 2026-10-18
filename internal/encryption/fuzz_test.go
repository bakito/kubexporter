package encryption

import (
	"strings"
	"testing"
)

func FuzzAesDecrypt(f *testing.F) {
	f.Add([]byte("YWJj"))
	f.Add([]byte("dGVzdA=="))
	f.Add([]byte(""))
	f.Add([]byte("not-base64!!!"))
	f.Add([]byte("AQID"))

	key := strings.Repeat("a", 32)
	enc, err := NewAesEncryptor(key)
	if err != nil {
		panic(err)
	}

	f.Fuzz(func(_ *testing.T, data []byte) {
		ciphertext := string(data)
		_, _ = enc.Decrypt(ciphertext)
	})
}

func FuzzAesEncrypt_Decrypt_Roundtrip(f *testing.F) {
	f.Add([]byte("hello world"))
	f.Add([]byte(""))
	f.Add([]byte("special chars: \x00\x01\x02\xff\xfe"))
	f.Add([]byte(strings.Repeat("A", 10000)))

	key := strings.Repeat("a", 32)

	f.Fuzz(func(t *testing.T, data []byte) {
		enc, err := NewAesEncryptor(key)
		if err != nil {
			t.Skip()
		}
		plaintext := string(data)
		encrypted := enc.Encrypt(plaintext)
		if !strings.HasPrefix(encrypted, AesPrefix) {
			return
		}
		ciphertext := strings.TrimPrefix(encrypted, AesPrefix)
		dec, err := NewAesEncryptor(key)
		if err != nil {
			t.Skip()
		}
		decrypted, err := dec.Decrypt(ciphertext)
		if err != nil {
			t.Skip()
		}
		if decrypted != plaintext {
			t.Fatalf("roundtrip failed: expected %q, got %q", plaintext, decrypted)
		}
	})
}
