package types

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/bakito/kubexporter/pkg/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const prefix = "AES@"

type Encrypted struct {
	AesKey     string     `json:"aesKey" yaml:"aesKey"`
	KindFields KindFields `json:"kindFields" yaml:"kindFields"`
	gcm        cipher.AEAD
	nonce      []byte
}

func (e *Encrypted) Setup() (err error) {
	if e.AesKey != "" {
		e.gcm, err = setupAES(e.AesKey)
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

func setupAES(key string) (cipher.AEAD, error) {
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

func Decrypt(aesKey string, files ...string) error {
	gcm, err := setupAES(aesKey)
	if err != nil {
		return err
	}
	nonceSize := gcm.NonceSize()

	for _, file := range files {
		us, err := utils.ReadFile(file)
		if err != nil {
			return err
		}
		if err := decryptFields(us.Object, gcm, nonceSize); err != nil {
			return err
		}
	}
	return nil
}

// transformNestedField transforms the nested field from the obj.
func decryptFields(obj map[string]interface{}, gcm cipher.AEAD, nonceSize int) error {
	m := obj
	for key, value := range m {
		switch e := value.(type) {
		case map[string]interface{}:
			if err := decryptFields(e, gcm, nonceSize); err != nil {
				return err
			}
		case string:
			if strings.HasPrefix(e, prefix) {

				ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(e, prefix))
				if err != nil {
					return err
				}

				if len(ciphertext) < nonceSize {
					return errors.New("invalid text size")
				}
				nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
				plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
				if err != nil {
					return err
				}
				m[key] = string(plaintext)
			}
		}
	}
	return nil
}
