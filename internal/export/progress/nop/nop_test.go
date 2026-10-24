package nop

import (
	"testing"

	"github.com/bakito/kubexporter/internal/export/progress"
)

func TestNewProgress(t *testing.T) {
	p := NewProgress()
	if p == nil {
		t.Fatal("expected progress to be non-nil")
	}
}

func TestNilProgress_Async(t *testing.T) {
	p := NewProgress()
	if p.Async() {
		t.Error("expected Async() to return false")
	}
}

func TestNilProgress_Run(t *testing.T) {
	p := NewProgress()
	err := p.Run()
	if err != nil {
		t.Errorf("expected Run() to return nil, got %v", err)
	}
}

func TestNilProgress_NewSearchBar(_ *testing.T) {
	p := NewProgress()
	// Should not panic
	p.NewSearchBar(progress.Step{
		WorkerID:    1,
		CurrentKind: "Deployment",
		PageSize:    10,
		CurrentPage: 1,
		Total:       100,
	})
}

func TestNilProgress_NewExportBar(_ *testing.T) {
	p := NewProgress()
	// Should not panic
	p.NewExportBar(progress.Step{
		WorkerID:    1,
		CurrentKind: "Deployment",
		PageSize:    10,
		CurrentPage: 1,
		Total:       100,
	})
}

func TestNilProgress_Reset(_ *testing.T) {
	p := NewProgress()
	// Should not panic
	p.Reset()
}

func TestNilProgress_Finish(_ *testing.T) {
	p := NewProgress()
	// Should not panic
	p.Finish()
}

func TestNilProgress_NewWorker(t *testing.T) {
	p := NewProgress()
	w := p.NewWorker()
	if w == nil {
		t.Error("expected NewWorker() to return non-nil")
	}
}

func TestNilProgress_IncrementMainBar(_ *testing.T) {
	p := NewProgress()
	// Should not panic
	p.IncrementMainBar()
}

func TestNilProgress_IncrementResourceBarBy(_ *testing.T) {
	p := NewProgress()
	// Should not panic
	p.IncrementResourceBarBy(1, 10)
}
