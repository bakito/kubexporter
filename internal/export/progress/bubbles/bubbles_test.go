package bubbles

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bakito/kubexporter/internal/export/progress"
	"github.com/bakito/kubexporter/internal/types"
)

func testResources(kinds ...string) []*types.GroupResource {
	var resources []*types.GroupResource
	for _, kind := range kinds {
		resources = append(resources, &types.GroupResource{
			APIGroup:    "",
			APIVersion:  "v1",
			APIResource: metav1.APIResource{Kind: kind},
		})
	}
	return resources
}

func newTestProgress(kinds ...string) (*bubblesProgress, *model) {
	p := newBubblesProgress(testResources(kinds...))
	return p, p.model
}

func TestMainBarStartsAtZero(t *testing.T) {
	_, m := newTestProgress("Pod", "Deployment", "Service")

	if m.mainPercent() != 0 {
		t.Errorf("expected main progress to start at 0, but was %v", m.mainPercent())
	}
}

func TestMainBarReaches100(t *testing.T) {
	p, m := newTestProgress("Pod", "Deployment", "Service")
	p.NewWorker()

	for range 3 {
		m.Update(updateMainMsg(1))
	}

	if m.mainPercent() != 1 {
		t.Errorf("expected main progress to be 1, but was %v", m.mainPercent())
	}
}

func TestMainBarDoesNotOverflow(t *testing.T) {
	_, m := newTestProgress("Pod", "Deployment")

	for range 5 {
		m.Update(updateMainMsg(1))
	}

	if m.mainPercent() != 1 {
		t.Errorf("expected main progress to be 1, but was %v", m.mainPercent())
	}
	if m.done != m.resources {
		t.Errorf("expected done to be %d, but was %d", m.resources, m.done)
	}
}

func TestSearchBar(t *testing.T) {
	p, m := newTestProgress("Pod")
	p.NewWorker()

	m.Update(searchMsg(progress.Step{WorkerID: 1, CurrentKind: "Pod"}))
	if m.workerStates[0].percent != 0 {
		t.Errorf("expected search progress to start at 0, but was %v", m.workerStates[0].percent)
	}

	m.Update(updateWorkerMsq{workerID: 1, incr: 1})
	if m.workerStates[0].percent != 1 {
		t.Errorf("expected search progress to be 1, but was %v", m.workerStates[0].percent)
	}
}

func TestExportBarReaches100(t *testing.T) {
	p, m := newTestProgress("Pod")
	p.NewWorker()

	m.Update(exportMsg(progress.Step{WorkerID: 1, CurrentKind: "Pod", Total: 4}))
	if m.workerStates[0].percent != 0 {
		t.Errorf("expected export progress to start at 0, but was %v", m.workerStates[0].percent)
	}

	for range 4 {
		m.Update(updateWorkerMsq{workerID: 1, incr: 1})
	}
	if m.workerStates[0].percent != 1 {
		t.Errorf("expected export progress to be 1, but was %v", m.workerStates[0].percent)
	}

	// must not overflow
	m.Update(updateWorkerMsq{workerID: 1, incr: 10})
	if m.workerStates[0].percent != 1 {
		t.Errorf("expected export progress to stay at 1, but was %v", m.workerStates[0].percent)
	}
}

func TestEmptyExportBarIsComplete(t *testing.T) {
	p, m := newTestProgress("Pod")
	p.NewWorker()

	m.Update(exportMsg(progress.Step{WorkerID: 1, CurrentKind: "Pod", Total: 0}))
	if m.workerStates[0].percent != 1 {
		t.Errorf("expected export progress to be 1, but was %v", m.workerStates[0].percent)
	}
}

func TestFinishCompletesEverything(t *testing.T) {
	p, m := newTestProgress("Pod", "Deployment")
	p.NewWorker()
	p.NewWorker()

	m.Update(exportMsg(progress.Step{WorkerID: 1, CurrentKind: "Pod", Total: 10}))
	m.Update(updateWorkerMsq{workerID: 1, incr: 3})
	m.Update(searchMsg(progress.Step{WorkerID: 2, CurrentKind: "Deployment"}))

	_, cmd := m.Update(finishMsg(true))
	if cmd == nil {
		t.Error("expected the program to be quit on finish")
	}

	if m.mainPercent() != 1 {
		t.Errorf("expected main progress to be 1, but was %v", m.mainPercent())
	}
	for i, state := range m.workerStates {
		if state.percent != 1 {
			t.Errorf("expected worker %d progress to be 1, but was %v", i+1, state.percent)
		}
		if state.icon != iconDone {
			t.Errorf("expected worker %d icon to be %q, but was %q", i+1, iconDone, state.icon)
		}
	}
}

func TestUnknownWorkerIsIgnored(_ *testing.T) {
	p, m := newTestProgress("Pod")
	p.NewWorker()

	// must not panic
	m.Update(searchMsg(progress.Step{WorkerID: 5, CurrentKind: "Pod"}))
	m.Update(updateWorkerMsq{workerID: 0, incr: 1})
}

func TestViewRowsAreAligned(t *testing.T) {
	p, m := newTestProgress("ConfigMap", "apps/StatefulSet")
	p.NewWorker()
	p.NewWorker()

	m.Update(tea.WindowSizeMsg{Width: 120})
	m.Update(exportMsg(progress.Step{WorkerID: 1, CurrentKind: "ConfigMap", Total: 10}))
	m.Update(updateWorkerMsq{workerID: 1, incr: 4})
	m.Update(searchMsg(progress.Step{WorkerID: 2, CurrentKind: "apps/StatefulSet"}))

	var columns []int
	for line := range strings.SplitSeq(m.render(), "\n") {
		plain := ansi.Strip(line)
		if idx := strings.IndexAny(plain, "━─"); idx >= 0 {
			columns = append(columns, idx)
		}
	}
	if len(columns) != 3 {
		t.Fatalf("expected 3 progress bars, but got %d", len(columns))
	}
	for i, c := range columns {
		if c != columns[0] {
			t.Errorf("bar of line %d starts at %d, but expected %d:\n%s", i, c, columns[0], m.render())
		}
	}
}
