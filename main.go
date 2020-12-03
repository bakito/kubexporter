package main

import (
	"archive/zip"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/olekukonko/tablewriter"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

const (
	defaultFileNamePattern = `{{default "_cluster_" .Namespace}}/{{if .Group}}{{printf "%s." .Group }}{{end}}{{.Kind}}.{{.Name}}.{{.Extension}}`
	defaultFormat          = "yaml"
	defaultTarget          = "exports"
)

func main() {

	var configFile string
	flag.StringVar(&configFile, "config", "", "config file")
	flag.Parse()
	silenceKlog()

	conf := &Config{
		FileNameTemplate: defaultFileNamePattern,
		OutputFormat:     defaultFormat,
		Target:           defaultTarget,
		Summary:          false,
		Progress:         true,
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
	err := conf.Validate()
	if err != nil {
		panic(err)
	}

	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	ctx := context.TODO()
	dcl, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		panic(err)
	}

	lists, err := dcl.ServerPreferredResources()
	if err != nil {
		panic(err)
	}

	var resources []groupResource

	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			r := groupResource{
				APIGroup:        gv.Group,
				APIVersion:      gv.Version,
				APIGroupVersion: gv.String(),
				APIResource:     resource,
			}
			if len(resource.Verbs) == 0 || conf.IsExcluded(r) || (!resource.Namespaced && conf.Namespace != "") {
				continue
			}

			resources = append(resources, r)
		}
	}

	sort.Slice(resources, func(i, j int) bool {
		ret := strings.Compare(resources[i].APIGroup, resources[j].APIGroup)
		if ret > 0 {
			return false
		} else if ret == 0 {
			return strings.Compare(resources[i].APIResource.Kind, resources[j].APIResource.Kind) < 0
		}
		return true
	})

	w := &worker{
		config: conf,
	}
	var prog *mpb.Progress
	var mainBar *mpb.Bar
	var recBar *mpb.Bar
	if conf.Progress {
		prog = mpb.New()
		mainBar = prog.AddBar(int64(len(resources)),
			mpb.PrependDecorators(
				// display our name with one space on the right
				decor.Name("Resources", decor.WC{W: len("Resources") + 1, C: decor.DidentRight}),
				decor.Elapsed(decor.ET_STYLE_GO),
			),
			mpb.AppendDecorators(
				decor.CurrentNoUnit(""),
				decor.Name("/"),
				decor.TotalNoUnit(""),
				decor.Name(" "),
				decor.Percentage(),
			),
		)

		recBar = prog.AddBar(int64(1),
			mpb.PrependDecorators(
				w.decorator(),
			),
			mpb.AppendDecorators(
				decor.CurrentNoUnit(""),
				decor.Name("/"),
				decor.TotalNoUnit(""),
				decor.Name(" "),
				decor.Percentage(),
			),
		)
	}

	w.mapper = restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dcl))
	w.client, err = dynamic.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	withError := false
	for i, res := range resources {
		w.startNewKind(res)

		if recBar != nil {
			recBar.SetCurrent(0)
			recBar.SetTotal(0, false)
		}
		start := time.Now()
		ul, err := w.list(ctx, res.APIGroup, res.APIVersion, res.APIResource.Kind)

		resources[i].QueryDuration = time.Now().Sub(start)
		start = time.Now()

		if err != nil {
			if errors.IsNotFound(err) {
				resources[i].Error = "Not Found"
				withError = true
			} else if errors.IsMethodNotSupported(err) {
				resources[i].Error = "Not Allowed"
				withError = true
			} else {
				resources[i].Error = "Error:" + err.Error()
				withError = true
			}
		} else {
			if recBar != nil {
				recBar.SetTotal(int64(len(ul.Items)), false)
			}
			for _, u := range ul.Items {
				conf.Excluded.FilterFields(res, u)

				us := &u
				b, err := conf.Marshal(us)
				if err != nil {
					panic(err)
				}
				filename, err := conf.FileName(res, us)
				if err != nil {
					panic(err)
				}

				_ = os.MkdirAll(filepath.Dir(filename), os.ModePerm)
				err = ioutil.WriteFile(filename, b, 0664)
				if err != nil {
					panic(err)
				}
				if recBar != nil {
					recBar.Increment()
				}
			}
		}
		if ul != nil {
			resources[i].Instances = len(ul.Items)
		}
		resources[i].ExportDuration = time.Now().Sub(start)

		if mainBar != nil {
			mainBar.Increment()
		}
	}
	if recBar != nil {
		recBar.SetTotal(100, true)
	}
	if prog != nil {
		prog.Wait()
	}

	if conf.Summary {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeaderLine(false)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetCenterSeparator("")
		table.SetColumnSeparator("")
		table.SetRowSeparator("")
		header := []string{"Group", "Version", "Kind", "Namespaces", "Instances", "Query Duration", "Export Duration"}
		if withError {
			header = append(header, "Error")
		}
		table.SetHeader(header)
		for _, r := range resources {
			table.Append(r.Report(withError))
		}
		table.Render()
	}

	if conf.Zip {
		if err := createZip(conf); err != nil {
			panic(err)
		}
	}
}

func createZip(conf *Config) error {
	// Get a Buffer to Write To
	var zipFile string
	if conf.Namespace != "" {
		zipFile = fmt.Sprintf("%s-%s-%s.zip", conf.Target, conf.Namespace, time.Now().Format("2006-01-02"))
	} else {
		zipFile = fmt.Sprintf("%s-%s.zip", conf.Target, time.Now().Format("2006-01-02"))
	}
	fmt.Printf("\ncreating zip archive %s\n", zipFile)

	outFile, err := os.Create(filepath.Join(conf.Target, zipFile))
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	// Create a new zip archive.
	w := zip.NewWriter(outFile)

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(info.Name()) != fmt.Sprintf(".%s", conf.OutputFormat) {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		f, err := w.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		return err
	}
	err = filepath.Walk(conf.Target, walker)
	if err != nil {
		return err
	}
	fmt.Printf("created archive %s\n", zipFile)
	return nil
}

// silenceKlog configure klog to not write messages to stdErr
func silenceKlog() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(fs)
	_ = fs.Parse([]string{"-logtostderr=false"})
	klog.SetOutput(ioutil.Discard)
}

func (w *worker) list(ctx context.Context, group, version, kind string) (*unstructured.UnstructuredList, error) {

	mapping, err := w.mapper.RESTMapping(schema.GroupKind{Group: group, Kind: kind}, version)
	if err != nil {
		return nil, err
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dr = w.client.Resource(mapping.Resource).Namespace(w.config.Namespace)
	} else {
		// for cluster-wide resources
		dr = w.client.Resource(mapping.Resource)
	}
	return dr.List(ctx, metav1.ListOptions{})
}

type worker struct {
	client           dynamic.Interface
	mapper           meta.RESTMapper
	currentKind      string
	elapsedDecorator decor.Decorator
	config           *Config
}

func (w *worker) startNewKind(gr groupResource) {
	w.currentKind = gr.GroupKind()
	w.elapsedDecorator = decor.NewElapsed(decor.ET_STYLE_GO, time.Now())
}

func (w *worker) decorator() decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		return fmt.Sprintf("%s %s", w.currentKind, w.elapsedDecorator.Decor(s))
	})
}
