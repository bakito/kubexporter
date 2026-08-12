package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/bakito/kubexporter/internal/types"
)

// encrypt.
var (
	encrypt = &cobra.Command{
		Use:   "encrypt <file-path(s)>",
		Short: "Encrypt secrets in exported resource files",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := evaluateAesKey(cmd)
			if err != nil {
				return err
			}

			printFlags = &genericclioptions.PrintFlags{
				OutputFormat:       new(types.DefaultFormat),
				JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
			}

			return types.Encrypt(printFlags, key, args...)
		},
	}
)

func init() {
	rootCmd.AddCommand(encrypt)
	aesKeyFlags(encrypt, "encryption")
}
