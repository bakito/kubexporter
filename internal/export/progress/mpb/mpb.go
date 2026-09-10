package mpb

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"github.com/bakito/kubexporter/internal/export/progress"
	"github.com/bakito/kubexporter/internal/types"
)

const (
	mainBarTitle = "Resources"

	iconMain   = "📦"
	iconSearch = "🔍"
	iconExport = "👷"
)

var (
	styleFiller  = lipgloss.NewStyle().Foreground(lipgloss.Color("#316CE6"))
	stylePadding = lipgloss.NewStyle().Foreground(lipgloss.Color("#5F5F5F"))
	styleMain    = lipgloss.NewStyle().Bold(true)
	styleLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8A8A"))
	styleDetails = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8A8A"))
)

// NewProgress creates a new progress bar based progress.
func NewProgress(resources []*types.GroupResource) progress.Progress {
	labelWidth := runewidth.StringWidth(mainBarTitle)
	for _, res := range resources {
		labelWidth = max(labelWidth, runewidth.StringWidth(res.GroupKind()))
	}
	return newMpbProgress(mpb.New(), len(resources), labelWidth)
}

func newMpbProgress(prog *mpb.Progress, resources, labelWidth int) *mpbProgress {
	sh := &shared{
		mainTotal:  int64(resources),
		labelWidth: labelWidth,
	}
	sh.mainBar = newMpbMainBar(prog, resources, sh)
	return &mpbProgress{
		prog:             prog,
		elapsedDecorator: decor.NewElapsed(decor.ET_STYLE_GO, time.Now()),
		shared:           sh,
	}
}

// shared holds the state shared between the main progress and all its workers.
type shared struct {
	mx          sync.Mutex
	mainBar     *mpb.Bar
	mainTotal   int64
	mainCurrent int64
	labelWidth  int
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

// barStyle is the common style of all bars.
func barStyle() mpb.BarFillerBuilder {
	return mpb.BarStyle().
		Lbound("").Rbound("").
		Filler("━").FillerMeta(render(styleFiller)).
		Tip("━").TipMeta(render(styleFiller)).
		Padding("─").PaddingMeta(render(stylePadding))
}

func newMpbMainBar(prog *mpb.Progress, size int, sh *shared) *mpb.Bar {
	elapsed := decor.NewElapsed(decor.ET_STYLE_GO, time.Now())
	return prog.New(int64(size), barStyle(),
		mpb.PrependDecorators(
			elapsedDecorator(elapsed),
			labelDecorator(iconMain, mainBarTitle, sh, styleMain),
		),
		mpb.AppendDecorators(
			countersDecorator(),
			detailsDecorator(""),
		),
	)
}

func (m *mpbProgress) NewSearchBar(step progress.Step) {
	m.shared.mx.Lock()
	defer m.shared.mx.Unlock()

	// make sure the previous bar is completed before it is replaced
	m.completeResourceBar()

	newBar := m.prog.New(1, barStyle(),
		mpb.PrependDecorators(
			elapsedDecorator(nil),
			labelDecorator(iconSearch, step.CurrentKind, m.shared, styleLabel),
		),
		mpb.AppendDecorators(
			countersDecorator(),
			detailsDecorator(pageDetails(step)),
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

	newBar := m.prog.New(int64(step.Total), barStyle(),
		mpb.PrependDecorators(
			elapsedDecorator(m.elapsedDecorator),
			labelDecorator(iconExport, step.CurrentKind, m.shared, styleLabel),
		),
		mpb.AppendDecorators(
			countersDecorator(),
			detailsDecorator(pageDetails(step)),
		),
		mpb.BarQueueAfter(m.resourceBar),
	)
	m.resourceBar = newBar
	m.resourceTotal = int64(step.Total)
	m.resourceCurrent = 0
}

// elapsedDecorator renders the elapsed time in a fixed width column.
func elapsedDecorator(elapsed decor.Decorator) decor.Decorator {
	return decor.Meta(decor.Any(func(s decor.Statistics) string {
		el := ""
		if elapsed != nil {
			el, _ = elapsed.Decor(s)
		}
		return fmt.Sprintf("%7s ", strings.TrimSpace(el))
	}), render(styleDetails))
}

// labelDecorator renders the icon and the label in aligned columns.
func labelDecorator(icon, value string, sh *shared, style lipgloss.Style) decor.Decorator {
	return decor.Meta(decor.Any(func(decor.Statistics) string {
		// the icons are emoji occupying two cells
		return icon + " " + padTo(value, sh.labelWidth) + " "
	}), render(style))
}

// countersDecorator renders the percentage and the current / total values in fixed width
// columns, so all bars have the same width.
func countersDecorator() decor.Decorator {
	return decor.Meta(decor.Any(func(s decor.Statistics) string {
		cur, _ := decor.CurrentNoUnit("").Decor(s)
		total, _ := decor.TotalNoUnit("").Decor(s)
		percent, _ := decor.Percentage().Decor(s)
		values := fmt.Sprintf("%s/%s", strings.TrimSpace(cur), strings.TrimSpace(total))
		return fmt.Sprintf("  %5s  %11s", strings.TrimSpace(percent), values)
	}), render(styleDetails))
}

// detailsDecorator renders the trailing details in a fixed width column.
func detailsDecorator(details string) decor.Decorator {
	return decor.Meta(decor.Any(func(decor.Statistics) string {
		return fmt.Sprintf("  %-9s", details)
	}), render(styleDetails))
}

// pageDetails describes the current page of a step.
func pageDetails(step progress.Step) string {
	if step.PageSize > 0 && step.CurrentPage > 0 {
		return fmt.Sprintf("page %d", step.CurrentPage)
	}
	return ""
}

// render adapts a lipgloss style to the meta function signature of mpb.
func render(style lipgloss.Style) func(string) string {
	return func(s string) string {
		return style.Render(s)
	}
}

// padTo pads the given value to the given display width.
func padTo(value string, width int) string {
	return value + strings.Repeat(" ", max(width-runewidth.StringWidth(value), 0))
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
