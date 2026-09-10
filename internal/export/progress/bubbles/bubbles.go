package bubbles

import (
	"fmt"
	"math"
	"strings"
	"time"

	bp "charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/bakito/kubexporter/internal/export/progress"
	"github.com/bakito/kubexporter/internal/types"
)

const (
	padding  = 2
	maxWidth = 150
	// minWidth is the minimal width of a progress bar.
	minWidth = 20
	// iconWidth is the display width of the emoji icons.
	iconWidth = 2

	mainProgressTitle = "Resources"

	iconMain   = "📦"
	iconSearch = "🔍"
	iconExport = "👷"
	iconDone   = "✅"
)

var (
	styleTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#316CE6"))
	styleLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8A8A"))
	styleMain    = lipgloss.NewStyle().Bold(true)
	styleDetails = lipgloss.NewStyle().Foreground(lipgloss.Color("#5F5F5F"))
)

func NewProgress(resources []*types.GroupResource) progress.Progress {
	return newBubblesProgress(resources)
}

func newBubblesProgress(resources []*types.GroupResource) *bubblesProgress {
	labelWidth := runewidth.StringWidth(mainProgressTitle)
	for _, res := range resources {
		labelWidth = max(labelWidth, runewidth.StringWidth(res.GroupKind()))
	}
	m := &model{
		resources:    len(resources),
		mainProgress: newProgressModel(),
		labelWidth:   labelWidth,
		start:        time.Now(),
	}
	return &bubblesProgress{
		model: m,
		// the program is created up front, so messages can be sent before Run() is called
		program: tea.NewProgram(m),
	}
}

func newProgressModel() bp.Model {
	return bp.New(
		bp.WithColors(lipgloss.Color("#6B89E8"), lipgloss.Color("#316CE6")),
		bp.WithScaled(true),
		bp.WithFillCharacters('━', '─'),
		bp.WithWidth(minWidth),
	)
}

type bubblesProgress struct {
	model   *model
	program *tea.Program
}

func (*bubblesProgress) Async() bool {
	return true
}

func (b *bubblesProgress) Run() error {
	_, err := b.program.Run()
	return err
}

func (b *bubblesProgress) NewSearchBar(step progress.Step) {
	b.program.Send(searchMsg(step))
}

func (b *bubblesProgress) NewExportBar(step progress.Step) {
	b.program.Send(exportMsg(step))
}

func (*bubblesProgress) Reset() {
	// not applicable
}

func (b *bubblesProgress) Finish() {
	b.program.Send(finishMsg(true))
}

func (b *bubblesProgress) NewWorker() progress.Progress {
	b.model.workerProgress = append(b.model.workerProgress, new(newProgressModel()))
	b.model.workerStates = append(b.model.workerStates, &workerState{})
	return b
}

func (b *bubblesProgress) IncrementMainBar() {
	b.program.Send(updateMainMsg(1))
}

func (b *bubblesProgress) IncrementResourceBarBy(id, inc int) {
	b.program.Send(updateWorkerMsq{workerID: id, incr: inc})
}

type model struct {
	resources      int
	done           int
	mainProgress   bp.Model
	workerProgress []*bp.Model
	workerStates   []*workerState
	labelWidth     int
	start          time.Time
}

type workerState struct {
	progress.Step
	percent float64
	icon    string
}

// details describes the current step of a worker.
func (w *workerState) details() string {
	if w.PageSize > 0 && w.CurrentPage > 0 {
		return fmt.Sprintf("page %d", w.CurrentPage)
	}
	return ""
}

func (*model) Init() tea.Cmd {
	return tick()
}

// tickMsg triggers a redraw, so the elapsed time keeps running.
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) mainPercent() float64 {
	if m.resources <= 0 {
		return 1
	}
	return math.Min(float64(m.done)/float64(m.resources), 1)
}

// worker returns the state of the given worker id or nil if the id is unknown.
func (m *model) worker(id int) *workerState {
	if id < 1 || id > len(m.workerStates) {
		return nil
	}
	return m.workerStates[id-1]
}

