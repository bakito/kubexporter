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
	pf := &genericclioptions.PrintFlags{
		OutputFormat:       ptr.To(types.DefaultFormat),
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
	}

	data := &unstructured.Unstructured{}
	data.SetUnstructuredContent(map[string]any{
		"kind": "Pod",
		"foo":  "bar",
	})

	t.Run("should print the object as yaml", func(t *testing.T) {
		var buf bytes.Buffer
		pf.OutputFormat = new("yaml")
		err := utils.PrintObj(pf, data, io.Writer(&buf))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		expected := "foo: bar\nkind: Pod\n"
		if buf.String() != expected {
			t.Errorf("expected %q, but got %q", expected, buf.String())
		}
	})

	t.Run("should print the object as json", func(t *testing.T) {
		var buf bytes.Buffer
		pf.OutputFormat = new("json")
		err := utils.PrintObj(pf, data, io.Writer(&buf))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		expected := "{\n    \"foo\": \"bar\",\n    \"kind\": \"Pod\"\n}\n"
		if buf.String() != expected {
			t.Errorf("expected %q, but got %q", expected, buf.String())
		}
	})

	t.Run("should fail with unsupported format", func(t *testing.T) {
		var buf bytes.Buffer
		pf.OutputFormat = new("xyz")
		err := utils.PrintObj(pf, data, io.Writer(&buf))
		if err == nil {
			t.Error("expected error")
		}
	})
}
