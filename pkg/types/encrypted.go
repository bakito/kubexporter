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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/bakito/kubexporter/pkg/render"
	"github.com/bakito/kubexporter/pkg/utils"
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
	case 16, 24, 32:
	default:
		return nil, fmt.Errorf("invalid key size %d: aesKey must be 16, 24 or 32 chars long", k)
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

func (e *Encrypted) doEncrypt(val any) string {
	if e.AesKey == "" {
		return ""
	}

	// Convert to string
	strVal := fmt.Sprintf("%v", val)

	// Don't encrypt if already encrypted or empty
	if strings.HasPrefix(strVal, prefix) || strVal == "" {
		return strVal
	}

	data := []byte(strVal)
	return prefix + base64.StdEncoding.EncodeToString(e.gcm.Seal(e.nonce, e.nonce, data, nil))
}

// EncryptFields encrypts fields for a given resource.
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
	table.Header("File", "Namespace", "Kind", "Name", "Decrypted Fields")

	for _, file := range files {
		us, err := utils.ReadFile(file)
		if err != nil {
			return err
		}
		var replaced int
		if replaced, err = decryptFields(us.Object, gcm, nonceSize); err != nil {
			return err
		}
		if err := table.Append([]string{file, us.GetNamespace(), us.GetKind(), us.GetName(), strconv.Itoa(replaced)}); err != nil {
			return err
		}

		if err := utils.WriteFile(printFlags, file, us); err != nil {
			return err
		}
	}

	return table.Render()
}

// Encrypt encrypts secrets in exported resource files.
func Encrypt(printFlags *genericclioptions.PrintFlags, aesKey string, files ...string) error {
	// Create a config with encryption settings for Secrets only
	// TODO: it could read the config from the file for flexibility
	config := &Config{
		Encrypted: &Encrypted{
			AesKey: aesKey,
			KindFields: KindFields{
				"Secret": {{"data"}, {"stringData"}},
			},
		},
	}
	if err := config.Encrypted.Setup(); err != nil {
		return err
	}

	table := render.Table()
	table.Header("File", "Namespace", "Kind", "Name", "Encrypted Fields")

	for _, file := range files {
		us, err := utils.ReadFile(file)
		if err != nil {
			return err
		}

		res := &GroupResource{
			APIResource: metav1.APIResource{
				Kind: us.GetKind(),
			},
		}
		config.EncryptFields(res, *us)
		encryptedCount := countEncryptedFields(us.Object)

		if err := table.Append([]string{file, us.GetNamespace(), us.GetKind(), us.GetName(), strconv.Itoa(encryptedCount)}); err != nil {
			return err
		}

		if err := utils.WriteFile(printFlags, file, us); err != nil {
			return err
		}
	}

	return table.Render()
}

// transformNestedField transforms the nested field from the obj.
func decryptFields(obj map[string]any, gcm cipher.AEAD, nonceSize int) (int, error) {
	var replaced int
	for key, value := range obj {
		switch e := value.(type) {
		case map[string]any:
			var cnt int
			var err error
			if cnt, err = decryptFields(e, gcm, nonceSize); err != nil {
				return 0, err
			}
			replaced += cnt
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

// countEncryptedFields counts the number of fields that have been encrypted.
func countEncryptedFields(obj map[string]any) int {
	var count int
	for _, value := range obj {
		switch e := value.(type) {
		case map[string]any:
			count += countEncryptedFields(e)
		case string:
			if strings.HasPrefix(e, prefix) {
				count++
			}
		}
	}
	return count
}
