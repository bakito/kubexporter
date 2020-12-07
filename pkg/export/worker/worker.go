package worker

import (
	"context"
	"fmt"
	"github.com/bakito/kubexporter/pkg/log"
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Worker interface
type Worker interface {
	GenerateWork(s *sync.WaitGroup, out chan *types.GroupResource) func(resource *types.GroupResource)
	Stop() int
}

type worker struct {
	id               int
	config           *types.Config
	mainBar          *mpb.Bar
	recBar           *mpb.Bar
	currentKind      string
	elapsedDecorator decor.Decorator
	client           dynamic.Interface
	mapper           meta.RESTMapper
	errors           int
	queryFinished    bool
}

func New(id int, config *types.Config, mapper meta.RESTMapper, client dynamic.Interface, prog *mpb.Progress, mainBar *mpb.Bar) Worker {
	w := &worker{
		id:               id + 1,
		mainBar:          mainBar,
		config:           config,
		mapper:           mapper,
		client:           client,
		elapsedDecorator: decor.NewElapsed(decor.ET_STYLE_GO, time.Now()),
	}

	if prog != nil {
		w.recBar = prog.AddBar(1,
			mpb.PrependDecorators(
				w.preDecorator(),
			),
			mpb.AppendDecorators(
				w.postDecorator(),
			),
		)
	}
	return w
}

// Stop end worker
func (w *worker) Stop() int {
	if w.recBar != nil {
		w.recBar.SetTotal(100, true)
	}
	return w.errors
}

// list resources
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

func (w *worker) preDecorator() decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		if s.Completed {
			return fmt.Sprintf("👷 %2d:", w.id)
		}
		if w.queryFinished && s.Total == 0 {
			return fmt.Sprintf("\U0001F971 %2d: idle", w.id)
		}
		if !w.queryFinished {
			return fmt.Sprintf("🔍 %2d: %s", w.id, w.currentKind)

		}
		return fmt.Sprintf("👷 %2d: %s %s", w.id, w.currentKind, w.elapsedDecorator.Decor(s))
	})
}

func (w *worker) postDecorator() decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		if s.Completed {
			return log.Check
		}
		return fmt.Sprintf("%s / %s %s",
			decor.CurrentNoUnit("").Decor(s),
			decor.TotalNoUnit("").Decor(s),
			decor.Percentage().Decor(s))
	})
}

// GenerateWork generate the work function
func (w *worker) GenerateWork(wg *sync.WaitGroup, out chan *types.GroupResource) func(resource *types.GroupResource) {

	return func(res *types.GroupResource) {
		defer wg.Done()
		w.queryFinished = false
		ctx := context.TODO()
		w.currentKind = res.GroupKind()
		w.elapsedDecorator = decor.NewElapsed(decor.ET_STYLE_GO, time.Now())

		if w.recBar != nil {
			w.recBar.SetCurrent(0)
			w.recBar.SetTotal(0, false)
		}
		start := time.Now()
		ul, err := w.list(ctx, res.APIGroup, res.APIVersion, res.APIResource.Kind)

		res.QueryDuration = time.Now().Sub(start)
		w.queryFinished = true
		start = time.Now()

		if err != nil {
			w.errors++
			if errors.IsNotFound(err) {
				res.Error = "Not Found"
			} else if errors.IsMethodNotSupported(err) {
				res.Error = "Not Allowed"
			} else {
				res.Error = "Error:" + err.Error()
			}
		} else {
			if w.recBar != nil {
				w.recBar.SetTotal(int64(len(ul.Items)), false)
			}
			if w.config.AsLists {
				w.exportLists(res, ul)
			} else {
				w.exportSingleResources(res, ul)
			}
		}
		if ul != nil {
			res.Instances = len(ul.Items)
		}
		res.ExportDuration = time.Now().Sub(start)

		if w.mainBar != nil {
			w.mainBar.Increment()
		}
		out <- res
	}
}

func (w *worker) exportLists(res *types.GroupResource, ul *unstructured.UnstructuredList) {
	if res == nil || ul == nil {
		return
	}
	clone := ul.DeepCopy()
	clone.Items = nil
	unstructured.RemoveNestedField(clone.Object, "metadata")

	perNs := make(map[string]*unstructured.UnstructuredList)
	for _, u := range ul.Items {
		w.config.FilterFields(res, u)

		if _, ok := perNs[u.GetNamespace()]; !ok {
			ul := &unstructured.UnstructuredList{}
			clone.DeepCopyInto(ul)
			perNs[u.GetNamespace()] = ul
		}
		perNs[u.GetNamespace()].Items = append(perNs[u.GetNamespace()].Items, u)
	}

	for ns, usl := range perNs {
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
		closeIgnoreError(f)

		err = w.config.PrintObj(usl, f)
		if err != nil {
			res.Error = err.Error()
			continue
		}

		if w.recBar != nil {
			w.recBar.IncrBy(len(usl.Items))
		}
	}
}

func (w *worker) exportSingleResources(res *types.GroupResource, ul *unstructured.UnstructuredList) {
	if res == nil || ul == nil {
		return
	}
	for _, u := range ul.Items {
		w.config.FilterFields(res, u)
		us := &u

		filename, err := w.config.FileName(res, us)
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
		closeIgnoreError(f)

		err = w.config.PrintObj(us, f)
		if err != nil {
			res.Error = err.Error()
			continue
		}

		if w.recBar != nil {
			w.recBar.Increment()
		}
	}
}

func closeIgnoreError(f io.Closer) func() {
	return func() {
		_ = f.Close()
	}
}
