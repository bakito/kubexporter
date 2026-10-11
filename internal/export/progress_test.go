package export

import (
	"fmt"
	"testing"

	"github.com/bakito/kubexporter/internal/export/progress/nop"
	"github.com/bakito/kubexporter/internal/types"
)

func TestNewProgress_QuietIsAlwaysNop(t *testing.T) {
	for _, p := range []types.Progress{
		types.ProgressBar,
		types.ProgressBarBubbles,
		types.ProgressSimple,
		types.ProgressNone,
		"",
	} {
		t.Run(string(p), func(t *testing.T) {
			config := types.NewConfig(nil, nil)
			config.Quiet = true
			config.Progress = p

			prog := newProgress(config, []*types.GroupResource{})
			if fmt.Sprintf("%T", prog) != fmt.Sprintf("%T", nop.NewProgress()) {
				t.Errorf("expected nop progress in quiet mode for %q, but got %T", p, prog)
			}
		})
	}
}
