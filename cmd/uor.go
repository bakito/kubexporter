package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// updateOwnerReferences
var updateOwnerReferences = &cobra.Command{
	Use:     "update-owner-references",
	Aliases: []string{"uor"},
	Short:   "Update owner references of an export against the current cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig(cmd, configFlags, printFlags)
		if err != nil {
			return err
		}

		err = config.Validate()
		if err != nil {
			return err
		}

		_, err = config.RestConfig()
		if err != nil {
			return err
		}

		var files []string
		err = filepath.Walk(config.Target, func(path string, info os.FileInfo, err error) error {

			if err != nil {

				fmt.Println(err)
				return nil
			}

			if !info.IsDir() && filepath.Ext(path) == ".yaml" {
				files = append(files, path)
			}

			return nil
		})

		return err

	},
}

func init() {
	rootCmd.AddCommand(updateOwnerReferences)
	configFlags.AddFlags(updateOwnerReferences.Flags())
	updateOwnerReferences.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}
