package nop

import "github.com/bakito/kubexporter/pkg/export/progress"

func NewProgress() progress.Progress {
	return &nilProgress{}
}

type nilProgress struct{}

func (n *nilProgress) Run() error {
	return nil
}

func (n *nilProgress) NewSearchBar(_ progress.Step) {
}

func (n *nilProgress) NewExportBar(_ progress.Step) {
}

func (n *nilProgress) Reset() {
}

func (n *nilProgress) NewWorker() progress.Progress {
	return n
}

func (n *nilProgress) IncrementMainBar() {
}

func (n *nilProgress) IncrementResourceBarBy(_ int, _ int) {
}
