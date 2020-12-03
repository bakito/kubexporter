package main

import (
	"github.com/bakito/kubexporter/pkg/types"
	"sync"

	"github.com/vardius/worker-pool/v2"
)

func main() {
	var wg sync.WaitGroup

	poolSize := 1
	jobsAmount := 3
	workersAmount := 2

	// create new pool
	pool := workerpool.New(poolSize)
	out := make(chan *types.GroupResource, jobsAmount)
	worker := func(res *types.GroupResource) {
		defer wg.Done()
		out <- res
	}

	for i := 1; i <= workersAmount; i++ {
		if err := pool.AddWorker(worker); err != nil {
			panic(err)
		}
	}

	wg.Add(jobsAmount)

	for i := 0; i < jobsAmount; i++ {
		if err := pool.Delegate(&types.GroupResource{}); err != nil {
			panic(err)
		}
	}

	go func() {
		// stop all workers after jobs are done
		wg.Wait()
		close(out)
		pool.Stop() // stop removes all workers from pool, to resume work add them again
	}()

}
