package mpb

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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
	// the bars are refreshed manually, so the console is only redrawn if something
	// actually changed, which avoids flickering on slow consoles like the windows one
	ref := newRefresher(refreshRate(), idleRefreshRate)
	p := newMpbProgress(mpb.New(mpb.WithManualRefresh(ref.ch)), len(resources), labelWidth)
	p.shared.refresh = ref
	return p
}

// refreshRate is the shortest interval the bars are redrawn with. The windows console is
// considerably slower at redrawing than other terminals, redrawing it less often
// reduces flickering.
func refreshRate() time.Duration {
	if runtime.GOOS == "windows" {
		return 250 * time.Millisecond
	}
	return 120 * time.Millisecond
}

// idleRefreshRate is the interval the bars are redrawn with when nothing changed.
// It keeps the elapsed time columns running.
const idleRefreshRate = time.Second

// refresher triggers the render cycles of the progress container. A cycle is only
// triggered if a bar was changed since the last one, or every idle interval, so the
// unchanged view is not redrawn over and over again.
type refresher struct {
	ch       chan any
	done     chan struct{}
	stop     sync.Once
	dirty    atomic.Bool
	interval time.Duration
	idle     time.Duration
}

func newRefresher(interval, idle time.Duration) *refresher {
	r := &refresher{
		ch:       make(chan any),
		done:     make(chan struct{}),
		interval: interval,
		idle:     idle,
	}
	go r.run()
	return r
}

func (r *refresher) run() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	last := time.Now()
	for {
		select {
		case <-r.done:
			return
		case t := <-ticker.C:
			if !r.dirty.Swap(false) && t.Sub(last) < r.idle {
				continue
			}
			last = t
			select {
			case r.ch <- nil:
			case <-r.done:
				return
			}
		}
	}
}

// touch marks the view as changed, so the next cycle redraws it.
func (r *refresher) touch() {
	if r != nil {
		r.dirty.Store(true)
	}
}

// close stops triggering render cycles.
func (r *refresher) close() {
	if r != nil {
		r.stop.Do(func() { close(r.done) })
	}
}

func newMpbProgress(prog *mpb.Progress, resources, labelWidth int) *mpbProgress {
	sh := &shared{
		mainTotal:  int64(resources),
		labelWidth: labelWidth,
	}
	sh.mainBar = newMpbMainBar(prog, resources, sh)
	return &mpbProgress{
		prog:   prog,
		shared: sh,
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
	refresh     *refresher
}

// barState is the mutable part rendered by the decorators of a worker bar.
// It is guarded by its own mutex, which must never be held while calling
// methods of the bar itself, as the bar renders its decorators in its own
// goroutine.
type barState struct {
	mx      sync.Mutex
	icon    string
	label   string
	details string
	elapsed decor.Decorator
}

func (b *barState) set(icon, label, details string, elapsed decor.Decorator) {
	b.mx.Lock()
	defer b.mx.Unlock()
	b.icon, b.label, b.details, b.elapsed = icon, label, details, elapsed
}

func (b *barState) iconAndLabel() (icon, label string) {
	b.mx.Lock()
	defer b.mx.Unlock()
	return b.icon, b.label
}

func (b *barState) detailsValue() string {
	b.mx.Lock()
	defer b.mx.Unlock()
	return b.details
}

func (b *barState) elapsedDecor() decor.Decorator {
	b.mx.Lock()
	defer b.mx.Unlock()
	return b.elapsed
}

type mpbProgress struct {
	id     int
	prog   *mpb.Progress
	shared *shared

	// state is the currently rendered step of the worker bar.
	state barState
	// kindElapsed measures the export duration of the current kind.
	kindElapsed decor.Decorator

	// resourceBar is the single bar of a worker. It is reused for all steps,
	// so the number of rendered lines never changes and the output does not flicker.
	resourceBar     *mpb.Bar
	resourceTotal   int64
	resourceCurrent int64
}

func (*mpbProgress) Async() bool {
	return true
}

func (m *mpbProgress) Run() error {
	m.prog.Wait()
	m.shared.refresh.close()
	return nil
}

func (m *mpbProgress) NewWorker() progress.Progress {
	return m.addWorker()
}

func (m *mpbProgress) addWorker() *mpbProgress {
	w := &mpbProgress{
		prog:        m.prog,
		shared:      m.shared,
		kindElapsed: decor.NewElapsed(decor.ET_STYLE_GO, time.Now()),
	}
	m.shared.mx.Lock()
	w.id = len(m.shared.workers) + 1
	m.shared.workers = append(m.shared.workers, w)
	m.shared.mx.Unlock()
	w.resourceBar = w.newResourceBar()
	return w
}

// newResourceBar creates the persistent bar of a worker. It is created with an
// unknown total, so it never completes on its own and can be reused for all
// steps of the worker. It is completed by Finish.
func (m *mpbProgress) newResourceBar() *mpb.Bar {
	return m.prog.New(0, barStyle(),
		mpb.PrependDecorators(
			elapsedDecorator(m.state.elapsedDecor),
			labelDecorator(m.state.iconAndLabel, m.shared, styleLabel),
		),
		mpb.AppendDecorators(
			countersDecorator(),
			detailsDecorator(m.state.detailsValue),
		),
	)
}

func (m *mpbProgress) Reset() {
	m.kindElapsed = decor.NewElapsed(decor.ET_STYLE_GO, time.Now())
}

// Finish completes all bars, making sure they all end up at 100%.
func (m *mpbProgress) Finish() {
	m.shared.mx.Lock()
	workers := append([]*mpbProgress{m}, m.shared.workers...)
	if inc := m.shared.mainTotal - m.shared.mainCurrent; inc > 0 {
		m.shared.mainCurrent = m.shared.mainTotal
		m.shared.mainBar.IncrInt64(inc)
	}
	m.shared.mx.Unlock()

	for _, w := range workers {
		w.completeResourceBar()
	}
}

// completeResourceBar fills the resource bar up to its total and completes it.
func (m *mpbProgress) completeResourceBar() {
	m.shared.mx.Lock()
	bar := m.resourceBar
	total := m.resourceTotal
	m.resourceCurrent = total
	m.shared.mx.Unlock()

	if bar == nil {
		return
	}
	bar.SetTotal(total, true)
	m.shared.refresh.touch()
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
			elapsedDecorator(func() decor.Decorator { return elapsed }),
			labelDecorator(func() (string, string) { return iconMain, mainBarTitle }, sh, styleMain),
		),
		mpb.AppendDecorators(
			countersDecorator(),
			detailsDecorator(func() string { return "" }),
		),
	)
}

