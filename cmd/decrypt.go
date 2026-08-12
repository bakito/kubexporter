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

	decrypt = &cobra.Command{
		Use:   "decrypt <file-path(s)>",
		Short: "Decrypt secrets in exported resource files",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := evaluateAesKey(cmd)
			if err != nil {
				return err
			}

			printFlags = &genericclioptions.PrintFlags{
				OutputFormat:       new(types.DefaultFormat),
				JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
			}

			return types.Decrypt(printFlags, key, args...)
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
		key, err = readKey()
		if err != nil {
			return "", err
		}
	}
	return key, nil
}

func init() {
	rootCmd.AddCommand(decrypt)
	aesKeyFlags(decrypt, "decryption")
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
