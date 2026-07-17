package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

	"github.com/bakito/kubexporter/pkg/export"
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/bakito/kubexporter/version"
)

var (
	cfgFile     string
	configFlags *genericclioptions.ConfigFlags
	printFlags  *genericclioptions.PrintFlags
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:     "kubexporter",
	Version: fmt.Sprintf("%s (rev: %s; date: %s)", version.Version, version.Revision, version.BuildDate),
	Short:   "easily export kubernetes resources",
	RunE: func(cmd *cobra.Command, _ []string) error {
		config, err := readConfig(cmd, configFlags, printFlags)
		if err != nil {
			return err
		}

		ex, err := export.NewExporter(config)
		if err != nil {
			return err
		}

		return ex.Export(context.TODO())
	},
}

func readConfig(
	cmd *cobra.Command,
	configFlags *genericclioptions.ConfigFlags,
	printFlags *genericclioptions.PrintFlags,
) (*types.Config, error) {
	config := types.NewConfig(configFlags, printFlags)

	if cfgFile != "" {
		if err := types.UpdateFrom(config, cfgFile); err != nil {
			return nil, err
		}
	}

	cmd.Flags().Visit(func(f *pflag.Flag) {
		switch f.Name {
		case "namespace":
			namespaces, _ := cmd.Flags().GetStringSlice(f.Name)
			config.Namespaces = namespaces
		case "include-cluster-resources":
			b, _ := cmd.Flags().GetBool(f.Name)
			config.IncludeClusterResources = b
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
		case "size":
			b, _ := cmd.Flags().GetBool(f.Name)
			config.PrintSize = b
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
		case "created-within":
			cw, _ := cmd.Flags().GetDuration(f.Name)
			config.CreatedWithin = cw
		case "archive":
			sl, _ := cmd.Flags().GetBool(f.Name)
			config.Archive = sl
		case "exclude-defaults":
			ed, _ := cmd.Flags().GetBool(f.Name)
			if ed && len(config.Excluded.Kinds) == 0 {
				config.Excluded.Kinds = types.DefaultExcludedKinds
			}
		case "otlp-metrics":
			ed, _ := cmd.Flags().GetBool(f.Name)
			if config.Metrics != nil {
				config.Metrics.OTLP.Enabled = ed
			}
		default:
		}
	})

	if err := config.Masked.Setup(); err != nil {
		return nil, err
	}

	if err := config.Encrypted.Setup(); err != nil {
		return nil, err
	}

	config.Encrypted.KindFields = config.Masked.KindFields.Diff(config.Encrypted.KindFields)

	correctProgressForNonTerminalRun(config)

	return config, nil
}

func correctProgressForNonTerminalRun(config *types.Config) {
	if config.Progress != types.ProgressNone &&
		config.Progress != types.ProgressSimple &&
		!isatty.IsTerminal(os.Stdout.Fd()) &&
		!isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		config.Progress = types.ProgressSimple
		config.Logger().Printf("Switching progress to %q in non terminal environment\n", config.Progress)
	}
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
	rootCmd.Flags().
		BoolP("exclude-defaults", "d", false, "If enabled, default excludes will be applied. ["+strings.Join(types.DefaultExcludedKinds, ", ")+"]")

	rootCmd.Flags().StringP(cflagP("target", "t", "exports"))
	rootCmd.Flags().IntP(cflagP("worker", "w", 1))
	rootCmd.Flags().BoolP(cflagP("clear-target", "c", false))
	rootCmd.Flags().BoolP(cflagP("quiet", "q", false))
	rootCmd.Flags().BoolP(cflagP("verbose", "v", false))
	rootCmd.Flags().Bool(cflag("summary", false))
	rootCmd.Flags().Bool(cflag("size", false))
	rootCmd.Flags().Bool(cflag("otlp-metrics", false))
	rootCmd.Flags().BoolP(cflagP("archive", "a", false))
	rootCmd.Flags().StringP(cflagP("progress", "p", string(types.ProgressBar)))
	rootCmd.Flags().BoolP(cflagP("lists", "l", false))
	rootCmd.Flags().StringSliceP(cflagP("include-kinds", "i", []string{}))
	rootCmd.Flags().StringSliceP(cflagP("exclude-kinds", "e", []string{}))
	rootCmd.Flags().Duration(cflag[time.Duration]("created-within", 0))
	rootCmd.Flags().StringSliceP(cflagP("namespace", "n", []string{}))
	rootCmd.Flags().Bool(cflag("include-cluster-resources", false))

	configFlags = genericclioptions.NewConfigFlags(true)
	configFlags.Namespace = nil
	configFlags.CacheDir = nil
	configFlags.AddFlags(rootCmd.Flags())

	printFlags = &genericclioptions.PrintFlags{
		OutputFormat:       new(types.DefaultFormat),
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
	}
	printFlags.AddFlags(rootCmd)

	// silence klog log output
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(fs)
	_ = fs.Parse([]string{"-logtostderr=false"})
	klog.SetOutput(io.Discard)
}
