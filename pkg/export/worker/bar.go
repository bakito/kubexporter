package worker

import (
	"fmt"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (w *worker) newSearchBar() {
	if w.prog != nil {
		prevBar := w.resourceBar
		w.resourceBar = w.prog.AddBar(1,
			mpb.PrependDecorators(
				w.preDecoratorSearch(),
			),
			mpb.AppendDecorators(
				w.postDecorator(),
			),
			mpb.BarQueueAfter(prevBar),
		)
	}
}

func (w *worker) newExportBar(ul *unstructured.UnstructuredList) {
	if w.resourceBar != nil && len(ul.Items) > 0 {
		prevBar := w.resourceBar
		w.resourceBar = w.prog.AddBar(int64(len(ul.Items)),
			mpb.PrependDecorators(
				w.preDecoratorExport(),
			),
			mpb.AppendDecorators(
				w.postDecorator(),
			),
			mpb.BarQueueAfter(prevBar),
		)
	}
}

func (w *worker) preDecoratorSearch() decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		page := ""
		if w.config.QueryPageSize > 0 {
			page = fmt.Sprintf(" (page %d)", w.currentPage)
		}
		return fmt.Sprintf("🔍 %2d: %s%s ", w.id, w.currentKind, page)
	})
}

func (w *worker) preDecoratorExport() decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		page := ""
		if w.config.QueryPageSize > 0 {
			page = fmt.Sprintf(" (page %d)", w.currentPage)
		}
		d, _ := w.elapsedDecorator.Decor(s)
		return fmt.Sprintf("👷 %2d: %s%s %s", w.id, w.currentKind, page, d)
	})
}

func (w *worker) postDecorator() decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		d1, _ := decor.CurrentNoUnit("").Decor(s)
		d2, _ := decor.TotalNoUnit("").Decor(s)
		d3, _ := decor.Percentage().Decor(s)
		return fmt.Sprintf("%s / %s %s", d1, d2, d3)
	})
}
