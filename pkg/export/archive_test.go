package export

import (
	"testing"
	"time"

	"github.com/bakito/kubexporter/pkg/types"
)

func TestArchiveName(t *testing.T) {
	tests := []struct {
		name     string
		config   *types.Config
		expected string
	}{
		{
			name: "all namespaces",
			config: &types.Config{
				Target: "/exports/cluster",
			},
			expected: "cluster-2026-06-26-080205.tar.gz",
		},
		{
			name: "single namespace",
			config: &types.Config{
				Namespaces: []string{"ns1"},
				Target:     "/exports/cluster",
			},
			expected: "cluster-ns1-2026-06-26-080205.tar.gz",
		},
		{
			name: "multiple namespaces",
			config: &types.Config{
				Namespaces: []string{"ns1", "ns2"},
				Target:     "/exports/cluster",
			},
			expected: "cluster-1b2e9198-2026-06-26-080205.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ex := &exporter{
				config: tt.config,
			}

			// Generate the archive name
			archiveName := ex.archiveName(time.Date(2026, 6, 26, 8, 2, 5, 0, time.UTC))

			// Assert it matches the expected name
			if tt.expected != archiveName {
				t.Errorf("archiveName() = %s, want %s", archiveName, tt.expected)
			}
		})
	}
}
