package cmd

import (
	"fmt"
	"os"

	"github.com/bakito/kubexporter/pkg/types"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

			if aesKey == "" {
				fmt.Println("Please the aes key: ")
				key, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return err
				}
				aesKey = string(key)
			}

			return types.Decrypt(aesKey, resourceFiles...)
		},
	}
)

func init() {
	rootCmd.AddCommand(decrypt)
	decrypt.PersistentFlags().StringArrayVar(&resourceFiles, "file", nil, "resource files to decrypt")
	decrypt.PersistentFlags().StringVar(&aesKey, "aes-key", "", "the decryption key")
}
