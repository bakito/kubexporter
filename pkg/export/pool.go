package export

import (
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/vardius/worker-pool/v2"
	"sync"
)

func runExport(workers []*Worker, resources []*types.GroupResource) (int, error) {
	var wg sync.WaitGroup

	poolSize := len(resources)

	// create new pool
	pool := workerpool.New(poolSize)
	out := make(chan *types.GroupResource, poolSize)

	for _, w := range workers {
		if err := pool.AddWorker(w.function(&wg, out)); err != nil {
			return 0, err
		}
	}

	wg.Add(len(resources))

	for _, res := range resources {
		if err := pool.Delegate(res); err != nil {
			return 0, err
		}
	}

	// stop all workers after jobs are done
	wg.Wait()
	close(out)
	pool.Stop()
	workerErrors := 0
	for _, w := range workers {
		workerErrors += w.Stop()
	}
	return workerErrors, nil
}
