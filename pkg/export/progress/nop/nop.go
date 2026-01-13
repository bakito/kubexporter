package nop

import "github.com/bakito/kubexporter/pkg/export/progress"

func NewProgress() progress.Progress {
	return &nilProgress{}
}

type nilProgress struct{}

func (*nilProgress) Async() bool {
	return false
}

func (*nilProgress) Run() error {
	return nil
}

func (*nilProgress) NewSearchBar(_ progress.Step) {
}

func (*nilProgress) NewExportBar(_ progress.Step) {
}

func (*nilProgress) Reset() {
}

func (n *nilProgress) NewWorker() progress.Progress {
	return n
}

func (*nilProgress) IncrementMainBar() {
}

func (*nilProgress) IncrementResourceBarBy(_, _ int) {
}
