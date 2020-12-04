package main

import (
	"flag"
	"fmt"
	"github.com/bakito/kubexporter/pkg/export"
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/bakito/kubexporter/version"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"k8s.io/klog/v2"
	"os"
	"time"
)

const (
	defaultFileNamePattern = `{{default "_cluster_" .Namespace}}/{{if .Group}}{{printf "%s." .Group }}{{end}}{{.Kind}}.{{.Name}}.{{.Extension}}`
	defaultFormat          = "yaml"
	defaultTarget          = "exports"
	na                     = "N/A"
)

func main() {
	var configFile string
	var worker int
	var namespace string
	var outputFormat string
	var ver bool
	flag.StringVar(&configFile, "config", "", "config file")
	flag.StringVar(&namespace, "namespace", na, "set the workspace")
	flag.StringVar(&outputFormat, "outputFormat", na, "set the output format (yaml / json)")
	flag.IntVar(&worker, "worker", -1, "set the number of workers")
	flag.BoolVar(&ver, "version", false, "get the version")
	flag.Parse()
	silenceKlog()

	if ver {
		fmt.Printf("kubexporter version: %s\n", version.Version)
		return
	}

	start := time.Now()
	defer func() { fmt.Printf("Total Duration: %s\n", time.Now().Sub(start).String()) }()

	conf := &types.Config{
		FileNameTemplate: defaultFileNamePattern,
		OutputFormat:     defaultFormat,
		Target:           defaultTarget,
		Summary:          false,
		Progress:         true,
		Worker:           1,
	}
	if configFile != "" {
		b, err := ioutil.ReadFile(configFile)
		if err != nil {
			panic(err)
		}
		err = yaml.Unmarshal(b, conf)
		if err != nil {
			panic(err)
		}
	}

	if namespace != na {
		conf.Namespace = namespace
	}
	if outputFormat != na {
		conf.OutputFormat = outputFormat
	}
	if worker > 0 {
		conf.Worker = worker
	}

	ex, err := export.NewExporter(conf)
	if err != nil {
		panic(err)
	}
	if ex.Export() != nil {
		panic(err)
	}

}

// silenceKlog configure klog to not write messages to stdErr
func silenceKlog() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(fs)
	_ = fs.Parse([]string{"-logtostderr=false"})
	klog.SetOutput(ioutil.Discard)
}
