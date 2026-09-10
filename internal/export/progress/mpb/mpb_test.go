package mpb

import (
	"io"
	"testing"
	"time"

	"github.com/vbauerster/mpb/v8"

	"github.com/bakito/kubexporter/internal/export/progress"
)

func newTestProgress(resources int) *mpbProgress {
	return newMpbProgress(mpb.New(mpb.WithOutput(io.Discard), mpb.WithWidth(80)), resources, len("Resources"))
}

func TestMainBarIsCompleted(t *testing.T) {
	p := newTestProgress(3)
	w := p.NewWorker()

	for range 3 {
		w.IncrementMainBar()
	}

	if p.shared.mainCurrent != p.shared.mainTotal {
		t.Errorf("expected main bar to be at %d, but was at %d", p.shared.mainTotal, p.shared.mainCurrent)
	}
	if !p.shared.mainBar.Completed() {
		t.Error("expected main bar to be completed")
	}
}

func TestMainBarDoesNotOverflow(t *testing.T) {
	p := newTestProgress(2)
	w := p.NewWorker()

	for range 5 {
		w.IncrementMainBar()
	}

	if p.shared.mainCurrent != 2 {
		t.Errorf("expected main bar to be at 2, but was at %d", p.shared.mainCurrent)
	}
}

func TestResourceBarsAreCompleted(t *testing.T) {
	p := newTestProgress(1)
	w := p.addWorker()

	w.NewSearchBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment"})
	searchBar := w.resourceBar
	w.IncrementResourceBarBy(1, 1)
	if !searchBar.Completed() {
		t.Error("expected search bar to be completed")
	}

	w.NewExportBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment", Total: 3})
	exportBar := w.resourceBar
	w.IncrementResourceBarBy(1, 3)
	if !exportBar.Completed() {
		t.Error("expected export bar to be completed")
	}
	// must not overflow
	w.IncrementResourceBarBy(1, 10)
	if w.resourceCurrent != 3 {
		t.Errorf("expected resource bar to be at 3, but was at %d", w.resourceCurrent)
	}
}

func TestIncompleteSearchBarIsCompletedOnNextBar(t *testing.T) {
	p := newTestProgress(1)
	w := p.addWorker()

	// the search bar is never incremented
	w.NewSearchBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment"})
	searchBar := w.resourceBar

	w.NewExportBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment", Total: 2})
	if !searchBar.Completed() {
		t.Error("expected search bar to be completed when it is replaced")
	}
}

func TestExportBarWithoutItemsCompletesSearchBar(t *testing.T) {
	p := newTestProgress(1)
	w := p.addWorker()

	w.NewSearchBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment"})
	searchBar := w.resourceBar
	w.NewExportBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment", Total: 0})

	if !searchBar.Completed() {
		t.Error("expected search bar to be completed")
	}
	if w.resourceBar != searchBar {
		t.Error("expected no new bar to be created for an empty export")
	}
}

// TestFinishCompletesEverything makes sure Run() (mpb.Wait) does not block, even if the
// export was aborted with partially filled bars.
func TestFinishCompletesEverything(t *testing.T) {
	p := newTestProgress(3)
	w1 := p.addWorker()
	w2 := p.addWorker()

	w1.NewSearchBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment"})
	w1.IncrementResourceBarBy(1, 1)
	w1.NewExportBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment", Total: 10})
	w1.IncrementResourceBarBy(1, 4)

	w2.NewSearchBar(progress.Step{WorkerID: 2, CurrentKind: "Pod"})

	w1.IncrementMainBar()

	p.Finish()

	if p.shared.mainCurrent != 3 {
		t.Errorf("expected main bar to be at 3, but was at %d", p.shared.mainCurrent)
	}
	if !p.shared.mainBar.Completed() {
		t.Error("expected main bar to be completed")
	}
	if !w1.resourceBar.Completed() {
		t.Error("expected worker 1 resource bar to be completed")
	}
	if !w2.resourceBar.Completed() {
		t.Error("expected worker 2 resource bar to be completed")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = p.Run()
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Run() did not return, progress bars were not completed")
	}
}

func TestWorkerIDs(t *testing.T) {
	p := newTestProgress(1)
	w1 := p.addWorker()
	w2 := p.addWorker()

	if w1.id != 1 || w2.id != 2 {
		t.Errorf("expected worker ids 1 and 2, but got %d and %d", w1.id, w2.id)
	}
}
