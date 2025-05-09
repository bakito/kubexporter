package types

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
	Error             string
	QueryDuration     time.Duration
	ExportDuration    time.Duration
}

// Report generate report rows.
func (r GroupResource) Report(withError, withPages bool) []string {
	row := []string{
		r.APIGroup,
		r.APIVersion,
		r.APIResource.Kind,
		strconv.FormatBool(r.APIResource.Namespaced),
		strconv.Itoa(r.Instances),
		strconv.Itoa(r.ExportedInstances),
		r.QueryDuration.String(),
	}
	if withPages {
		row = append(row, strconv.Itoa(r.Pages))
	}
	row = append(row, r.ExportDuration.String())
	if withError {
		row = append(row, r.Error)
	}
	return row
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

// Sort GroupResource.
func Sort(resources []*GroupResource) func(int, int) bool {
	return func(i, j int) bool {
		ret := strings.Compare(resources[i].APIGroup, resources[j].APIGroup)
		if ret > 0 {
			return false
		} else if ret == 0 {
			return resources[i].APIResource.Kind < resources[j].APIResource.Kind
		}
		return true
	}
}
