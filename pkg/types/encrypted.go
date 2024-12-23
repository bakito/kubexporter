package types

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/bakito/kubexporter/pkg/render"
	"github.com/bakito/kubexporter/pkg/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	prefix    = "KUBEXPORTER_AES@"
	EnvAesKey = "KUBEXPORTER_AES_KEY"
)

type Encrypted struct {
	AesKey     string     `json:"aesKey"     yaml:"aesKey"`
	KindFields KindFields `json:"kindFields" yaml:"kindFields"`
	gcm        cipher.AEAD
	nonce      []byte
}

func (e *Encrypted) Setup() (err error) {
	if k, ok := os.LookupEnv(EnvAesKey); ok {
		e.AesKey = k
	}
	if e.AesKey != "" {
		e.gcm, err = setupAES(e.AesKey)
		if err != nil {
			return err
		}

		e.nonce = make([]byte, e.gcm.NonceSize())

		if _, err = io.ReadFull(rand.Reader, e.nonce); err != nil {
			return err
		}
	} else if len(e.KindFields) > 0 {
		return fmt.Errorf("encrypted mode needs a valid aesKey."+
			" please remove the 'encrypted config' or provide the 'aesKey' in the config of via env variable %q",
			EnvAesKey,
		)
	}
	return nil
}

func setupAES(key string) (cipher.AEAD, error) {
	k := len(key)
	switch k {
	default:
		return nil, fmt.Errorf("invalid key size %d: aesKey must be 16, 24 or 32 chars long", k)
	case 16, 24, 32:
		break
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

func Decrypt(printFlags *genericclioptions.PrintFlags, aesKey string, files ...string) error {
	gcm, err := setupAES(aesKey)
	if err != nil {
		return err
	}
	nonceSize := gcm.NonceSize()

	table := render.Table()
	table.SetHeader([]string{"File", "Namespace", "Kind", "Name", "Decrypted Fields"})

	for _, file := range files {
		us, err := utils.ReadFile(file)
		if err != nil {
			return err
		}
		if replaced, err := decryptFields(us.Object, gcm, nonceSize); err != nil {
			return err
		} else {
			table.Append([]string{file, us.GetNamespace(), us.GetKind(), us.GetName(), strconv.Itoa(replaced)})
		}

		if err := utils.WriteFile(printFlags, file, us); err != nil {
			return err
		}
	}

	table.Render()
	return nil
}

// transformNestedField transforms the nested field from the obj.
func decryptFields(obj map[string]interface{}, gcm cipher.AEAD, nonceSize int) (int, error) {
	var replaced int
	for key, value := range obj {
		switch e := value.(type) {
		case map[string]interface{}:
			if cnt, err := decryptFields(e, gcm, nonceSize); err != nil {
				return 0, err
			} else {
				replaced += cnt
			}
		case string:
			if strings.HasPrefix(e, prefix) {

				ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(e, prefix))
				if err != nil {
					return 0, err
				}

				if len(ciphertext) < nonceSize {
					return 0, errors.New("invalid text size")
				}
				nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
				plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
				if err != nil {
					return 0, err
				}
				obj[key] = string(plaintext)
				replaced++
			}
		}
	}
	return replaced, nil
}
