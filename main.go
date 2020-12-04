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
)

const (
	defaultFileNamePattern     = `{{default "_cluster_" .Namespace}}/{{if .Group}}{{printf "%s." .Group }}{{end}}{{.Kind}}.{{.Name}}.{{.Extension}}`
	defaultListFileNamePattern = `{{default "_cluster_" .Namespace}}/{{if .Group}}{{printf "%s." .Group }}{{end}}{{.Kind}}.{{.Extension}}`
	defaultFormat              = "yaml"
	defaultTarget              = "exports"
	na                         = "N/A"
)

func main() {
	var configFile string
	var worker int
	var namespace string
	var outputFormat string
	var ver bool
	var rmTarget bool
	flag.StringVar(&configFile, "config", "", "config file")
	flag.StringVar(&namespace, "namespace", na, "set the workspace")
	flag.StringVar(&outputFormat, "output-format", na, "set the output format (yaml / json)")
	flag.IntVar(&worker, "worker", -1, "set the number of workers")
	flag.BoolVar(&ver, "version", false, "get the version")
	flag.BoolVar(&rmTarget, "rm-target", false, "delete the target dir before executing")
	flag.Parse()
	silenceKlog()

	if ver {
		fmt.Printf("kubexporter version: %s\n", version.Version)
		return
	}

	conf := &types.Config{
		FileNameTemplate:     defaultFileNamePattern,
		ListFileNameTemplate: defaultListFileNamePattern,
		OutputFormat:         defaultFormat,
		Target:               defaultTarget,
		Summary:              false,
		Progress:             true,
		Worker:               1,
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
	if rmTarget {
		conf.ClearTarget = rmTarget
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
