package worker

import (
	"testing"
	"time"

	"github.com/bakito/kubexporter/internal/export/progress"
)

// countingProgress records the increments of the resource bar.
type countingProgress struct {
	progress.Progress
	total     int
	increment int
}

func (*countingProgress) Async() bool {
	return false
}

func (c *countingProgress) NewWorker() progress.Progress {
	return c
}

func (*countingProgress) Run() error {
	return nil
}

func (*countingProgress) Reset() {
}

func (*countingProgress) Finish() {
}

func (*countingProgress) NewSearchBar(_ progress.Step) {
}

func (c *countingProgress) NewExportBar(step progress.Step) {
	c.total = step.Total
	c.increment = 0
}

func (*countingProgress) IncrementMainBar() {
}

func (c *countingProgress) IncrementResourceBarBy(_, inc int) {
	c.increment += inc
}

func TestWorker_exportProgressReaches100(t *testing.T) {
	tests := []struct {
		name     string
		asLists  bool
		excluded bool
	}{
		{name: "single resources"},
		{name: "single resources with excluded instances", excluded: true},
		{name: "lists", asLists: true},
		{name: "lists with excluded instances", asLists: true, excluded: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := setupWorker(t)
			prog := &countingProgress{}
			w.prog = prog
			w.config.AsLists = tt.asLists
			if tt.excluded {
				// the test deployments have no creation timestamp, so they are all excluded
				w.config.CreatedWithin = time.Minute
			}

			res, ul := getTestData()
			prog.NewExportBar(progress.Step{WorkerID: 1, Total: len(ul.Items)})

			if tt.asLists {
				w.exportLists(res, ul)
			} else {
				w.exportSingleResources(res, ul)
			}

			if prog.increment != prog.total {
				t.Errorf("expected the resource bar to be incremented %d times, but was %d",
					prog.total, prog.increment)
			}
		})
	}
}
