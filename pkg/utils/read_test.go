package utils_test

import (
	"bytes"
	"io"

	"github.com/bakito/kubexporter/pkg/types"
	"github.com/bakito/kubexporter/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/ptr"
)

var _ = Describe("Utils", func() {
	var pf *genericclioptions.PrintFlags
	BeforeEach(func() {
		pf = &genericclioptions.PrintFlags{
			OutputFormat:       ptr.To(types.DefaultFormat),
			JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
		}
	})

	Context("PrintObj", func() {
		var data *unstructured.Unstructured
		BeforeEach(func() {
			data = &unstructured.Unstructured{}
			data.SetUnstructuredContent(map[string]interface{}{
				"kind": "Pod",
				"foo":  "bar",
			})
		})
		It("should print the object as yaml", func() {
			var buf bytes.Buffer
			pf.OutputFormat = ptr.To("yaml")
			err := utils.PrintObj(pf, data, io.Writer(&buf))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(buf.String()).Should(Equal(`foo: bar
kind: Pod
`))
		})
		It("should print the object as json", func() {
			var buf bytes.Buffer
			pf.OutputFormat = ptr.To("json")

			err := utils.PrintObj(pf, data, io.Writer(&buf))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(buf.String()).Should(Equal(`{
    "foo": "bar",
    "kind": "Pod"
}
`))
		})
		It("should fail with unsupported format", func() {
			var buf bytes.Buffer
			pf.OutputFormat = ptr.To("xyz")
			err := utils.PrintObj(pf, data, io.Writer(&buf))
			Ω(err).Should(HaveOccurred())
		})
	})
})
