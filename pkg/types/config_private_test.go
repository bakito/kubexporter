package types

import (
	"reflect"
	"testing"
)

func TestConfig_normalizeNamespaces(t *testing.T) {
	tests := []struct {
		name       string
		namespace  *string
		namespaces []string
		expected   []string
	}{
		{
			name:       "should have no namespaces",
			namespace:  nil,
			namespaces: nil,
			expected:   EmptyNamespaces(),
		},
		{
			name:       "should sort and dedup namespaces",
			namespace:  nil,
			namespaces: []string{"ns2", "ns1", "ns2"},
			expected:   []string{"ns1", "ns2"},
		},
		{
			name:       "should add single namespace to namespaces",
			namespace:  new("ns1"),
			namespaces: []string{"ns2"},
			expected:   []string{"ns1", "ns2"},
		},
		{
			name:       "should sort and dedup namespace and namespaces",
			namespace:  new("ns1"),
			namespaces: []string{"ns2", "ns1"},
			expected:   []string{"ns1", "ns2"},
		},
		{
			name:       "should trim namespaces",
			namespace:  new(" ns1 "),
			namespaces: []string{"\tns2", "ns1  "},
			expected:   []string{"ns1", "ns2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Namespace:  tt.namespace,
				Namespaces: tt.namespaces,
			}
			cfg.normalizeNamespaces()

			if !reflect.DeepEqual(cfg.Namespaces, tt.expected) {
				t.Errorf("normalizeNamespaces() = %v, want %v", cfg.Namespaces, tt.expected)
			}
			if cfg.Namespace != nil {
				t.Errorf("Namespace must be set to nil, got %s", *cfg.Namespace)
			}
		})
	}
}
