package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/bakito/kubexporter/internal/secret"
	"github.com/bakito/kubexporter/internal/types"
)

// decrypt.
var (
	aesKey                string
	aesKeySecretNamespace string
	aesKeySecretName      string
	aesKeySecretKey       string

	agePublicKey               string
	ageIdentity                string
	ageIdentitySecretNamespace string
	ageIdentitySecretName      string
	ageIdentitySecretKey       string

	decrypt = &cobra.Command{
		Use:   "decrypt <file-path(s)>",
		Short: "Decrypt secrets in exported resource files",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Evaluate both AES and age keys to support mixed encrypted files
			key, err := evaluateAesKey(cmd)
			if err != nil {
				return err
			}

			ageID, err := evaluateAgeKey(cmd)
			if err != nil {
				return err
			}

			printFlags = &genericclioptions.PrintFlags{
				OutputFormat:       new(types.DefaultFormat),
				JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
			}

			return types.Decrypt(printFlags, key, ageID, args...)
		},
	}
)

func evaluateAesKey(cmd *cobra.Command) (key string, err error) {
	// 	use flag aes key value
	key = aesKey

	if k, ok := os.LookupEnv(types.EnvAesKey); ok {
		key = k
	}

	if aesKeySecretNamespace != "" && aesKeySecretName != "" && aesKeySecretKey != "" {
		config, err := readConfig(cmd, configFlags, printFlags)
		if err != nil {
			return "", err
		}
		key, err = secret.ReadKey(cmd.Context(), config, aesKeySecretNamespace, aesKeySecretName, aesKeySecretKey)
		if err != nil {
			return "", err
		}
	}

	if key == "" {
		// In non-TTY environments (e.g., CI), readKey fails with ioctl error.
		// If age identity is provided (directly or via K8s secret), skip the AES key prompt.
		if ageIdentity != "" || ageIdentitySecretNamespace != "" {
			return "", nil
		}
		key, err = readKey()
		if err != nil {
			return "", err
		}
	}
	return key, nil
}

func evaluateAgeKey(cmd *cobra.Command) (key string, err error) {
	key = ageIdentity

	if k, ok := os.LookupEnv(types.EnvAgeIdentity); ok {
		key = k
	}

	if ageIdentitySecretNamespace != "" && ageIdentitySecretName != "" && ageIdentitySecretKey != "" {
		config, err := readConfig(cmd, configFlags, printFlags)
		if err != nil {
			return "", err
		}
		key, err = secret.ReadKey(
			cmd.Context(),
			config,
			ageIdentitySecretNamespace,
			ageIdentitySecretName,
			ageIdentitySecretKey,
		)
		if err != nil {
			return "", err
		}
	}

	return key, nil
}

func init() {
	rootCmd.AddCommand(decrypt)
	aesKeyFlags(decrypt, "decryption")
	ageKeyFlags(decrypt, "decryption")
}

func aesKeyFlags(cmd *cobra.Command, mode string) {
	cmd.PersistentFlags().StringVar(&aesKey, "aes-key", "", fmt.Sprintf("the %s key", mode))
	cmd.PersistentFlags().
		StringVar(&aesKeySecretNamespace, "aes-key-secret-namespace", "", fmt.Sprintf("the namespace of the %s key secret", mode))
	cmd.PersistentFlags().
		StringVar(&aesKeySecretName, "aes-key-secret-name", "", fmt.Sprintf("the name of the %s key secret", mode))
	cmd.PersistentFlags().
		StringVar(&aesKeySecretKey, "aes-key-secret-key", "", fmt.Sprintf("the key of the %s key secret", mode))
}

func ageKeyFlags(cmd *cobra.Command, mode string) {
	cmd.PersistentFlags().StringVar(&agePublicKey, "age-public-key", "", "the age public key for "+mode)
	cmd.PersistentFlags().StringVar(&ageIdentity, "age-identity", "", "the age identity key for "+mode)
	cmd.PersistentFlags().
		StringVar(&ageIdentitySecretNamespace, "age-identity-secret-namespace", "", fmt.Sprintf("the namespace of the %s age identity secret", mode))
	cmd.PersistentFlags().
		StringVar(&ageIdentitySecretName, "age-identity-secret-name", "", fmt.Sprintf("the name of the %s age identity secret", mode))
	cmd.PersistentFlags().
		StringVar(&ageIdentitySecretKey, "age-identity-secret-key", "", fmt.Sprintf("the key of the %s age identity secret", mode))
}
