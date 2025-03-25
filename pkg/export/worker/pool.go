package worker

import (
	"context"
	"sync"

	wp "github.com/vardius/worker-pool/v2"

	"github.com/bakito/kubexporter/pkg/types"
)

// RunExport run the export wit the given workers.
func RunExport(ctx context.Context, workers []Worker, resources []*types.GroupResource) (*Stats, error) {
	var wg sync.WaitGroup

	poolSize := len(resources)

	// create new pool
	pool := wp.New(poolSize)
	out := make(chan *types.GroupResource, poolSize)

	for _, w := range workers {
		if err := pool.AddWorker(w.GenerateWork(ctx, &wg, out)); err != nil {
			return nil, err
		}
	}

	wg.Add(len(resources))

	for _, res := range resources {
		if err := pool.Delegate(res); err != nil {
			return nil, err
		}
	}

	// stop all workers after jobs are done
	wg.Wait()
	close(out)
	pool.Stop()
	st := &Stats{}
	for _, w := range workers {
		s := w.Stop()
		st.Add(&s)
	}
	return st, nil
}
