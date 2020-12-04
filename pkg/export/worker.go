package export

import (
	"context"
	"fmt"
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

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
}

// end worker
func (w *worker) stop() int {
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

func (w *worker) decorator() decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		return fmt.Sprintf("worker: %2d: %s %s", w.id, w.currentKind, w.elapsedDecorator.Decor(s))
	})
}
