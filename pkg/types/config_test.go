package types_test

import (
	"github.com/bakito/kubexporter/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("Config", func() {
	var (
		config *types.Config
		res    *types.GroupResource
	)
	BeforeEach(func() {
		config = &types.Config{
			FileNameTemplate:     types.DefaultFileNameTemplate,
			ListFileNameTemplate: types.DefaultListFileNameTemplate,
			OutputFormat:         types.DefaultFormat,
			Target:               "target",
			Worker:               1,
		}
		res = &types.GroupResource{
			APIGroup: "group",
			APIResource: v1.APIResource{
				Kind: "kind",
			},
		}
	})

	Context("IsExcluded", func() {
		It("should not be excluded if no includes and excludes", func() {
			Ω(config.IsExcluded(res)).Should(BeFalse())
		})
		It("should not be excluded if kind is included", func() {
			config.Included.Kinds = []string{"group.kind"}
			Ω(config.IsExcluded(res)).Should(BeFalse())
		})
		It("should be excluded if kind is not in included", func() {
			config.Included.Kinds = []string{"group.kind2"}
			Ω(config.IsExcluded(res)).Should(BeTrue())
		})
		It("should be excluded if kind is excluded", func() {
			config.Excluded.Kinds = []string{"group.kind"}
			Ω(config.IsExcluded(res)).Should(BeTrue())
		})
		It("should not be excluded if kind is not excluded", func() {
			config.Excluded.Kinds = []string{"group.kind2"}
			Ω(config.IsExcluded(res)).Should(BeFalse())
		})
	})

	Context("FileName / ListFileName", func() {
		var (
			res *types.GroupResource
		)
		BeforeEach(func() {
			res = &types.GroupResource{
				APIGroup: "group",
				APIResource: v1.APIResource{
					Kind: "kind",
				},
			}
		})

		Context("FileName", func() {
			var (
				us *unstructured.Unstructured
			)
			BeforeEach(func() {
				us = &unstructured.Unstructured{
					Object: map[string]interface{}{
						"kind": "Kind",
						"metadata": map[string]interface{}{
							"namespace": "namespace",
							"name":      "name",
						},
					},
				}
			})

			It("should generate a file name with group", func() {
				Ω(config.FileName(res, us)).Should(Equal("namespace/group.kind.name.yaml"))
			})
			It("should generate a file name without group", func() {
				res.APIGroup = ""
				Ω(config.FileName(res, us)).Should(Equal("namespace/kind.name.yaml"))
			})
		})

		Context("FileName", func() {
			It("should generate a file name with group", func() {
				Ω(config.ListFileName(res, "namespace")).Should(Equal("namespace/group.kind.yaml"))
			})
			It("should generate a file name without group", func() {
				res.APIGroup = ""
				Ω(config.ListFileName(res, "namespace")).Should(Equal("namespace/kind.yaml"))
			})
		})
	})

	Context("Validate", func() {
		It("should be valid", func() {
			Ω(config.Validate()).ShouldNot(HaveOccurred())
		})
		It("should have invalid workers", func() {
			config.Worker = 0
			err := config.Validate()
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("worker must be > 0"))
		})
		It("should have invalid output format", func() {
			config.OutputFormat = "foo"
			err := config.Validate()
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("unsupported output format [foo]"))
		})
		It("should have invalid file template", func() {
			config.FileNameTemplate = ""
			err := config.Validate()
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("file name template must not be empty"))
		})
		It("should have not parsable file template", func() {
			config.FileNameTemplate = "{{dsfa"
			err := config.Validate()
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("error parsing file name template [{{dsfa]"))
		})
		It("should have invalid list file template", func() {
			config.ListFileNameTemplate = ""
			err := config.Validate()
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("list file name template must not be empty"))
		})
		It("should have not parsable list file template", func() {
			config.ListFileNameTemplate = "{{dsfa"
			err := config.Validate()
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("error parsing list file name template [{{dsfa]"))
		})
		It("quiet should swithc progress and summary to false", func() {
			config.Quiet = true
			config.Progress = true
			config.Summary = true
			err := config.Validate()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(config.Progress).Should(BeFalse())
			Ω(config.Summary).Should(BeFalse())
		})
	})

	Context("FilterFields", func() {
		var (
			us unstructured.Unstructured
		)
		BeforeEach(func() {
			config.Excluded = types.Excluded{
				Fields: [][]string{
					{"status"},
					{"metadata", "uid"},
				},
				KindFields: map[string][][]string{
					"group.kind2": {
						[]string{"metadata", "revision"},
					},
				},
			}
			us = unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "kind",
					"metadata": map[string]interface{}{
						"name":     "name",
						"uid":      "uid",
						"revision": "revision",
					},
					"status": map[string]interface{}{
						"foo": "bar",
					},
				},
			}
		})
		It("should filter default fields", func() {
			config.FilterFields(res, us)
			Ω(us.Object).Should(HaveKey("metadata"))
			Ω(us.Object["metadata"]).Should(HaveKey("name"))
			Ω(us.Object["metadata"]).Should(HaveKey("revision"))
			Ω(us.Object["metadata"]).ShouldNot(HaveKey("uid"))
			Ω(us.Object).ShouldNot(HaveKey("status"))
		})
		It("should filter default fields and kindFields", func() {
			res.APIResource.Kind = "kind2"
			config.FilterFields(res, us)
			Ω(us.Object).Should(HaveKey("metadata"))
			Ω(us.Object["metadata"]).Should(HaveKey("name"))
			Ω(us.Object["metadata"]).ShouldNot(HaveKey("revision"))
			Ω(us.Object["metadata"]).ShouldNot(HaveKey("uid"))
			Ω(us.Object).ShouldNot(HaveKey("status"))
		})
	})

	Context("Marshal", func() {
		var (
			data map[string]string
		)
		BeforeEach(func() {
			data = map[string]string{
				"foo": "bar",
			}
		})
		It("should marshal as yaml", func() {
			config.OutputFormat = "yaml"
			b, err := config.Marshal(data)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(b)).Should(Equal(`foo: bar
`))
		})
		It("should marshal as json", func() {
			config.OutputFormat = "json"
			b, err := config.Marshal(data)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(b)).Should(Equal(`{
  "foo": "bar"
}
`))
		})
		It("should fail with unsupported format", func() {
			config.OutputFormat = "xyz"
			_, err := config.Marshal(data)
			Ω(err).Should(HaveOccurred())
		})
	})
})
