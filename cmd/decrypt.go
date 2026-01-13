package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/ptr"

	"github.com/bakito/kubexporter/pkg/types"
)

// decrypt.
var (
	aesKey string

	decrypt = &cobra.Command{
		Use:   "decrypt <file-path(s)>",
		Short: "Decrypt secrets in exported resource files",
		RunE: func(_ *cobra.Command, args []string) (err error) {
			if k, ok := os.LookupEnv(types.EnvAesKey); ok {
				aesKey = k
			}

			if aesKey == "" {
				aesKey, err = readKey()
				if err != nil {
					return err
				}
			}

			printFlags = &genericclioptions.PrintFlags{
				OutputFormat:       ptr.To(types.DefaultFormat),
				JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
			}

			return types.Decrypt(printFlags, aesKey, args...)
		},
	}
)

func init() {
	rootCmd.AddCommand(decrypt)
	decrypt.PersistentFlags().StringVar(&aesKey, "aes-key", "", "the decryption key")
}
