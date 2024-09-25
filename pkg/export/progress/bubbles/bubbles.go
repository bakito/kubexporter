package bubbles

import (
	"github.com/bakito/kubexporter/pkg/export/progress"
	bp "github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

const (
	padding  = 2
	maxWidth = 200
)

func NewProgress(resources int) progress.Progress {
	return &bubblesProgress{
		model: &model{
			resources:    float64(resources),
			mainProgress: bp.New(bp.WithScaledGradient("#FF7CCB", "#FDFF8C")),
			mainPercent:  1 / float64(resources),
		},
	}
}

type bubblesProgress struct {
	model   *model
	program *tea.Program
}

func (b *bubblesProgress) Run() {
	b.program = tea.NewProgram(b.model)

	go func() {
		_, _ = b.program.Run()
	}()
}

func (b *bubblesProgress) NewSearchBar(currentKind string, pageSize int, currentPage int) {
}

func (b *bubblesProgress) NewExportBar(currentKind string, pageSize int, currentPage int, size int) {
}

func (b *bubblesProgress) Wait() {
	b.program.Send(exit(true))
}

func (b *bubblesProgress) Reset() {
	// not applicable
}

func (b *bubblesProgress) NewWorker() progress.Progress {
	b.model.workerProgress = append(b.model.workerProgress, bp.New(bp.WithScaledGradient("#FF7CCB", "#FDFF8C")))
	return b
}

func (b *bubblesProgress) IncrementMainBar() {
	b.program.Send(updateMainMsq(1))
}

func (b *bubblesProgress) IncrementResourceBarBy(i int) {
}

type model struct {
	resources      float64
	mainProgress   bp.Model
	workerProgress []bp.Model
	mainPercent    float64
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.mainProgress.Width = msg.Width - padding*2 - 4
		if m.mainProgress.Width > maxWidth {
			m.mainProgress.Width = maxWidth
		}
		for _, workerProgress := range m.workerProgress {

			workerProgress.Width = msg.Width - padding*2 - 4
			if workerProgress.Width > maxWidth {
				workerProgress.Width = maxWidth
			}
		}
		return m, nil

	case updateMainMsq:
		m.mainPercent += 1 / m.resources
		if m.mainPercent > 1.0 {
			m.mainPercent = 1.0
			return m, tea.Quit
		}

		return m, nil

	case exit:
		return m, tea.Quit

	default:
		return m, nil
	}
}

func (m *model) View() string {
	pad := strings.Repeat(" ", padding)
	return "\n" +
		"Resources: " + pad + m.mainProgress.ViewAs(m.mainPercent) + "\n\n"
}

type updateMainMsq int
type exit bool