func (m *model) startBar(step progress.Step, icon string) {
	state := m.worker(step.WorkerID)
	if state == nil {
		return
	}
	state.Step = step
	state.icon = icon
	// as long as nothing was processed, the bar is empty
	state.percent = 0
	if step.Total == 0 && icon == iconExport {
		// nothing to export for this step
		state.percent = 1
	}
}

// setWidth adapts the width of all bars to the available terminal width.
func (m *model) setWidth(width int) {
	// icon, label, bar and the details column
	available := width - 2*padding - iconWidth - 1 - m.labelWidth - 2 - m.detailsWidth() - 2
	barWidth := min(max(available, minWidth), maxWidth)
	m.mainProgress.SetWidth(barWidth)
	for _, bar := range m.workerProgress {
		bar.SetWidth(barWidth)
	}
}

// detailsWidth is the width of the trailing details column.
func (m *model) detailsWidth() int {
	width := runewidth.StringWidth(m.mainDetails())
	for _, state := range m.workerStates {
		width = max(width, runewidth.StringWidth(state.details()))
	}
	return width
}

func (m *model) mainDetails() string {
	return fmt.Sprintf("%d/%d", m.done, m.resources)
}

func (m *model) Update(msgIn tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msgIn.(type) {
	case tea.KeyPressMsg:
		return m, tea.Quit

	case tickMsg:
		return m, tick()

	case tea.WindowSizeMsg:
		m.setWidth(msg.Width)
		return m, nil

	case updateMainMsg:
		if m.done < m.resources {
			m.done += int(msg)
		}
		if m.done > m.resources {
			m.done = m.resources
		}
		return m, nil

	case searchMsg:
		m.startBar(progress.Step(msg), iconSearch)
		return m, nil

	case exportMsg:
		m.startBar(progress.Step(msg), iconExport)
		return m, nil

	case updateWorkerMsq:
		state := m.worker(msg.workerID)
		if state == nil {
			return m, nil
		}
		if state.Total <= 0 {
			// unknown total (search), the step is done as soon as it reports progress
			state.percent = 1
		} else {
			state.percent = math.Min(state.percent+float64(msg.incr)/float64(state.Total), 1)
		}
		return m, nil

	case finishMsg:
		// make sure everything ends up at 100%
		m.done = m.resources
		for _, state := range m.workerStates {
			state.percent = 1
			state.icon = iconDone
		}
		return m, tea.Quit

	default:
		return m, nil
	}
}

func (m *model) View() tea.View {
	return tea.NewView(m.render())
}

// render renders the whole progress view.
func (m *model) render() string {
	pad := strings.Repeat(" ", padding)
	title := styleTitle.Render("kubexporter")
	elapsed := styleDetails.Render("⏱ " + time.Since(m.start).Truncate(time.Second).String())

	var sb strings.Builder
	sb.WriteString("\n" + pad + title + "  " + elapsed + "\n\n")
	sb.WriteString(pad + row(
		iconMain,
		styleMain.Render(padTo(mainProgressTitle, m.labelWidth)),
		m.mainProgress.ViewAs(m.mainPercent()),
		m.mainDetails(),
	))
	sb.WriteString("\n")

	for i, bar := range m.workerProgress {
		state := m.workerStates[i]
		sb.WriteString(pad + row(
			state.icon,
			styleLabel.Render(padTo(state.CurrentKind, m.labelWidth)),
			bar.ViewAs(state.percent),
			state.details(),
		))
	}
	return sb.String()
}

// row renders a single aligned progress line.
// All icons are emoji occupying two cells.
func row(icon, label, bar, details string) string {
	return icon + " " + label + "  " + bar + "  " + styleDetails.Render(details) + "\n"
}

// padTo pads the given value to the given display width.
func padTo(value string, width int) string {
	return value + strings.Repeat(" ", max(width-runewidth.StringWidth(value), 0))
}

type (
	updateMainMsg   int
	updateWorkerMsq struct {
		workerID int
		incr     int
	}
)

type (
	finishMsg bool
	searchMsg progress.Step
	exportMsg progress.Step
)
