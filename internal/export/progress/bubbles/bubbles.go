package bubbles

import (
	"fmt"
	"math"
	"strings"

	bp "charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bakito/kubexporter/internal/export/progress"
	"github.com/bakito/kubexporter/internal/types"
)

const (
	padding  = 2
	maxWidth = 150

	mainProgressTitle = "Resources"

	iconSearch = "🔍"
	iconExport = "👷"
	iconDone   = "✅"
)

func NewProgress(resources []*types.GroupResource) progress.Progress {
	return newBubblesProgress(resources)
}

func newBubblesProgress(resources []*types.GroupResource) *bubblesProgress {
	var maxLen float64
	for _, res := range resources {
		maxLen = math.Max(maxLen, float64(len(res.GroupKind())))
	}
	m := &model{
		resources:    len(resources),
		mainProgress: newProgressModel(),
		maxLen:       int(maxLen),
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
		bp.WithFillCharacters('█', '░'),
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
	maxLen         int
}

type workerState struct {
	progress.Step
	percent float64
	icon    string
}

func (*model) Init() tea.Cmd {
	return nil
}

func (m *model) mainPercent() float64 {
	if m.resources <= 0 {
		return 1
	}
	return math.Min(float64(m.done)/float64(m.resources), 1)
}

// worker returns the state of the given worker id or nil if the id is unknown.
func (m *model) worker(id int) (*workerState, *bp.Model) {
	if id < 1 || id > len(m.workerStates) {
		return nil, nil
	}
	return m.workerStates[id-1], m.workerProgress[id-1]
}

func (m *model) startBar(step progress.Step, icon string) {
	state, bar := m.worker(step.WorkerID)
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
	bar.SetWidth(m.mainProgress.Width() - m.maxLen - 3 + len(mainProgressTitle))
}

func (m *model) Update(msgIn tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msgIn.(type) {
	case tea.KeyPressMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.mainProgress.SetWidth(msg.Width - padding*2 - len(mainProgressTitle) - 3)
		if m.mainProgress.Width() > maxWidth {
			m.mainProgress.SetWidth(maxWidth)
		}
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
		state, _ := m.worker(msg.workerID)
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
	pad := strings.Repeat(" ", padding)
	view := "\n" + pad + mainProgressTitle + ": " + m.mainProgress.ViewAs(m.mainPercent()) + "\n\n"
	var viewSb strings.Builder
	for i, workerProgress := range m.workerProgress {
		viewSb.WriteString(pad + fmt.Sprintf(
			"%s %s: %s",
			m.workerStates[i].icon,
			m.workerStates[i].CurrentKind,
			strings.Repeat(" ", m.maxLen-len(m.workerStates[i].CurrentKind)),
		) + workerProgress.ViewAs(m.workerStates[i].percent) + "\n")
	}
	view += viewSb.String()
	return tea.NewView(view)
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
