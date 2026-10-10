package types

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/bakito/kubexporter/internal/render"
	"github.com/bakito/kubexporter/internal/utils"
)

const (
	EnvAesKey       = "KUBEXPORTER_AES_KEY"
	EnvAgePublicKey = "KUBEXPORTER_AGE_PUBLIC_KEY"
	EnvAgeIdentity  = "KUBEXPORTER_AGE_IDENTITY"
)

// LookupEnvAgePublicKey returns the age public key from the environment variable.
func LookupEnvAgePublicKey() (string, bool) {
	return os.LookupEnv(EnvAgePublicKey)
}

// Encrypted holds configuration for encrypting and decrypting resource fields.
// It supports both AES and age encryption, using the corresponding encryptor types.
type Encrypted struct {
	AesKey       string     `docs:"The AES key to use for field encryption"       json:"aesKey"       yaml:"aesKey"`
	AgePublicKey string     `docs:"The age public key for encryption (recipient)" json:"agePublicKey" yaml:"agePublicKey"`
	AgeIdentity  string     `docs:"The age identity key for decryption"           json:"ageIdentity"  yaml:"ageIdentity"`
	KindFields   KindFields `docs:"The fields to encrypt for each kind"           json:"kindFields"   yaml:"kindFields"`

	aes *AesEncryptor
	age *AgeEncryptor
}

// Setup initializes the encrypted config, resolving environment variables and
// creating the appropriate encryptors.
func (e *Encrypted) Setup() error {
	if k, ok := os.LookupEnv(EnvAesKey); ok {
		e.AesKey = k
	}
	if pk, ok := os.LookupEnv(EnvAgePublicKey); ok {
		e.AgePublicKey = pk
	}
	if id, ok := os.LookupEnv(EnvAgeIdentity); ok {
		e.AgeIdentity = id
	}

	if e.AesKey != "" {
		aes, err := NewAesEncryptor(e.AesKey)
		if err != nil {
			return err
		}
		e.aes = aes
	}

	if e.AgePublicKey != "" || e.AgeIdentity != "" {
		age, err := NewAgeEncryptor(e.AgePublicKey, e.AgeIdentity)
		if err != nil {
			return err
		}
		e.age = age
	}

	if len(e.KindFields) > 0 && e.AesKey == "" && e.AgePublicKey == "" {
		return fmt.Errorf("encrypted mode needs a valid aesKey or agePublicKey."+
			" please remove the 'encrypted config' or provide the 'aesKey' via env variable %q or 'agePublicKey' via env variable %q",
			EnvAesKey, EnvAgePublicKey,
		)
	}
	return nil
}

// doEncrypt encrypts the given value using the configured encryption method.
// Age encryption is preferred over AES when both are available.
func (e *Encrypted) doEncrypt(val any) string {
	strVal := fmt.Sprintf("%v", val)

	// Don't encrypt if already encrypted or empty
	if strings.HasPrefix(strVal, aesPrefix) ||
		strings.HasPrefix(strVal, agePrefix) || strVal == "" {
		return strVal
	}

	// Use age encryption if configured
	if e.age != nil && e.age.HasRecipients() {
		return e.age.Encrypt(strVal)
	}

	// Use AES encryption if configured
	if e.aes != nil {
		return e.aes.Encrypt(strVal)
	}

	return ""
}

// EncryptFields encrypts fields for a given resource.
func (c *Config) EncryptFields(res *GroupResource, us unstructured.Unstructured) {
	transformNestedFields(c.Encrypted.KindFields, c.Encrypted.doEncrypt, res.GroupKind(), us)
}

