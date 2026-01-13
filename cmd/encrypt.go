package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/ptr"

	"github.com/bakito/kubexporter/pkg/types"
)

// encrypt.
var (
	encryptAesKey string

	encrypt = &cobra.Command{
		Use:   "encrypt <file-path(s)>",
		Short: "Encrypt secrets in exported resource files",
		RunE: func(_ *cobra.Command, args []string) (err error) {
			if k, ok := os.LookupEnv(types.EnvAesKey); ok {
				encryptAesKey = k
			}

			if encryptAesKey == "" {
				encryptAesKey, err = readKey()
				if err != nil {
					return err
				}
			}

			printFlags = &genericclioptions.PrintFlags{
				OutputFormat:       ptr.To(types.DefaultFormat),
				JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
			}

			return types.Encrypt(printFlags, encryptAesKey, args...)
		},
	}
)

func init() {
	rootCmd.AddCommand(encrypt)
	encrypt.PersistentFlags().StringVar(&encryptAesKey, "aes-key", "", "the encryption key")
}
