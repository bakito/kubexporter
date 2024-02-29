package types

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const prefix = "AES@"

type Encrypted struct {
	AesKey     string     `json:"aesKey" yaml:"aesKey"`
	KindFields KindFields `json:"kindFields" yaml:"kindFields"`
	gcm        cipher.AEAD
	nonce      []byte
}

func (e *Encrypted) Setup() error {
	if e.AesKey != "" {
		c, err := aes.NewCipher([]byte(e.AesKey))
		if err != nil {
			return err
		}

		e.gcm, err = cipher.NewGCM(c)
		if err != nil {
			return err
		}

		e.nonce = make([]byte, e.gcm.NonceSize())

		if _, err = io.ReadFull(rand.Reader, e.nonce); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encrypted) doEncrypt(val interface{}) string {
	if e.AesKey == "" {
		return ""
	}
	data := []byte(fmt.Sprintf("%v", val))
	return prefix + base64.StdEncoding.EncodeToString(e.gcm.Seal(e.nonce, e.nonce, data, nil))
}

// EncryptFields encrypts fields for a given resource
func (c *Config) EncryptFields(res *GroupResource, us unstructured.Unstructured) {
	transformNestedFields(c.Encrypted.KindFields, c.Encrypted.doEncrypt, res.GroupKind(), us)
}
