package cmd

import (
	"flag"
	"fmt"
	"github.com/bakito/kubexporter/pkg/export"
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/bakito/kubexporter/version"
	"github.com/spf13/cobra"
	"io/ioutil"
	"k8s.io/klog/v2"
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const (
	defaultFileNamePattern     = `{{default "_cluster_" .Namespace}}/{{if .Group}}{{printf "%s." .Group }}{{end}}{{.Kind}}.{{.Name}}.{{.Extension}}`
	defaultListFileNamePattern = `{{default "_cluster_" .Namespace}}/{{if .Group}}{{printf "%s." .Group }}{{end}}{{.Kind}}.{{.Extension}}`
	defaultFormat              = "yaml"
	defaultTarget              = "exports"
)

var (
	cfgFile string
	config  *types.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "kubexporter",
	Version: version.Version,
	Short:   "easily export kubernetes resources",
	RunE: func(cmd *cobra.Command, args []string) error {

		ex, err := export.NewExporter(config)
		if err != nil {
			panic(err)
		}
		if ex.Export() != nil {
			panic(err)
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kubexporter.yaml)")
	rootCmd.Flags().StringP("namespace", "n", "", "If present, the namespace scope for this export")
	rootCmd.Flags().StringP("output-format", "f", defaultFormat, "Set the output format [yaml(default), json]")
	rootCmd.Flags().IntP("worker", "w", 1, "The number of worker to use for the export")
	rootCmd.Flags().BoolP("clear-target", "c", false, "If enabled, the target dir is deleted before running the new export")
	rootCmd.Flags().BoolP("silent", "s", false, "If enabled, output is prevented")
	rootCmd.Flags().Bool("summary", false, "If enabled, a summary is printed")
	rootCmd.Flags().BoolP("as-lists", "l", false, "If enabled, all resources are exported as lists instead of individual files")

	_ = viper.BindPFlag("namespace", rootCmd.Flags().Lookup("namespace"))
	viper.SetDefault("namespace", "")
	_ = viper.BindPFlag("outputFormat", rootCmd.Flags().Lookup("output-format"))
	viper.SetDefault("outputFormat", defaultFormat)
	_ = viper.BindPFlag("worker", rootCmd.Flags().Lookup("worker"))
	viper.SetDefault("worker", 1)
	_ = viper.BindPFlag("clearTarget", rootCmd.Flags().Lookup("clear-target"))
	viper.SetDefault("clearTarget", false)
	_ = viper.BindPFlag("silent", rootCmd.Flags().Lookup("silent"))
	viper.SetDefault("silent", false)
	_ = viper.BindPFlag("summary", rootCmd.Flags().Lookup("summary"))
	viper.SetDefault("summary", false)
	_ = viper.BindPFlag("asList", rootCmd.Flags().Lookup("as-lists"))
	viper.SetDefault("asList", false)
	viper.SetDefault("progress", true)
	viper.SetDefault("fileNameTemplate", defaultFileNamePattern)
	viper.SetDefault("listFileNameTemplate", defaultListFileNamePattern)
	viper.SetDefault("target", defaultTarget)

	// silence klog log output
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(fs)
	_ = fs.Parse([]string{"-logtostderr=false"})
	klog.SetOutput(ioutil.Discard)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	if cfgFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".kubexporter" (without extension).
		v.AddConfigPath(home)
		v.SetConfigName(".kubexporter")
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("KUBEXPORTER")
	v.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	_ = v.ReadInConfig()

	config = &types.Config{}
	if err := viper.Unmarshal(config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
