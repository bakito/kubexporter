package worker

import (
	"fmt"

	"github.com/bakito/kubexporter/pkg/log"
	"github.com/fatih/color"
	"github.com/vbauerster/mpb/v8/decor"
)

func (w *worker) preDecoratorList() decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		page := ""
		if w.config.QueryPageSize > 0 {
			page = fmt.Sprintf(" (page %d)", w.currentPage)
		}
		return fmt.Sprintf("🔍 %2d: %s%s", w.id, w.currentKind, page)
	})
}

func (w *worker) preDecoratorExport() decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		if s.Completed {
			return fmt.Sprintf("👷 %2d:", w.id)
		}
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
		if s.Completed {
			return log.Check
		}
		d1, _ := decor.CurrentNoUnit("").Decor(s)
		d2, _ := decor.TotalNoUnit("").Decor(s)
		d3, _ := decor.Percentage().Decor(s)
		str := fmt.Sprintf("%s / %s %s", d1, d2, d3)

		return str
	})
}

func toMetaFunc(c *color.Color) func(string) string {
	return func(s string) string {
		return c.Sprint(s)
	}
}
