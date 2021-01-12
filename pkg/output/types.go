package output

import "github.com/bakito/kubexporter/pkg/log"

type Output interface {
	PrintStats(log.YALI)
	Do(log.YALI) error
}
