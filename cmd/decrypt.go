package cmd

import (
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/spf13/cobra"
)

// decrypt
var (
	resourceFiles []string
	aesKey        string

	decrypt = &cobra.Command{
		Use:     "decrypt",
		Aliases: []string{"uor"},
		Short:   "Decrypt secrets in exported resource files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return types.Decrypt(aesKey, resourceFiles...)
		},
	}
)

func init() {
	rootCmd.AddCommand(decrypt)
	decrypt.PersistentFlags().StringArrayVar(&resourceFiles, "file", nil, "resource files to decrypt")
	decrypt.PersistentFlags().StringVar(&aesKey, "aes-key", "", "the decryption key")
}
