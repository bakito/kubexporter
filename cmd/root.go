package cmd

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/bakito/kubexporter/pkg/export"
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/bakito/kubexporter/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

var (
	cfgFile     string
	configFlags *genericclioptions.ConfigFlags
	printFlags  *genericclioptions.PrintFlags
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "kubexporter",
	Version: version.Version,
	Short:   "easily export kubernetes resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig(cmd, configFlags, printFlags)
		if err != nil {
			return err
		}

		ex, err := export.NewExporter(config)
		if err != nil {
			return err
		}

		return ex.Export()
	},
}

func readConfig(cmd *cobra.Command, configFlags *genericclioptions.ConfigFlags, printFlags *genericclioptions.PrintFlags) (*types.Config, error) {
	config := types.NewConfig(configFlags, printFlags)

	if cfgFile != "" {
		if err := types.UpdateFrom(config, cfgFile); err != nil {
			return nil, err
		}
	}

	cmd.Flags().Visit(func(f *pflag.Flag) {
		switch f.Name {
		case "namespace":
			config.Namespace = f.Value.String()
		case "target":
			config.Target = f.Value.String()
		case "worker":
			i, _ := cmd.Flags().GetInt(f.Name)
			config.Worker = i
		case "clear-target":
			b, _ := cmd.Flags().GetBool(f.Name)
			config.ClearTarget = b
		case "quiet":
			b, _ := cmd.Flags().GetBool(f.Name)
			config.Quiet = b
		case "verbose":
			b, _ := cmd.Flags().GetBool(f.Name)
			config.Verbose = b
		case "summary":
			b, _ := cmd.Flags().GetBool(f.Name)
			config.Summary = b
		case "progress":
			config.Progress = types.Progress(f.Value.String())
		case "lists":
			b, _ := cmd.Flags().GetBool(f.Name)
			config.AsLists = b
		case "include-kinds":
			sl, _ := cmd.Flags().GetStringSlice(f.Name)
			config.Included.Kinds = sl
		case "exclude-kinds":
			sl, _ := cmd.Flags().GetStringSlice(f.Name)
			config.Excluded.Kinds = sl
		}
	})

	if config.Masked.Replacement == "" {
		config.Masked.Replacement = types.DefaultMaskReplacement
	}

	return config, nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	rootCmd.Flags().StringP("target", "t", "", "Set the target directory (default exports)")
	rootCmd.Flags().IntP("worker", "w", 1, "The number of worker to use for the export")
	rootCmd.Flags().BoolP("clear-target", "c", false, "If enabled, the target dir is deleted before running the new export")
	rootCmd.Flags().BoolP("quiet", "q", false, "If enabled, output is prevented")
	rootCmd.Flags().BoolP("verbose", "v", false, "If enabled, errors during export are listed in summary")
	rootCmd.Flags().Bool("summary", false, "If enabled, a summary is printed")
	rootCmd.Flags().StringP("progress", "p", string(types.ProgressBar), "Progress mode bar|simple|none (default bar) ")
	rootCmd.Flags().BoolP("lists", "l", false, "If enabled, all resources are exported as lists instead of individual files")
	rootCmd.Flags().StringSliceP("include-kinds", "i", []string{}, "Export only included kinds, if included kinds are defined, excluded will be ignored")
	rootCmd.Flags().StringSliceP("exclude-kinds", "e", []string{}, "Do not export excluded kinds")

	configFlags = genericclioptions.NewConfigFlags(true)
	configFlags.AddFlags(rootCmd.Flags())

	printFlags = &genericclioptions.PrintFlags{
		OutputFormat:       pointer.StringPtr(types.DefaultFormat),
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
	}
	printFlags.AddFlags(rootCmd)

	// silence klog log output
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(fs)
	_ = fs.Parse([]string{"-logtostderr=false"})
	klog.SetOutput(ioutil.Discard)
}
