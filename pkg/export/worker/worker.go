package worker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bakito/kubexporter/pkg/client"
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/bakito/kubexporter/pkg/utils"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Worker interface
type Worker interface {
	GenerateWork(ctx context.Context, s *sync.WaitGroup, out chan *types.GroupResource) func(resource *types.GroupResource)
	Stop() Stats
}

type worker struct {
	id               int
	config           *types.Config
	mainBar          *mpb.Bar
	resourceBar      *mpb.Bar
	prog             *mpb.Progress
	currentKind      string
	currentPage      int
	elapsedDecorator decor.Decorator
	ac               *client.ApiClient
	queryFinished    bool
	stats            Stats
}

// Stats worker stats
type Stats struct {
	Errors     int
	namespaces map[string]bool
	Kinds      int
	Pages      int
	Resources  int
}

// Add stats
func (s *Stats) Add(o *Stats) {
	if o != nil {
		s.Kinds += o.Kinds
		s.Pages += o.Pages
		s.Resources += o.Resources
		s.Errors += o.Errors
		for ns := range o.namespaces {
			s.addNamespace(ns)
		}
	}
}

func (s *Stats) addNamespace(ns string) {
	if s.namespaces == nil {
		s.namespaces = make(map[string]bool)
	}
	s.namespaces[ns] = true
}

// Namespaces get the number of namespaces
func (s *Stats) Namespaces() int {
	return len(s.namespaces)
}

// HasErrors true if errors >0
func (s *Stats) HasErrors() bool {
	return s.Errors > 0
}

// New create a new worker
func New(id int, config *types.Config, ac *client.ApiClient, prog *mpb.Progress, mainBar *mpb.Bar) Worker {
	w := &worker{
		id:               id + 1,
		mainBar:          mainBar,
		config:           config,
		ac:               ac,
		elapsedDecorator: decor.NewElapsed(decor.ET_STYLE_GO, time.Now()),
		prog:             prog,
	}

	return w
}

// Stop end worker
func (w *worker) Stop() Stats {
	return w.stats
}

// list resources
func (w *worker) list(ctx context.Context, group, version, kind string, continueValue string) (*unstructured.UnstructuredList, error) {
	mapping, err := w.ac.Mapper.RESTMapping(schema.GroupKind{Group: group, Kind: kind}, version)
	if err != nil {
		return nil, err
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dr = w.ac.Client.Resource(mapping.Resource).Namespace(w.config.Namespace)
	} else {
		// for cluster-wide resources
		dr = w.ac.Client.Resource(mapping.Resource)
	}
	opts := metav1.ListOptions{Continue: continueValue}
	if !w.config.AsLists {
		// for lists, we do no pagination
		opts.Limit = int64(w.config.QueryPageSize)
	}
	return dr.List(ctx, opts)
}

// GenerateWork generate the work function
func (w *worker) GenerateWork(ctx context.Context, wg *sync.WaitGroup, out chan *types.GroupResource) func(resource *types.GroupResource) {
	return func(res *types.GroupResource) {
		defer wg.Done()
		w.stats.Kinds++
		w.queryFinished = false
		w.currentKind = res.GroupKind()
		w.elapsedDecorator = decor.NewElapsed(decor.ET_STYLE_GO, time.Now())

		hasMorePages := ""
		for {
			hasMorePages = w.listResources(ctx, res, hasMorePages)
			if hasMorePages == "" {
				break
			}
		}
		w.stats.Resources += res.ExportedInstances
		w.stats.Pages += res.Pages

		if w.config.Progress == types.ProgressSimple {
			w.config.Logger().Checkf("%s\n", res.GroupKind())
		}

		if w.mainBar != nil {
			w.mainBar.Increment()
		}
		out <- res
	}
}

