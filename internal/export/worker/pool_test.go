package worker

import (
	"context"
	"sync"
	"testing"

	"github.com/bakito/kubexporter/internal/types"
)

// mockWorker is a simple mock implementation of the Worker interface for testing.
type mockWorker struct {
	id      int
	stopped bool
	stats   Stats
}

func (*mockWorker) GenerateWork(
	_ context.Context,
	s *sync.WaitGroup,
	out chan *types.GroupResource,
) func(resource *types.GroupResource) {
	return func(resource *types.GroupResource) {
		defer s.Done()
		out <- resource
	}
}

func (m *mockWorker) Stop() Stats {
	m.stopped = true
	return m.stats
}

func TestRunExport_empty_workers(t *testing.T) {
	ctx := context.Background()
	var workers []Worker
	resources := []*types.GroupResource{}

	st, err := RunExport(ctx, workers, resources)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if st == nil {
		t.Fatal("expected stats to be non-nil")
	}
}

func TestRunExport_single_worker_single_resource(t *testing.T) {
	ctx := context.Background()
	worker := &mockWorker{
		id: 1,
		stats: Stats{
			Kinds:     1,
			Resources: 5,
		},
	}
	workers := []Worker{worker}
	resources := []*types.GroupResource{
		{
			APIGroup:   "apps",
			APIVersion: "v1",
		},
	}

	st, err := RunExport(ctx, workers, resources)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if st == nil {
		t.Fatal("expected stats to be non-nil")
	}
	if st.Kinds != 1 {
		t.Errorf("expected 1 kind, got %d", st.Kinds)
	}
	if st.Resources != 5 {
		t.Errorf("expected 5 resources, got %d", st.Resources)
	}
	if !worker.stopped {
		t.Error("expected worker to be stopped")
	}
}

func TestRunExport_multiple_workers(t *testing.T) {
	ctx := context.Background()
	worker1 := &mockWorker{
		id: 1,
		stats: Stats{
			Kinds:     2,
			Resources: 10,
			Pages:     3,
		},
	}
	worker2 := &mockWorker{
		id: 2,
		stats: Stats{
			Kinds:     1,
			Resources: 5,
			Pages:     2,
		},
	}
	workers := []Worker{worker1, worker2}
	resources := []*types.GroupResource{
		{APIGroup: "apps", APIVersion: "v1"},
		{APIGroup: "", APIVersion: "v1"},
	}

	st, err := RunExport(ctx, workers, resources)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if st == nil {
		t.Fatal("expected stats to be non-nil")
	}
	if st.Kinds != 3 {
		t.Errorf("expected 3 kinds, got %d", st.Kinds)
	}
	if st.Resources != 15 {
		t.Errorf("expected 15 resources, got %d", st.Resources)
	}
	if st.Pages != 5 {
		t.Errorf("expected 5 pages, got %d", st.Pages)
	}
}

func TestRunExport_with_namespaces(t *testing.T) {
	ctx := context.Background()
	worker := &mockWorker{
		id: 1,
		stats: Stats{
			Kinds:     1,
			Resources: 5,
			namespaces: map[string]bool{
				"default": true,
				"test":    true,
			},
		},
	}
	workers := []Worker{worker}
	resources := []*types.GroupResource{
		{APIGroup: "apps", APIVersion: "v1"},
	}

	st, err := RunExport(ctx, workers, resources)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if st.Namespaces() != 2 {
		t.Errorf("expected 2 namespaces, got %d", st.Namespaces())
	}
}

func TestRunExport_canceled_context(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	worker := &mockWorker{
		id: 1,
		stats: Stats{
			Kinds:     1,
			Resources: 5,
		},
	}
	workers := []Worker{worker}
	resources := []*types.GroupResource{
		{APIGroup: "apps", APIVersion: "v1"},
	}

	st, err := RunExport(ctx, workers, resources)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if st == nil {
		t.Fatal("expected stats to be non-nil")
	}
}
