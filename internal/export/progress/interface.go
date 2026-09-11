package progress

type Progress interface {
	Async() bool
	NewSearchBar(step Step)
	NewExportBar(step Step)
	Run() error
	Reset()
	NewWorker() Progress
	// Finish is called when the export is done. It must make sure all progress bars are completed (100%).
	Finish()

	IncrementMainBar()
	IncrementResourceBarBy(id, inc int)
}

type Step struct {
	WorkerID    int
	CurrentKind string
	PageSize    int
	CurrentPage int
	Total       int
}