func (m *mpbProgress) NewSearchBar(step progress.Step) {
	// the search has no known total, the bar is filled as soon as the query returned
	m.startStep(iconSearch, step, 1, nil)
}

func (m *mpbProgress) NewExportBar(step progress.Step) {
	if step.Total <= 0 {
		// nothing to export, complete the current (search) step
		m.fillResourceBar()
		return
	}
	m.startStep(iconExport, step, int64(step.Total), m.kindElapsed)
}

// startStep points the reused resource bar to a new step.
func (m *mpbProgress) startStep(icon string, step progress.Step, total int64, elapsed decor.Decorator) {
	m.state.set(icon, step.CurrentKind, pageDetails(step), elapsed)

	m.shared.mx.Lock()
	bar := m.resourceBar
	m.resourceTotal = total
	m.resourceCurrent = 0
	m.shared.mx.Unlock()

	if bar == nil {
		return
	}
	bar.SetCurrent(0)
	bar.SetTotal(total, false)
	m.shared.refresh.touch()
}

// fillResourceBar fills the resource bar up to its total, without completing it.
func (m *mpbProgress) fillResourceBar() {
	m.shared.mx.Lock()
	bar := m.resourceBar
	inc := m.resourceTotal - m.resourceCurrent
	m.resourceCurrent = m.resourceTotal
	m.shared.mx.Unlock()

	if bar == nil || inc <= 0 {
		return
	}
	bar.IncrInt64(inc)
	m.shared.refresh.touch()
}

// elapsedDecorator renders the elapsed time in a fixed width column.
func elapsedDecorator(get func() decor.Decorator) decor.Decorator {
	return decor.Meta(decor.Any(func(s decor.Statistics) string {
		el := ""
		if elapsed := get(); elapsed != nil {
			el, _ = elapsed.Decor(s)
		}
		return fmt.Sprintf("%7s ", strings.TrimSpace(el))
	}), render(styleDetails))
}

// labelDecorator renders the icon and the label in aligned columns.
func labelDecorator(get func() (string, string), sh *shared, style lipgloss.Style) decor.Decorator {
	return decor.Meta(decor.Any(func(decor.Statistics) string {
		icon, value := get()
		if icon == "" {
			// the icons are emoji occupying two cells
			icon = "  "
		}
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
func detailsDecorator(get func() string) decor.Decorator {
	return decor.Meta(decor.Any(func(decor.Statistics) string {
		return fmt.Sprintf("  %-9s", get())
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
	if m.shared.mainCurrent >= m.shared.mainTotal {
		m.shared.mx.Unlock()
		return
	}
	m.shared.mainCurrent++
	bar := m.shared.mainBar
	m.shared.mx.Unlock()

	bar.Increment()
	m.shared.refresh.touch()
}

func (m *mpbProgress) IncrementResourceBarBy(_, inc int) {
	m.shared.mx.Lock()
	bar := m.resourceBar
	if bar == nil || inc <= 0 {
		m.shared.mx.Unlock()
		return
	}
	// never increment beyond the total
	if remaining := m.resourceTotal - m.resourceCurrent; int64(inc) > remaining {
		inc = int(remaining)
	}
	if inc <= 0 {
		m.shared.mx.Unlock()
		return
	}
	m.resourceCurrent += int64(inc)
	m.shared.mx.Unlock()

	bar.IncrBy(inc)
	m.shared.refresh.touch()
}
