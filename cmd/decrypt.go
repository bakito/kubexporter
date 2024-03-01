package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bakito/kubexporter/pkg/types"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/ptr"
)

// decrypt
var (
	aesKey string

	decrypt = &cobra.Command{
		Use:   "decrypt <file-path(s)>",
		Short: "Decrypt secrets in exported resource files",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
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

func readKey() (string, error) {
	// restore terminal state on interrupt https://github.com/golang/go/issues/31180
	oldState, err := term.GetState(syscall.Stdin)
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(syscall.Stdin, oldState) }()

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	go func() {
		for range sigch {
			_ = term.Restore(syscall.Stdin, oldState)
			os.Exit(0)
		}
	}()

	fmt.Println("Please the aes key: ")
	key, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	return string(key), nil
}

func init() {
	rootCmd.AddCommand(decrypt)
	decrypt.PersistentFlags().StringVar(&aesKey, "aes-key", "", "the decryption key")
}
