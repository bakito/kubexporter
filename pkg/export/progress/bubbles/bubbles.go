package bubbles

import (
	"fmt"
	"math"
	"strings"

	bp "github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bakito/kubexporter/pkg/export/progress"
	"github.com/bakito/kubexporter/pkg/types"
)

const (
	padding  = 2
	maxWidth = 150

	mainProgressTitle = "Resources"
)

func NewProgress(resources []*types.GroupResource) progress.Progress {
	var maxLen float64
	for _, res := range resources {
		maxLen = math.Max(maxLen, float64(len(res.GroupKind())))
	}
	return &bubblesProgress{
		model: &model{
			resources:    float64(len(resources)),
			mainProgress: newProgressModel(),
			mainPercent:  1 / float64(len(resources)),
			maxLen:       int(maxLen),
		},
	}
}

func newProgressModel() bp.Model {
	return bp.New(
		bp.WithScaledGradient("#6B89E8", "#316CE6"),
		bp.WithFillCharacters('█', '░'),
	)
}

type bubblesProgress struct {
	model   *model
	program *tea.Program
}

func (b *bubblesProgress) Async() bool {
	return true
}

func (b *bubblesProgress) Run() error {
	b.program = tea.NewProgram(b.model)
	_, err := b.program.Run()
	if err != nil {
		return err
	}
	b.program.Send(exitMsg(true))
	return nil
}

func (b *bubblesProgress) NewSearchBar(step progress.Step) {
	b.program.Send(searchMsg(step))
}

func (b *bubblesProgress) NewExportBar(step progress.Step) {
	b.program.Send(exportMsg(step))
}

func (b *bubblesProgress) Reset() {
	// not applicable
}

func (b *bubblesProgress) NewWorker() progress.Progress {
	w := newProgressModel()
	b.model.workerProgress = append(b.model.workerProgress, &w)
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
	resources      float64
	mainProgress   bp.Model
	mainPercent    float64
	workerProgress []*bp.Model
	workerStates   []*workerState
	maxLen         int
}

type workerState struct {
	progress.Step
	percent float64
	icon    string
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msgIn tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msgIn.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.mainProgress.Width = msg.Width - padding*2 - len(mainProgressTitle) - 3
		if m.mainProgress.Width > maxWidth {
			m.mainProgress.Width = maxWidth
		}
		return m, nil

	case updateMainMsg:
		m.mainPercent += 1 / m.resources
		if m.mainPercent > 1.0 {
			m.mainPercent = 1.0
			return m, tea.Quit
		}

		return m, nil

	case searchMsg:
		m.workerStates[msg.WorkerID-1].Total = msg.Total
		if m.workerStates[msg.WorkerID-1].Total == 0 {
			m.workerStates[msg.WorkerID-1].percent = 1
		} else {
			m.workerStates[msg.WorkerID-1].percent = 0
		}
		m.workerStates[msg.WorkerID-1].Step = progress.Step(msg)
		m.workerStates[msg.WorkerID-1].icon = "🔍"
		m.workerProgress[msg.WorkerID-1].Width = m.mainProgress.Width - m.maxLen - 3 + len(mainProgressTitle)
		return m, nil
	case exportMsg:
		m.workerStates[msg.WorkerID-1].Total = msg.Total
		if m.workerStates[msg.WorkerID-1].Total == 0 {
			m.workerStates[msg.WorkerID-1].percent = 1
		} else {
			m.workerStates[msg.WorkerID-1].percent = 0
		}

		m.workerStates[msg.WorkerID-1].Step = progress.Step(msg)
		m.workerStates[msg.WorkerID-1].icon = "👷"
		m.workerProgress[msg.WorkerID-1].Width = m.mainProgress.Width - m.maxLen - 3 + len(mainProgressTitle)
		return m, nil
	case updateWorkerMsq:
		if m.workerStates[msg.workerID-1].Total == 0 {
			m.workerStates[msg.workerID-1].percent = 1
		} else {
			incr := float64(msg.incr) / float64(m.workerStates[msg.workerID-1].Total)
			m.workerStates[msg.workerID-1].percent += incr
		}
		return m, nil
	case exitMsg:
		return m, tea.Quit

	default:
		return m, nil
	}
}

func (m *model) View() string {
	pad := strings.Repeat(" ", padding)
	view := "\n" + pad + mainProgressTitle + ": " + m.mainProgress.ViewAs(m.mainPercent) + "\n\n"
	for i, workerProgress := range m.workerProgress {
		view += pad + fmt.Sprintf(
			"%s %s: %s",
			m.workerStates[i].icon,
			m.workerStates[i].CurrentKind,
			strings.Repeat(" ", m.maxLen-len(m.workerStates[i].CurrentKind)),
		) + workerProgress.ViewAs(m.workerStates[i].percent) + "\n"
	}
	return view
}

type (
	updateMainMsg   int
	updateWorkerMsq struct {
		workerID int
		incr     int
	}
)

type (
	exitMsg   bool
	searchMsg progress.Step
	exportMsg progress.Step
)
