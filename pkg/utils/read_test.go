package utils_test

import (
	"bytes"
	"io"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/ptr"

	"github.com/bakito/kubexporter/pkg/types"
	"github.com/bakito/kubexporter/pkg/utils"
)

func TestPrintObj(t *testing.T) {
	data := &unstructured.Unstructured{}
	data.SetUnstructuredContent(map[string]any{
		"kind": "Pod",
		"foo":  "bar",
	})

	tests := []struct {
		name     string
		format   string
		expected string
		wantErr  bool
	}{
		{
			name:     "should print the object as yaml",
			format:   "yaml",
			expected: "foo: bar\nkind: Pod\n",
		},
		{
			name:     "should print the object as json",
			format:   "json",
			expected: "{\n    \"foo\": \"bar\",\n    \"kind\": \"Pod\"\n}\n",
		},
		{
			name:    "should fail with unsupported format",
			format:  "xyz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := &genericclioptions.PrintFlags{
				OutputFormat:       new(tt.format),
				JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
			}
			if tt.format == "" {
				pf.OutputFormat = ptr.To(types.DefaultFormat)
			}

			var buf bytes.Buffer
			err := utils.PrintObj(pf, data, io.Writer(&buf))
			if (err != nil) != tt.wantErr {
				t.Errorf("PrintObj() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && buf.String() != tt.expected {
				t.Errorf("PrintObj() = %q, want %q", buf.String(), tt.expected)
			}
		})
	}
}
