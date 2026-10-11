package types_test

import (
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bakito/kubexporter/internal/types"
)

func TestGroupResource_GroupKind(t *testing.T) {
	tests := []struct {
		name     string
		res      *types.GroupResource
		expected string
	}{
		{
			name: "kind only",
			res: &types.GroupResource{
				APIResource: metav1.APIResource{
					Kind: "kind",
				},
			},
			expected: "kind",
		},
		{
			name: "group and kind",
			res: &types.GroupResource{
				APIGroup: "group",
				APIResource: metav1.APIResource{
					Kind: "kind",
				},
			},
			expected: "group.kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.res.GroupKind(); got != tt.expected {
				t.Errorf("GroupResource.GroupKind() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "microseconds",
			duration: 500 * time.Microsecond,
			expected: "500µs",
		},
		{
			name:     "milliseconds",
			duration: 150 * time.Millisecond,
			expected: "150ms",
		},
		{
			name:     "seconds",
			duration: 1*time.Second + 500*time.Millisecond,
			expected: "1.5s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := types.FormatDuration(tt.duration); got != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.duration, got, tt.expected)
			}
		})
	}
}

func TestGroupResource_Report(t *testing.T) {
	res := &types.GroupResource{
		APIGroup:          "apps",
		APIVersion:        "v1",
		APIResource:       metav1.APIResource{Kind: "Deployment", Namespaced: true},
		Instances:         10,
		ExportedInstances: 8,
		ExportedSize:      1024,
		Pages:             2,
		Error:             "timeout",
		QueryDuration:     100 * time.Millisecond,
		ExportDuration:    200 * time.Millisecond,
	}

	t.Run("with all flags", func(t *testing.T) {
		row := res.Report(true, true, true)
		// [APIGroup, APIVersion, Kind, namespaced, Instances, ExportedInstances, Size, QueryDuration, Pages, ExportDuration, Error]
		if len(row) != 11 {
			t.Fatalf("expected 11 columns, got %d", len(row))
		}
		if row[0] != "apps" {
			t.Errorf("expected APIGroup 'apps', got %q", row[0])
		}
		if row[3] != "yes" {
			t.Errorf("expected namespaced 'yes', got %q", row[3])
		}
		if row[10] != "timeout" {
			t.Errorf("expected error 'timeout', got %q", row[10])
		}
	})

	t.Run("without size", func(t *testing.T) {
		row := res.Report(false, false, false)
		// [APIGroup, APIVersion, Kind, namespaced, Instances, ExportedInstances, QueryDuration, ExportDuration]
		if len(row) != 8 {
			t.Fatalf("expected 8 columns, got %d", len(row))
		}
	})

	t.Run("without pages", func(t *testing.T) {
		row := res.Report(true, false, false)
		// [APIGroup, APIVersion, Kind, namespaced, Instances, ExportedInstances, Size, QueryDuration, ExportDuration]
		if len(row) != 9 {
			t.Fatalf("expected 9 columns, got %d", len(row))
		}
	})

	t.Run("without error", func(t *testing.T) {
		row := res.Report(false, false, true)
		// [APIGroup, APIVersion, Kind, namespaced, Instances, ExportedInstances, QueryDuration, Pages, ExportDuration]
		if len(row) != 9 {
			t.Fatalf("expected 9 columns, got %d", len(row))
		}
	})

	t.Run("cluster-scoped resource", func(t *testing.T) {
		res2 := &types.GroupResource{
			APIResource: metav1.APIResource{Kind: "Namespace", Namespaced: false},
		}
		row := res2.Report(false, false, false)
		if row[3] != "" {
			t.Errorf("expected empty namespaced for cluster-scoped, got %q", row[3])
		}
	})
}

func TestGroupResource_Kind(t *testing.T) {
	res := &types.GroupResource{
		APIResource: metav1.APIResource{Kind: "Deployment"},
	}
	if got := res.Kind(); got != "Deployment" {
		t.Errorf("Kind() = %q, want %q", got, "Deployment")
	}
}

func TestGroupResource_Report_contains_duration(t *testing.T) {
	res := &types.GroupResource{
		APIResource:    metav1.APIResource{Kind: "Pod"},
		QueryDuration:  500 * time.Microsecond,
		ExportDuration: 1 * time.Second,
	}
	row := res.Report(false, false, false)
	// QueryDuration should be at index 6, ExportDuration at index 7
	if !strings.Contains(row[6], "µs") {
		t.Errorf("expected microseconds in query duration, got %q", row[6])
	}
	if !strings.Contains(row[7], "s") {
		t.Errorf("expected seconds in export duration, got %q", row[7])
	}
}
