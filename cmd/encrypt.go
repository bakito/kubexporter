package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/bakito/kubexporter/internal/secret"
	"github.com/bakito/kubexporter/internal/types"
)

// encrypt.
var (
	agePublicKeySecretNamespace string
	agePublicKeySecretName      string
	agePublicKeySecretKey       string

	encrypt = &cobra.Command{
		Use:   "encrypt <file-path(s)>",
		Short: "Encrypt secrets in exported resource files",
		RunE: func(cmd *cobra.Command, args []string) error {
			printFlags = &genericclioptions.PrintFlags{
				OutputFormat:       new(types.DefaultFormat),
				JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
			}

			// Use age encryption if age public key is provided
			if agePublicKey != "" {
				pk := agePublicKey
				if k, ok := os.LookupEnv(types.EnvAgePublicKey); ok {
					pk = k
				}
				if agePublicKeySecretNamespace != "" && agePublicKeySecretName != "" && agePublicKeySecretKey != "" {
					config, err := readConfig(cmd, configFlags, printFlags)
					if err != nil {
						return err
					}
					pk, err = secret.ReadKey(
						cmd.Context(),
						config,
						agePublicKeySecretNamespace,
						agePublicKeySecretName,
						agePublicKeySecretKey,
					)
					if err != nil {
						return err
					}
				}
				return types.EncryptWithAge(printFlags, pk, args...)
			}

			// Use AES encryption
			key, err := evaluateAesKey(cmd)
			if err != nil {
				return err
			}

			return types.Encrypt(printFlags, key, args...)
		},
	}
)

func init() {
	rootCmd.AddCommand(encrypt)
	aesEncryptKeyFlags(encrypt, "encryption")
	ageEncryptKeyFlags(encrypt, "encryption")
}

func aesEncryptKeyFlags(cmd *cobra.Command, mode string) {
	cmd.PersistentFlags().StringVar(&aesKey, "aes-key", "", fmt.Sprintf("the %s key", mode))
	cmd.PersistentFlags().
		StringVar(&aesKeySecretNamespace, "aes-key-secret-namespace", "", fmt.Sprintf("the namespace of the %s key secret", mode))
	cmd.PersistentFlags().
		StringVar(&aesKeySecretName, "aes-key-secret-name", "", fmt.Sprintf("the name of the %s key secret", mode))
	cmd.PersistentFlags().
		StringVar(&aesKeySecretKey, "aes-key-secret-key", "", fmt.Sprintf("the key of the %s key secret", mode))
}

func ageEncryptKeyFlags(cmd *cobra.Command, mode string) {
	cmd.PersistentFlags().StringVar(&agePublicKey, "age-public-key", "", "the age public key for "+mode)
	cmd.PersistentFlags().
		StringVar(&agePublicKeySecretNamespace, "age-public-key-secret-namespace", "", "the namespace of the age public key secret")
	cmd.PersistentFlags().
		StringVar(&agePublicKeySecretName, "age-public-key-secret-name", "", "the name of the age public key secret")
	cmd.PersistentFlags().
		StringVar(&agePublicKeySecretKey, "age-public-key-secret-key", "", "the key of the age public key secret")
}
