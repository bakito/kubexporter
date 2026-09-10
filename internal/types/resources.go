package types

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GroupResource group resource information.
type GroupResource struct {
	APIGroup          string
	APIGroupVersion   string
	APIResource       metav1.APIResource
	APIVersion        string
	Instances         int
	ExportedInstances int
	Pages             int
	ExportedSize      int64
	Error             string
	QueryDuration     time.Duration
	ExportDuration    time.Duration
}

// Report generates report rows.
func (r GroupResource) Report(withSize, withError, withPages bool) []string {
	namespaced := ""
	if r.APIResource.Namespaced {
		namespaced = "yes"
	}
	row := []string{
		r.APIGroup,
		r.APIVersion,
		r.APIResource.Kind,
		namespaced,
		strconv.Itoa(r.Instances),
		strconv.Itoa(r.ExportedInstances),
	}
	if withSize {
		row = append(row, humanize.Bytes(uint64(r.ExportedSize)))
	}
	row = append(row, FormatDuration(r.QueryDuration))
	if withPages {
		row = append(row, strconv.Itoa(r.Pages))
	}
	row = append(row, FormatDuration(r.ExportDuration))
	if withError {
		row = append(row, r.Error)
	}
	return row
}

// FormatDuration formats a duration in a short, human friendly way.
func FormatDuration(d time.Duration) string {
	switch {
	case d == 0:
		return "0s"
	case d < time.Millisecond:
		return d.Round(time.Microsecond).String()
	case d < time.Second:
		return d.Round(time.Millisecond).String()
	default:
		return d.Round(10 * time.Millisecond).String()
	}
}

// GroupKind get concatenated group and kind.
func (r GroupResource) GroupKind() string {
	if r.APIGroup != "" {
		return fmt.Sprintf("%s.%s", r.APIGroup, r.APIResource.Kind)
	}
	return r.APIResource.Kind
}

// Kind get the kind.
func (r GroupResource) Kind() string {
	return r.APIResource.Kind
}