func (w *worker) listResources(ctx context.Context, res *types.GroupResource, hasMorePages string) string {
	w.currentPage = res.Pages + 1
	w.newSearchBar()
	start := time.Now()
	ul, err := w.list(ctx, res.APIGroup, res.APIVersion, res.APIResource.Kind, hasMorePages)

	if w.resourceBar != nil {
		w.resourceBar.IncrBy(1)
	}

	res.QueryDuration += time.Since(start)
	w.queryFinished = true
	start = time.Now()

	if err != nil {
		w.stats.Errors++
		if errors.IsNotFound(err) {
			res.Error = "Not Found"
		} else if errors.IsMethodNotSupported(err) {
			res.Error = "Not Allowed"
		} else {
			res.Error = "Error:" + err.Error()
		}
	} else {
		w.newExportBar(ul)
		if w.config.AsLists {
			res.ExportedInstances += w.exportLists(res, ul)
		} else {
			res.ExportedInstances += w.exportSingleResources(res, ul)
		}
	}
	res.ExportDuration += time.Since(start)

	res.Instances += len(ul.Items)
	res.Pages++

	return ul.GetContinue()
}

func (w *worker) exportLists(res *types.GroupResource, ul *unstructured.UnstructuredList) int {
	if res == nil || ul == nil {
		return 0
	}
	clone := ul.DeepCopy()
	clone.Items = nil
	unstructured.RemoveNestedField(clone.Object, "metadata")

	perNs := make(map[string]*unstructured.UnstructuredList)
	for _, u := range ul.Items {
		if !w.config.IsInstanceExcluded(res, u) {
			w.config.FilterFields(res, u)
			w.config.MaskFields(res, u)
			w.config.EncryptFields(res, u)
			w.config.SortSliceFields(res, u)

			if _, ok := perNs[u.GetNamespace()]; !ok {
				ul := &unstructured.UnstructuredList{}
				clone.DeepCopyInto(ul)
				perNs[u.GetNamespace()] = ul
			}
			perNs[u.GetNamespace()].Items = append(perNs[u.GetNamespace()].Items, u)
		}
	}

	cnt := 0
	for ns, usl := range perNs {
		w.stats.addNamespace(ns)
		filename, err := w.config.ListFileName(res, ns)
		if err != nil {
			res.Error = err.Error()
			continue
		}

		filename = filepath.Join(w.config.Target, filename)
		_ = os.MkdirAll(filepath.Dir(filename), os.ModePerm)

		f, err := os.Create(filename)
		if err != nil {
			res.Error = err.Error()
			continue
		}

		err = utils.PrintObj(w.config.PrintFlags, usl, f)
		if err != nil {
			res.Error = err.Error()
			continue
		}
		closeIgnoreError(f)()

		if w.resourceBar != nil {
			w.resourceBar.IncrBy(len(usl.Items))
		}
		cnt += len(usl.Items)
	}
	return cnt
}

func (w *worker) exportSingleResources(res *types.GroupResource, ul *unstructured.UnstructuredList) int {
	if res == nil || ul == nil {
		return 0
	}
	names := make(map[string]int)
	cnt := 0
	for _, u := range ul.Items {
		if !w.config.IsInstanceExcluded(res, u) {
			cnt++
			w.stats.addNamespace(u.GetNamespace())
			w.config.FilterFields(res, u)
			w.config.MaskFields(res, u)
			w.config.EncryptFields(res, u)
			w.config.SortSliceFields(res, u)
			us := &u

			namespaceName := strings.ToLower(fmt.Sprintf("%s.%s", us.GetNamespace(), us.GetName()))
			nameCnt := names[namespaceName]

			filename, err := w.config.FileName(res, us, nameCnt)
			if err != nil {
				res.Error = err.Error()
				continue
			}

			names[namespaceName] = nameCnt + 1

			filename = filepath.Join(w.config.Target, filename)
			_ = os.MkdirAll(filepath.Dir(filename), os.ModePerm)

			f, err := os.Create(filename)
			if err != nil {
				res.Error = err.Error()
				continue
			}

			err = utils.PrintObj(w.config.PrintFlags, us, f)
			if err != nil {
				res.Error = err.Error()
				continue
			}
			closeIgnoreError(f)
		}

		if w.resourceBar != nil {
			w.resourceBar.Increment()
		}
	}
	return cnt
}

func closeIgnoreError(f io.Closer) func() {
	return func() {
		_ = f.Close()
	}
}
