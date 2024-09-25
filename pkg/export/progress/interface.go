package progress

type Progress interface {
	NewSearchBar(currentKind string, pageSize int, currentPage int)
	NewExportBar(currentKind string, pageSize int, currentPage int, size int)
	Wait()
	Reset()
	NewWorker() Progress

	IncrementMainBar()
	IncrementResourceBarBy(i int)
}
