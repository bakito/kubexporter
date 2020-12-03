package main

import (
	"flag"
	"github.com/bakito/kubexporter/pkg/export"
	"github.com/bakito/kubexporter/pkg/types"
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
)

func main() {
	start := time.Now()
	defer func() { println(time.Now().Sub(start).String()) }()
	var configFile string
	flag.StringVar(&configFile, "config", "", "config file")
	flag.Parse()
	silenceKlog()

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
