package types_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bakito/kubexporter/pkg/types"
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
