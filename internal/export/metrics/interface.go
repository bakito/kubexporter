package metrics

import (
	"time"

	"github.com/bakito/kubexporter/internal/export/worker"
	"github.com/bakito/kubexporter/internal/log"
	"github.com/bakito/kubexporter/internal/types"
)

// Provider metrics provider interface.
type Provider interface {
	Stats() *worker.Stats
	Config() *types.Config
	Start() time.Time
	Logger() log.YALI
	ClusterHost() string
}
