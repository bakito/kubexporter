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

// TestResourceBarIsReused makes sure every step reuses the same bar, so the number
// of rendered lines stays constant and the output does not flicker.
func TestResourceBarIsReused(t *testing.T) {
	p := newTestProgress(1)
	w := p.addWorker()

	bar := w.resourceBar
	if bar == nil {
		t.Fatal("expected the worker to have a bar")
	}

	w.NewSearchBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment"})
	w.IncrementResourceBarBy(1, 1)
	if w.resourceCurrent != w.resourceTotal {
		t.Errorf("expected the search step to be at %d, but was at %d", w.resourceTotal, w.resourceCurrent)
	}

	w.NewExportBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment", Total: 3})
	if w.resourceCurrent != 0 || w.resourceTotal != 3 {
		t.Errorf("expected the export step to be at 0/3, but was at %d/%d", w.resourceCurrent, w.resourceTotal)
	}
	w.IncrementResourceBarBy(1, 3)
	// must not overflow
	w.IncrementResourceBarBy(1, 10)
	if w.resourceCurrent != 3 {
		t.Errorf("expected resource bar to be at 3, but was at %d", w.resourceCurrent)
	}

	if w.resourceBar != bar {
		t.Error("expected the worker bar to be reused for all steps")
	}
	if bar.Completed() {
		t.Error("expected the reused bar to stay open until the export is finished")
	}
}

func TestExportBarWithoutItemsFillsSearchStep(t *testing.T) {
	p := newTestProgress(1)
	w := p.addWorker()

	w.NewSearchBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment"})
	w.NewExportBar(progress.Step{WorkerID: 1, CurrentKind: "Deployment", Total: 0})

	if w.resourceCurrent != w.resourceTotal {
		t.Errorf("expected the search step to be filled to %d, but was at %d", w.resourceTotal, w.resourceCurrent)
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

// TestRefresherOnlyRefreshesOnChange makes sure the console is not redrawn while
// nothing changed, which is the main source of flickering on slow consoles.
func TestRefresherOnlyRefreshesOnChange(t *testing.T) {
	r := newRefresher(10*time.Millisecond, time.Hour)
	defer r.close()

	select {
	case <-r.ch:
		t.Error("expected no refresh while nothing changed")
	case <-time.After(100 * time.Millisecond):
	}

	r.touch()
	select {
	case <-r.ch:
	case <-time.After(time.Second):
		t.Error("expected a refresh after a change")
	}
}

// TestRefresherRefreshesWhenIdle makes sure the elapsed time columns keep running.
func TestRefresherRefreshesWhenIdle(t *testing.T) {
	r := newRefresher(10*time.Millisecond, 20*time.Millisecond)
	defer r.close()

	select {
	case <-r.ch:
	case <-time.After(time.Second):
		t.Error("expected an idle refresh")
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
