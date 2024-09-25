package nop

import "github.com/bakito/kubexporter/pkg/export/progress"

func NewProgress() progress.Progress {
	return &nilProgress{}
}

type nilProgress struct{}

func (n *nilProgress) Run() {
}

func (n *nilProgress) NewSearchBar(_ string, _ int, _ int) {
}

func (n *nilProgress) NewExportBar(_ string, _ int, _ int, _ int) {
}

func (n *nilProgress) Wait() {
}

func (n *nilProgress) Reset() {
}

func (n *nilProgress) NewWorker() progress.Progress {
	return n
}

func (n *nilProgress) IncrementMainBar() {
}

func (n *nilProgress) IncrementResourceBarBy(i int) {
}