// EncryptWithAge encrypts secrets in exported resource files using age encryption.
func EncryptWithAge(printFlags *genericclioptions.PrintFlags, agePublicKey string, files ...string) error {
	config := &Config{
		Encrypted: &Encrypted{
			AgePublicKey: agePublicKey,
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

		err = table.Append(
			[]string{file, us.GetNamespace(), us.GetKind(), us.GetName(), strconv.Itoa(encryptedCount)})
		if err != nil {
			return err
		}

		if err := utils.WriteFile(printFlags, file, us); err != nil {
			return err
		}
	}

	return table.Render()
}

func Decrypt(printFlags *genericclioptions.PrintFlags, aesKey, ageIdentity string, files ...string) error {
	var aesEnc *AesEncryptor
	var ageEnc *AgeEncryptor

	if aesKey != "" {
		var err error
		aesEnc, err = NewAesEncryptor(aesKey)
		if err != nil {
			return err
		}
	}

	if ageIdentity != "" {
		var err error
		ageEnc, err = NewAgeEncryptor("", ageIdentity)
		if err != nil {
			return err
		}
	}

	if aesEnc == nil && (!ageEnc.HasIdentities()) {
		return errors.New("decrypt requires either an AES key or an age identity")
	}

	table := render.Table()
	table.Header("File", "Namespace", "Kind", "Name", "Algorithm", "Decrypted Fields")

	for _, file := range files {
		us, err := utils.ReadFile(file)
		if err != nil {
			return err
		}
		var res decryptResult
		if res, err = decryptFields(us.Object, aesEnc, ageEnc); err != nil {
			return err
		}

		algorithm := ""
		if res.hasAES && res.hasAge {
			algorithm = "AES + age"
		} else if res.hasAES {
			algorithm = "AES"
		} else if res.hasAge {
			algorithm = "age"
		}

		err = table.Append(
			[]string{file, us.GetNamespace(), us.GetKind(), us.GetName(), algorithm, strconv.Itoa(res.count)})
		if err != nil {
			return err
		}

		if err := utils.WriteFile(printFlags, file, us); err != nil {
			return err
		}
	}

	return table.Render()
}

// Encrypt encrypts secrets in exported resource files using AES encryption.
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
	table.Header("File", "Namespace", "Kind", "Name", "Algorithm", "Encrypted Fields")

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

		err = table.Append(
			[]string{file, us.GetNamespace(), us.GetKind(), us.GetName(), "AES", strconv.Itoa(encryptedCount)})
		if err != nil {
			return err
		}

		if err := utils.WriteFile(printFlags, file, us); err != nil {
			return err
		}
	}

	return table.Render()
}

// decryptResult holds the result of decrypting fields.
type decryptResult struct {
	count   int
	hasAES  bool
	hasAge  bool
}

// decryptFields recursively decrypts encrypted fields in the given object.
func decryptFields(obj map[string]any, aesEnc *AesEncryptor, ageEnc *AgeEncryptor) (decryptResult, error) {
	var res decryptResult
	for key, value := range obj {
		switch v := value.(type) {
		case map[string]any:
			child, err := decryptFields(v, aesEnc, ageEnc)
			if err != nil {
				return res, err
			}
			res.count += child.count
			res.hasAES = res.hasAES || child.hasAES
			res.hasAge = res.hasAge || child.hasAge
		case string:
			// Try AES decryption
			if aesEnc != nil {
				if after, ok := strings.CutPrefix(v, aesPrefix); ok {
					plaintext, err := aesEnc.Decrypt(after)
					if err != nil {
						return res, err
					}
					obj[key] = plaintext
					res.count++
					res.hasAES = true
					continue
				}
			}

			// Try age decryption
			if ageEnc != nil && ageEnc.HasIdentities() {
				if after, ok := strings.CutPrefix(v, agePrefix); ok {
					plaintext, err := ageEnc.Decrypt(after)
					if err != nil {
						return res, fmt.Errorf("age decryption failed: %w", err)
					}
					obj[key] = plaintext
					res.count++
					res.hasAge = true
					continue
				}
			}
		}
	}
	return res, nil
}

// countEncryptedFields counts the number of fields that have been encrypted.
func countEncryptedFields(obj map[string]any) int {
	var count int
	for _, value := range obj {
		switch e := value.(type) {
		case map[string]any:
			count += countEncryptedFields(e)
		case string:
			if strings.HasPrefix(e, aesPrefix) || strings.HasPrefix(e, agePrefix) {
				count++
			}
		}
	}
	return count
}
