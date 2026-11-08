package worker

import (
	"context"

	wp "github.com/vardius/worker-pool/v3"

	"github.com/bakito/kubexporter/internal/types"
)

// RunExport run the export wit the given workers.
func RunExport(ctx context.Context, workers []Worker, resources []*types.GroupResource) (*Stats, error) {
	poolSize := len(resources)

	// create new pool
	pool := wp.New[*types.GroupResource](poolSize)

	for _, w := range workers {
		pool.AddWorker(w.GenerateWork())
	}

	for _, res := range resources {
		if err := pool.Delegate(ctx, res); err != nil {
			return nil, err
		}
	}

	// stop all workers after jobs are done
	pool.Stop()
	st := &Stats{}
	for _, w := range workers {
		st.Add(new(w.Stop()))
	}
	return st, nil
}
