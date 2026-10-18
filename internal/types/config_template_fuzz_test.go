package types

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func FuzzConfigFileNameTemplate(f *testing.F) {
	f.Add([]byte(`{{.Namespace}}/{{.Kind}}.{{.Name}}.{{.Extension}}`))
	f.Add([]byte(`{{default "_cluster_" .Namespace}}/{{.Kind}}.{{.Name}}.{{.Extension}}`))
	f.Add([]byte(`{{if .Group}}{{printf "%s." .Group }}{{end}}{{.Kind}}.{{.Name}}.{{.Extension}}`))
	f.Add([]byte(`invalid template {{`))
	f.Add([]byte(`{{.NonExistent}}`))
	f.Add([]byte(`{{.Namespace | upper}}`))
	f.Add([]byte(`{{.Name | nospace}}`))
	f.Add([]byte(``))
	f.Add([]byte(`{{.Namespace | repeat 1000000}}`))

	printFlags := genericclioptions.NewPrintFlags("yaml")
	format := "yaml"
	printFlags.OutputFormat = &format

	f.Fuzz(func(_ *testing.T, templ []byte) {
		config := NewConfig(nil, printFlags)
		config.FileNameTemplate = string(templ)
		config.ListFileNameTemplate = string(templ)

		res := &GroupResource{}
		us := &unstructured.Unstructured{}
		_, _ = config.FileName(res, us, 0)
	})
}

func FuzzMaskedSetup(f *testing.F) {
	f.Add([]byte("md5"))
	f.Add([]byte("sha1"))
	f.Add([]byte("sha256"))
	f.Add([]byte(""))
	f.Add([]byte("invalid"))
	f.Add([]byte("SHA256"))
	f.Add([]byte("md5\n"))
	f.Add([]byte("\x00\x01\x02"))

	f.Fuzz(func(_ *testing.T, checksum []byte) {
		m := &Masked{
			Checksum: string(checksum),
		}
		_ = m.Setup()
	})
}
