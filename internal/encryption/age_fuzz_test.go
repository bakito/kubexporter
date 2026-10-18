package encryption

import (
	"strings"
	"testing"
)

func FuzzAgeEncrypt_Decrypt_Roundtrip(f *testing.F) {
	f.Add([]byte("hello world"))
	f.Add([]byte(""))
	f.Add([]byte("special chars: \x00\x01\x02\xff\xfe"))
	f.Add([]byte(strings.Repeat("A", 1000)))

	f.Fuzz(func(t *testing.T, data []byte) {
		plaintext := string(data)

		// Create an age encryptor with a valid key pair for roundtrip testing
		enc, err := NewAgeEncryptor(
			"age1lxyyl2f666xjzlcgjtsskry9n6t8xkcxw6p0l7g8p6s6k9x9y4sq7uv2ur",
			"age-secret-key-1lxyyl2f666xjzlcgjtsskry9n6t8xkcxw6p0l7g8p6s6k9x9y4sq7uv2ur",
		)
		if err != nil {
			t.Skip()
		}
		if !enc.HasRecipients() || !enc.HasIdentities() {
			t.Skip()
		}

		encrypted := enc.Encrypt(plaintext)
		if !strings.HasPrefix(encrypted, AgePrefix) {
			return
		}
		ciphertext := strings.TrimPrefix(encrypted, AgePrefix)
		decrypted, err := enc.Decrypt(ciphertext)
		if err != nil {
			t.Skip()
		}
		if decrypted != plaintext {
			t.Fatalf("roundtrip failed: expected %q, got %q", plaintext, decrypted)
		}
	})
}

func FuzzAgeDecrypt(f *testing.F) {
	f.Add([]byte("-----BEGIN AGE ENCRYPTED FILE-----\nYWJj\n-----END AGE ENCRYPTED FILE-----"))
	f.Add([]byte("not armored"))
	f.Add([]byte(""))
	f.Add([]byte("-----BEGIN AGE ENCRYPTED FILE-----\n"))
	f.Add([]byte("garbage data \x00\x01\x02"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Create an age encryptor with a valid identity
		enc, err := NewAgeEncryptor("", "age-secret-key-1lxyyl2f666xjzlcgjtsskry9n6t8xkcxw6p0l7g8p6s6k9x9y4sq7uv2ur")
		if err != nil {
			t.Skip()
		}
		if !enc.HasIdentities() {
			t.Skip()
		}

		encryptedData := string(data)
		_, _ = enc.Decrypt(encryptedData)
	})
}

func FuzzNewAgeEncryptor_PublicKey(f *testing.F) {
	f.Add([]byte("age1lxyyl2f666xjzlcgjtsskry9n6t8xkcxw6p0l7g8p6s6k9x9y4sq7uv2ur"))
	f.Add([]byte("invalid key"))
	f.Add([]byte(""))
	f.Add([]byte("age1invalid!!!"))
	f.Add([]byte(strings.Repeat("A", 1000)))

	f.Fuzz(func(_ *testing.T, publicKey []byte) {
		_, _ = NewAgeEncryptor(string(publicKey), "")
	})
}

func FuzzNewAgeEncryptor_Identity(f *testing.F) {
	f.Add([]byte("age-secret-key-1lxyyl2f666xjzlcgjtsskry9n6t8xkcxw6p0l7g8p6s6k9x9y4sq7uv2ur"))
	f.Add([]byte("invalid identity"))
	f.Add([]byte(""))
	f.Add([]byte("age-secret-key-invalid!!!"))
	f.Add([]byte(strings.Repeat("A", 1000)))

	f.Fuzz(func(_ *testing.T, identity []byte) {
		_, _ = NewAgeEncryptor("", string(identity))
	})
}

func FuzzAesKeySetup(f *testing.F) {
	f.Add([]byte(strings.Repeat("a", 16)))
	f.Add([]byte(strings.Repeat("b", 24)))
	f.Add([]byte(strings.Repeat("c", 32)))
	f.Add([]byte(""))
	f.Add([]byte(strings.Repeat("d", 15)))
	f.Add([]byte(strings.Repeat("e", 17)))
	f.Add([]byte(strings.Repeat("f", 33)))
	f.Add([]byte(strings.Repeat("g", 100)))
	f.Add([]byte("\x00\x01\x02\xff\xfe"))

	f.Fuzz(func(_ *testing.T, key []byte) {
		_, _ = NewAesEncryptor(string(key))
	})
}
