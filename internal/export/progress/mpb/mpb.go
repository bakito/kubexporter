package mpb

import (
	"fmt"
	"sync"
	"time"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"github.com/bakito/kubexporter/internal/export/progress"
)

func NewProgress(resources int) progress.Progress {
	return newMpbProgress(mpb.New(), resources)
}

func newMpbProgress(prog *mpb.Progress, resources int) *mpbProgress {
	main := &mpbProgress{
		prog:             prog,
		elapsedDecorator: decor.NewElapsed(decor.ET_STYLE_GO, time.Now()),
		shared: &shared{
			mainBar:   newMpbMainBar(prog, resources),
			mainTotal: int64(resources),
		},
	}
	return main
}

// shared holds the state shared between the main progress and all its workers.
type shared struct {
	mx          sync.Mutex
	mainBar     *mpb.Bar
	mainTotal   int64
	mainCurrent int64
	workers     []*mpbProgress
}

type mpbProgress struct {
	id               int
	prog             *mpb.Progress
	elapsedDecorator decor.Decorator
	shared           *shared

	resourceBar     *mpb.Bar
	resourceTotal   int64
	resourceCurrent int64
}

func (*mpbProgress) Async() bool {
	return true
}

func (m *mpbProgress) Run() error {
	m.prog.Wait()
	return nil
}

func (m *mpbProgress) NewWorker() progress.Progress {
	return m.addWorker()
}

func (m *mpbProgress) addWorker() *mpbProgress {
	m.shared.mx.Lock()
	defer m.shared.mx.Unlock()
	w := &mpbProgress{
		prog:             m.prog,
		id:               len(m.shared.workers) + 1,
		elapsedDecorator: decor.NewElapsed(decor.ET_STYLE_GO, time.Now()),
		shared:           m.shared,
	}
	m.shared.workers = append(m.shared.workers, w)
	return w
}

func (m *mpbProgress) Reset() {
	m.elapsedDecorator = decor.NewElapsed(decor.ET_STYLE_GO, time.Now())
}

// Finish completes all bars, making sure they all end up at 100%.
func (m *mpbProgress) Finish() {
	m.shared.mx.Lock()
	defer m.shared.mx.Unlock()

	for _, w := range m.shared.workers {
		w.completeResourceBar()
	}
	m.completeResourceBar()

	if inc := m.shared.mainTotal - m.shared.mainCurrent; inc > 0 {
		m.shared.mainBar.IncrInt64(inc)
		m.shared.mainCurrent = m.shared.mainTotal
	}
}

// completeResourceBar fills the current resource bar up to its total.
func (m *mpbProgress) completeResourceBar() {
	if m.resourceBar == nil {
		return
	}
	if inc := m.resourceTotal - m.resourceCurrent; inc > 0 {
		m.resourceBar.IncrInt64(inc)
	}
	m.resourceCurrent = m.resourceTotal
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
	m.shared.mx.Lock()
	defer m.shared.mx.Unlock()

	// make sure the previous bar is completed before it is replaced
	m.completeResourceBar()

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
	m.resourceTotal = 1
	m.resourceCurrent = 0
}

func (m *mpbProgress) NewExportBar(step progress.Step) {
	m.shared.mx.Lock()
	defer m.shared.mx.Unlock()

	if m.resourceBar == nil || step.Total <= 0 {
		// nothing to export, complete the current (search) bar
		m.completeResourceBar()
		return
	}

	// make sure the previous bar is completed before it is replaced
	m.completeResourceBar()

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
	m.resourceTotal = int64(step.Total)
	m.resourceCurrent = 0
}

func (m *mpbProgress) preDecoratorSearch(currentKind string, pageSize, currentPage int) decor.Decorator {
	return decor.Any(func(decor.Statistics) string {
		page := ""
		if pageSize > 0 {
			page = fmt.Sprintf(" (page %d)", currentPage)
		}
		return fmt.Sprintf("🔍 %2d: %s%s ", m.id, currentKind, page)
	})
}

func (m *mpbProgress) preDecoratorExport(currentKind string, pageSize, currentPage int) decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		page := ""
		if pageSize > 0 {
			page = fmt.Sprintf(" (page %d)", currentPage)
		}
		d, _ := m.elapsedDecorator.Decor(s)
		return fmt.Sprintf("👷 %2d: %s%s %s", m.id, currentKind, page, d)
	})
}

func (*mpbProgress) postDecorator() decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		d1, _ := decor.CurrentNoUnit("").Decor(s)
		d2, _ := decor.TotalNoUnit("").Decor(s)
		d3, _ := decor.Percentage().Decor(s)
		return fmt.Sprintf("%s / %s %s", d1, d2, d3)
	})
}

func (m *mpbProgress) IncrementMainBar() {
	m.shared.mx.Lock()
	defer m.shared.mx.Unlock()

	if m.shared.mainCurrent >= m.shared.mainTotal {
		return
	}
	m.shared.mainCurrent++
	m.shared.mainBar.Increment()
}

func (m *mpbProgress) IncrementResourceBarBy(_, inc int) {
	m.shared.mx.Lock()
	defer m.shared.mx.Unlock()

	if m.resourceBar == nil || inc <= 0 {
		return
	}
	// never increment beyond the total
	if remaining := m.resourceTotal - m.resourceCurrent; int64(inc) > remaining {
		inc = int(remaining)
	}
	if inc <= 0 {
		return
	}
	m.resourceCurrent += int64(inc)
	m.resourceBar.IncrBy(inc)
}
