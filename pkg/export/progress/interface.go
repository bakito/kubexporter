package progress

type Progress interface {
	Async() bool
	NewSearchBar(step Step)
	NewExportBar(step Step)
	Run() error
	Reset()
	NewWorker() Progress

	IncrementMainBar()
	IncrementResourceBarBy(id int, inc int)
}

type Step struct {
	WorkerID    int
	CurrentKind string
	PageSize    int
	CurrentPage int
	Total       int
}
