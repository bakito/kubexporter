package cmd

import (
	"github.com/spf13/cobra"

	"github.com/bakito/kubexporter/pkg/uor"
)

// updateOwnerReferences.
var updateOwnerReferences = &cobra.Command{
	Use:     "update-owner-references",
	Aliases: []string{"uor"},
	Short:   "Update owner references of an export against the current cluster",
	RunE: func(cmd *cobra.Command, _ []string) error {
		config, err := readConfig(cmd, configFlags, printFlags)
		if err != nil {
			return err
		}

		return uor.Update(config)
	},
}

func init() {
	rootCmd.AddCommand(updateOwnerReferences)
	configFlags.AddFlags(updateOwnerReferences.Flags())
	printFlags.AddFlags(updateOwnerReferences)
	updateOwnerReferences.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}
