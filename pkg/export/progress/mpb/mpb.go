package mpb

import (
	"fmt"
	"time"

	"github.com/bakito/kubexporter/pkg/export/progress"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func NewProgress(resources int) progress.Progress {
	prog := mpb.New()
	return &mpbProgress{
		prog:             prog,
		elapsedDecorator: decor.NewElapsed(decor.ET_STYLE_GO, time.Now()),
		mainBar:          newMpbMainBar(prog, resources),
	}
}

type mpbProgress struct {
	id               int
	workers          int
	prog             *mpb.Progress
	elapsedDecorator decor.Decorator
	mainBar          *mpb.Bar
	resourceBar      *mpb.Bar
}

func (m *mpbProgress) Async() bool {
	return true
}

func (m *mpbProgress) Run() error {
	m.prog.Wait()
	return nil
}

func (m *mpbProgress) NewWorker() progress.Progress {
	m.workers++
	return &mpbProgress{
		prog:             m.prog,
		id:               m.workers,
		elapsedDecorator: decor.NewElapsed(decor.ET_STYLE_GO, time.Now()),
		mainBar:          m.mainBar,
	}
}

func (m *mpbProgress) Reset() {
	m.elapsedDecorator = decor.NewElapsed(decor.ET_STYLE_GO, time.Now())
}

func newMpbMainBar(prog *mpb.Progress, size int) *mpb.Bar {
	bar := prog.AddBar(int64(size),
		mpb.PrependDecorators(
			// display our name with one space on the right
			decor.Name("Resources", decor.WC{W: len("Resources") + 1, C: decor.DindentRight}),
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
	return bar
}

func (m *mpbProgress) NewSearchBar(step progress.Step) {
	newBar := m.prog.AddBar(1,
		mpb.PrependDecorators(
			m.preDecoratorSearch(step.CurrentKind, step.PageSize, step.CurrentPage),
		),
		mpb.AppendDecorators(
			m.postDecorator(),
		),
		mpb.BarQueueAfter(m.resourceBar),
	)
	m.resourceBar = newBar
}

func (m *mpbProgress) NewExportBar(step progress.Step) {
	if m.resourceBar != nil && step.Total > 0 {
		newBar := m.prog.AddBar(int64(step.Total),
			mpb.PrependDecorators(
				m.preDecoratorExport(step.CurrentKind, step.PageSize, step.CurrentPage),
			),
			mpb.AppendDecorators(
				m.postDecorator(),
			),
			mpb.BarQueueAfter(m.resourceBar),
		)
		m.resourceBar = newBar
	}
}

func (m *mpbProgress) preDecoratorSearch(currentKind string, pageSize int, currentPage int) decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		page := ""
		if pageSize > 0 {
			page = fmt.Sprintf(" (page %d)", currentPage)
		}
		return fmt.Sprintf("🔍 %2d: %s%s ", m.id, currentKind, page)
	})
}

func (m *mpbProgress) preDecoratorExport(currentKind string, pageSize int, currentPage int) decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		page := ""
		if pageSize > 0 {
			page = fmt.Sprintf(" (page %d)", currentPage)
		}
		d, _ := m.elapsedDecorator.Decor(s)
		return fmt.Sprintf("👷 %2d: %s%s %s", m.id, currentKind, page, d)
	})
}

func (m *mpbProgress) postDecorator() decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		d1, _ := decor.CurrentNoUnit("").Decor(s)
		d2, _ := decor.TotalNoUnit("").Decor(s)
		d3, _ := decor.Percentage().Decor(s)
		return fmt.Sprintf("%s / %s %s", d1, d2, d3)
	})
}

func (m *mpbProgress) IncrementMainBar() {
	m.mainBar.Increment()
}

func (m *mpbProgress) IncrementResourceBarBy(_ int, inc int) {
	if m.resourceBar != nil {
		m.resourceBar.IncrBy(inc)
	}
}
