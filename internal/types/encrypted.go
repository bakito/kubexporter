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

	"github.com/bakito/kubexporter/internal/encryption"
	"github.com/bakito/kubexporter/internal/render"
	"github.com/bakito/kubexporter/internal/utils"
)

const (
	// EnvAesKey is the environment variable for the AES key.
	EnvAesKey = encryption.EnvAesKey
	// EnvAgePublicKey is the environment variable for the age public key.
	EnvAgePublicKey = encryption.EnvAgePublicKey
	// EnvAgeIdentity is the environment variable for the age identity.
	EnvAgeIdentity = encryption.EnvAgeIdentity
)

// Encryptor is the interface for encrypting values.
type Encryptor interface {
	Encrypt(plaintext string) string
}

// Decryptor is the interface for decrypting values.
type Decryptor interface {
	Decrypt(ciphertext string) (string, error)
}

// AesEncryptorInterface is the interface for AES encryption.
type AesEncryptorInterface interface {
	Encryptor
	Decryptor
}

// AgeEncryptorInterface is the interface for age encryption.
type AgeEncryptorInterface interface {
	Encryptor
	Decryptor
	HasRecipients() bool
	HasIdentities() bool
}

// LookupEnvAgePublicKey returns the age public key from the environment variable.
func LookupEnvAgePublicKey() (string, bool) {
	return os.LookupEnv(EnvAgePublicKey)
}

// Encrypted holds configuration for encrypting and decrypting resource fields.
// It supports both age and AES encryption, using the corresponding encryptor types.
type Encrypted struct {
	AesKey       string     `docs:"The AES key to use for field encryption"               json:"aesKey"       yaml:"aesKey"`
	AgePublicKey string     `docs:"The age public key for encryption (recipient)"         json:"agePublicKey" yaml:"agePublicKey"`
	AgeIdentity  string     `docs:"The age identity for decryption (string or file path)" json:"ageIdentity"  yaml:"ageIdentity"`
	KindFields   KindFields `docs:"The fields to encrypt for each kind"                   json:"kindFields"   yaml:"kindFields"`

	aes AesEncryptorInterface
	age AgeEncryptorInterface
}

// Setup initializes the encrypted config, resolving environment variables and
// creating the appropriate encryptors.
func (e *Encrypted) Setup() error {
	if e.AesKey == "" {
		if k, ok := os.LookupEnv(EnvAesKey); ok {
			e.AesKey = k
		}
	}
	if e.AgePublicKey == "" {
		if pk, ok := os.LookupEnv(EnvAgePublicKey); ok {
			e.AgePublicKey = pk
		}
	}

	// Validate if kind fields are configured but no key is explicitly set, fail early.
	if len(e.KindFields) > 0 && e.AesKey == "" && e.AgePublicKey == "" {
		return fmt.Errorf("encrypted mode needs a valid agePublicKey or aesKey."+
			" please remove the 'encrypted config' or provide the 'agePublicKey' via env variable %q or 'aesKey' via env variable %q",
			EnvAgePublicKey, EnvAesKey,
		)
	}

	if e.AgeIdentity == "" {
		if id, ok := os.LookupEnv(EnvAgeIdentity); ok {
			e.AgeIdentity = id
		}
	}

	// The identity might be a file path, so read it if necessary
	if e.AgeIdentity != "" {
		id, err := encryption.ReadAgeIdentity(e.AgeIdentity)
		if err != nil {
			return err
		}
		e.AgeIdentity = id
	}

	if e.AesKey != "" {
		aes, err := encryption.NewAesEncryptor(e.AesKey)
		if err != nil {
			return err
		}
		e.aes = aes
	}

	if e.AgePublicKey != "" || e.AgeIdentity != "" {
		age, err := encryption.NewAgeEncryptor(e.AgePublicKey, e.AgeIdentity)
		if err != nil {
			return err
		}
		e.age = age
	}

	return nil
}

// Algorithm returns the name of the encryption algorithm that will be used.
// It returns "age" when age is configured, "AES" when only AES is configured,
// and "age + AES" when both are configured.
func (e *Encrypted) Algorithm() string {
	hasAge := e.age != nil && e.age.HasRecipients()
	hasAES := e.aes != nil
	switch {
	case hasAge && hasAES:
		return "age + AES"
	case hasAge:
		return "age"
	case hasAES:
		return "AES"
	default:
		return ""
	}
}

// doEncrypt encrypts the given value using the configured encryption method.
// Age encryption is preferred over AES when both are configured.
func (e *Encrypted) doEncrypt(val any) string {
	strVal := fmt.Sprintf("%v", val)

	// Don't encrypt if already encrypted or empty
	if strings.HasPrefix(strVal, encryption.AesPrefix) ||
		strings.HasPrefix(strVal, encryption.AgePrefix) || strVal == "" {
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

// Encrypt encrypts secrets in exported resource files using AES encryption.
func Encrypt(printFlags *genericclioptions.PrintFlags, aesKey string, files ...string) error {
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

		table.Append(
			file, us.GetNamespace(), us.GetKind(), us.GetName(), config.Encrypted.Algorithm(), strconv.Itoa(encryptedCount))

		if err := utils.WriteFile(printFlags, file, us); err != nil {
			return err
		}
	}

	return table.Render()
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

		table.Append(
			file, us.GetNamespace(), us.GetKind(), us.GetName(), config.Encrypted.Algorithm(), strconv.Itoa(encryptedCount))

		if err := utils.WriteFile(printFlags, file, us); err != nil {
			return err
		}
	}

	return table.Render()
}

// Decrypt decrypts encrypted fields in exported resource files.
func Decrypt(printFlags *genericclioptions.PrintFlags, aesKey, ageIdentity string, files ...string) error {
	var aesEnc *encryption.AesEncryptor
	var ageEnc *encryption.AgeEncryptor

	if aesKey != "" {
		var err error
		aesEnc, err = encryption.NewAesEncryptor(aesKey)
		if err != nil {
			return err
		}
	}

	if ageIdentity != "" {
		var err error
		ageEnc, err = encryption.NewAgeEncryptor("", ageIdentity)
		if err != nil {
			return err
		}
	}

	if aesEnc == nil && (!ageEnc.HasIdentities()) {
		return errors.New("decrypt requires either an age identity or an AES key")
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

		algorithm := "none"
		switch {
		case res.hasAES && res.hasAge:
			algorithm = "AES + age"
		case res.hasAES:
			algorithm = "AES"
		case res.hasAge:
			algorithm = "age"
		}

		table.Append(
			file, us.GetNamespace(), us.GetKind(), us.GetName(), algorithm, strconv.Itoa(res.count))

		if err := utils.WriteFile(printFlags, file, us); err != nil {
			return err
		}
	}

	return table.Render()
}

// decryptResult holds the result of decrypting fields.
type decryptResult struct {
	count  int
	hasAES bool
	hasAge bool
}

// decryptFields recursively decrypts encrypted fields in the given object.
func decryptFields(
	obj map[string]any,
	aesEnc *encryption.AesEncryptor,
	ageEnc *encryption.AgeEncryptor,
) (decryptResult, error) {
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
				if after, ok := strings.CutPrefix(v, encryption.AesPrefix); ok {
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
				if after, ok := strings.CutPrefix(v, encryption.AgePrefix); ok {
					plaintext, err := ageEnc.Decrypt(after)
					if err != nil {
						return res, err
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
			if strings.HasPrefix(e, encryption.AesPrefix) || strings.HasPrefix(e, encryption.AgePrefix) {
				count++
			}
		}
	}
	return count
}
