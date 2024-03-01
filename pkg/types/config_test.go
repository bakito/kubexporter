package types_test

import (
	"os"

	"github.com/bakito/kubexporter/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/ptr"
)

var _ = Describe("Config", func() {
	var (
		config *types.Config
		pf     *genericclioptions.PrintFlags
		res    *types.GroupResource
	)
	BeforeEach(func() {
		pf = &genericclioptions.PrintFlags{
			OutputFormat:       ptr.To(types.DefaultFormat),
			JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
		}
		config = types.NewConfig(nil, pf)
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

	Context("KindsByField", func() {
		var us unstructured.Unstructured
		BeforeEach(func() {
			config.Excluded = types.Excluded{
				KindsByField: map[string][]types.FieldValue{
					"group.kind": {
						types.FieldValue{
							Field:  []string{"metadata", "name"},
							Values: []string{"name1", "name2"},
						},
						types.FieldValue{
							Field:  []string{"metadata", "namespace"},
							Values: []string{"namespace1"},
						},
					},
				},
				Kinds: []string{"foo.Bar"},
			}
			us = unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "kind",
					"metadata": map[string]interface{}{
						"namespace": "namespace",
						"name":      "name",
					},
				},
			}
		})

		It("should not be excluded if no match", func() {
			Ω(config.IsInstanceExcluded(res, us)).Should(BeFalse())
		})

		It("should be excluded if name matches", func() {
			Ω(unstructured.SetNestedField(us.Object, "name1", "metadata", "name")).ShouldNot(HaveOccurred())
			Ω(config.IsInstanceExcluded(res, us)).Should(BeTrue())

			Ω(unstructured.SetNestedField(us.Object, "name2", "metadata", "name")).ShouldNot(HaveOccurred())
			Ω(config.IsInstanceExcluded(res, us)).Should(BeTrue())
		})

		It("should be excluded if namespace matches", func() {
			Ω(unstructured.SetNestedField(us.Object, "namespace1", "metadata", "namespace")).ShouldNot(HaveOccurred())
			Ω(config.IsInstanceExcluded(res, us)).Should(BeTrue())
		})

		Context("ConsiderOwnerReferences", func() {
			BeforeEach(func() {
				us.SetOwnerReferences([]v1.OwnerReference{{APIVersion: "foo/v1", Kind: "Bar"}})
			})
			It("if enabled it should be excluded if the owner is excluded", func() {
				config.ConsiderOwnerReferences = true
				Ω(config.IsInstanceExcluded(res, us)).Should(BeTrue())
			})
			It("if enabled it should not be excluded if the owner is not excluded", func() {
				us.SetOwnerReferences([]v1.OwnerReference{{APIVersion: "foofoo/v1", Kind: "Bar"}})
				config.ConsiderOwnerReferences = true
				Ω(config.IsInstanceExcluded(res, us)).Should(BeFalse())
			})
			It("if disabled it should be not excluded if the owner is excluded", func() {
				config.ConsiderOwnerReferences = false
				Ω(config.IsInstanceExcluded(res, us)).Should(BeFalse())
			})
		})
	})
	Context("FileName / ListFileName", func() {
		var (
			res *types.GroupResource
			sep = string(os.PathSeparator)
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
			var us *unstructured.Unstructured
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
				Ω(config.FileName(res, us, 0)).Should(Equal("namespace" + sep + "group.kind.name.yaml"))
			})
			It("should generate a file name with group and index", func() {
				Ω(config.FileName(res, us, 1)).Should(Equal("namespace" + sep + "group.kind.name_1.yaml"))
			})
			It("should generate a file name without group", func() {
				res.APIGroup = ""
				Ω(config.FileName(res, us, 0)).Should(Equal("namespace" + sep + "kind.name.yaml"))
			})
		})

		Context("FileName", func() {
			It("should generate a file name with group", func() {
				Ω(config.ListFileName(res, "namespace")).Should(Equal("namespace" + sep + "group.kind.yaml"))
			})
			It("should generate a file name without group", func() {
				res.APIGroup = ""
				Ω(config.ListFileName(res, "namespace")).Should(Equal("namespace" + sep + "kind.yaml"))
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
		It("quiet should switch progress and summary to false", func() {
			config.Quiet = true
			config.Progress = types.ProgressBar
			config.Summary = true
			err := config.Validate()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(config.Progress).Should(Equal(types.ProgressNone))
			Ω(config.Summary).Should(BeFalse())
		})
		It("should set progress default to bar", func() {
			config.Progress = ""
			err := config.Validate()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(config.Progress).Should(Equal(types.ProgressBar))
			config.Progress = "foo"
			err = config.Validate()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(config.Progress).Should(Equal(types.ProgressBar))
		})
	})

	Context("FilterFields", func() {
		var us unstructured.Unstructured
		BeforeEach(func() {
			config.Excluded = types.Excluded{
				Fields: [][]string{
					{"status"},
					{"metadata", "uid"},
					{"spec", "slice", "a"},
					{"spec", "slice", "b", "bb"},
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
					"spec": map[string]interface{}{
						"foo": "bar",
						"slice": []interface{}{
							map[string]interface{}{
								"a": "A",
								"b": map[string]interface{}{
									"ba": "BA",
									"bb": "BB",
								},
							},
						},
					},
					"status": "abc",
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
		It("should filter slice fields", func() {
			config.FilterFields(res, us)

			// slice support
			Ω(us.Object["spec"]).Should(HaveKey("foo"))
			Ω(us.Object["spec"]).Should(HaveKey("slice"))
			sl, _, err := unstructured.NestedSlice(us.Object, "spec", "slice")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(sl).Should(HaveLen(1))
			Ω(sl[0]).ShouldNot(HaveKey("a"))
			Ω(sl[0]).Should(HaveKey("b"))
			b, ok := sl[0].(map[string]interface{})["b"].(map[string]interface{})
			Ω(ok).Should(BeTrue())
			Ω(b).Should(HaveKey("ba"))
			Ω(b).ShouldNot(HaveKey("bb"))
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

	Context("MaskedFields", func() {
		var us unstructured.Unstructured
		BeforeEach(func() {
			config.Masked = &types.Masked{
				Replacement: "***",
				KindFields: map[string][][]string{
					"group.kind": {
						[]string{"data"},
						[]string{"status"},
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
					"data": map[string]interface{}{
						"a": "A",
						"b": "BB",
					},

					"status": "abc",
				},
			}
		})
		It("should mask status and all data fields", func() {
			Ω(config.Masked.Setup()).ShouldNot(HaveOccurred())
			config.MaskFields(res, us)
			Ω(us.Object["status"]).Should(Equal("***"))
			Ω(us.Object["data"]).Should(HaveKey("a"))
			Ω(us.Object["data"].(map[string]interface{})["a"]).Should(Equal("***"))
			Ω(us.Object["data"]).Should(HaveKey("b"))
			Ω(us.Object["data"].(map[string]interface{})["b"]).Should(Equal("***"))
		})
		It("should generate the md5 checksum of status and all data fields", func() {
			config.Masked.Checksum = "md5"
			Ω(config.Masked.Setup()).ShouldNot(HaveOccurred())
			config.MaskFields(res, us)
			Ω(us.Object["status"]).Should(Equal("900150983cd24fb0d6963f7d28e17f72"))
			Ω(us.Object["data"]).Should(HaveKey("a"))
			Ω(us.Object["data"].(map[string]interface{})["a"]).Should(Equal("7fc56270e7a70fa81a5935b72eacbe29"))
			Ω(us.Object["data"]).Should(HaveKey("b"))
			Ω(us.Object["data"].(map[string]interface{})["b"]).Should(Equal("9d3d9048db16a7eee539e93e3618cbe7"))
		})
		It("should generate the sha1 checksum of status and all data fields", func() {
			config.Masked.Checksum = "sha1"
			Ω(config.Masked.Setup()).ShouldNot(HaveOccurred())
			config.MaskFields(res, us)
			Ω(us.Object["status"]).Should(Equal("a9993e364706816aba3e25717850c26c9cd0d89d"))
			Ω(us.Object["data"]).Should(HaveKey("a"))
			Ω(us.Object["data"].(map[string]interface{})["a"]).
				Should(Equal("6dcd4ce23d88e2ee9568ba546c007c63d9131c1b"))
			Ω(us.Object["data"]).Should(HaveKey("b"))
			Ω(us.Object["data"].(map[string]interface{})["b"]).
				Should(Equal("71c9db717578b9ee49a59e69375c16c0627dffef"))
		})
		It("should generate the sha256 checksum of status and all data fields", func() {
			config.Masked.Checksum = "sha256"
			Ω(config.Masked.Setup()).ShouldNot(HaveOccurred())
			config.MaskFields(res, us)
			Ω(us.Object["status"]).Should(Equal("23097d223405d8228642a477bda255b32aadbce4bda0b3f7e36c9da7"))
			Ω(us.Object["data"]).Should(HaveKey("a"))
			Ω(us.Object["data"].(map[string]interface{})["a"]).
				Should(Equal("5cfe2cddbb9940fb4d8505e25ea77e763a0077693dbb01b1a6aa94f2"))
			Ω(us.Object["data"]).Should(HaveKey("b"))
			Ω(us.Object["data"].(map[string]interface{})["b"]).
				Should(Equal("a6eaa57c9e088b8466692680ab779768f4cf36622bc893aee163be9c"))
		})
		It("should fail with an unknown checksum", func() {
			config.Masked.Checksum = "foo"
			Ω(config.Masked.Setup()).Should(HaveOccurred())
		})
	})

	Context("SortSlices", func() {
		var us unstructured.Unstructured
		BeforeEach(func() {
			config.SortSlices = map[string][][]string{
				"group.kind": {},
			}
			us = unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":        "kind",
					"stringSlice": []interface{}{"C", "A", "B", "AA"},
					"intSlice":    []interface{}{int64(3), int64(1), int64(2), int64(4)},
					"floatSlice":  []interface{}{1.3, 1.1, 1.2, 1.4},
					"structSlice": []interface{}{map[string]interface{}{"field": "val2"}, map[string]interface{}{"field": "val1"}},
				},
			}
		})
		It("should sort the string slice", func() {
			config.SortSlices["group.kind"] = [][]string{{"stringSlice"}}
			config.SortSliceFields(res, us)
			Ω(us.Object["stringSlice"]).Should(Equal([]interface{}{"A", "AA", "B", "C"}))
		})
		It("should sort the int slice", func() {
			config.SortSlices["group.kind"] = [][]string{{"intSlice"}}
			config.SortSliceFields(res, us)
			Ω(us.Object["intSlice"]).Should(Equal([]interface{}{int64(1), int64(2), int64(3), int64(4)}))
		})
		It("should sort the float slice", func() {
			config.SortSlices["group.kind"] = [][]string{{"floatSlice"}}
			config.SortSliceFields(res, us)
			Ω(us.Object["floatSlice"]).Should(Equal([]interface{}{1.1, 1.2, 1.3, 1.4}))
		})
		It("should sort the struct slice", func() {
			config.SortSlices["group.kind"] = [][]string{{"structSlice"}}
			config.SortSliceFields(res, us)
			Ω(us.Object["structSlice"]).Should(Equal([]interface{}{map[string]interface{}{"field": "val1"}, map[string]interface{}{"field": "val2"}}))
		})
	})

	Context("read-config", func() {
		var cfg *types.Config
		BeforeEach(func() {
			cfg = types.NewConfig(nil, nil)
			err := types.UpdateFrom(cfg, "../../config.yaml")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should read Excluded.Kinds correctly", func() {
			Ω(cfg.Excluded.Kinds).Should(ContainElement("Pod"))
			Ω(cfg.Excluded.Kinds).Should(ContainElement("batch.Job"))
		})

		It("should read Excluded.Fields correctly", func() {
			Ω(cfg.Excluded.Fields).Should(ContainElement([]string{"status"}))
			Ω(cfg.Excluded.Fields).Should(ContainElement([]string{"metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration"}))
		})

		It("should read Excluded.KindsField correctly", func() {
			Ω(cfg.Excluded.KindFields).Should(HaveKey("Secret"))
			Ω(cfg.Excluded.KindFields["Secret"]).Should(ContainElement([]string{"metadata", "annotations", "openshift.io/token-secret.name"}))
			Ω(cfg.Excluded.KindFields["Secret"]).Should(ContainElement([]string{"metadata", "annotations", "openshift.io/token-secret.value"}))
		})

		It("should read Excluded.KindsByField correctly", func() {
			Ω(cfg.Excluded.KindsByField).Should(HaveKey("Secret"))
			Ω(cfg.Excluded.KindsByField["Secret"]).Should(HaveLen(1))
			Ω(cfg.Excluded.KindsByField["Secret"][0].Field).Should(Equal([]string{"type"}))
			Ω(cfg.Excluded.KindsByField["Secret"][0].Values).Should(Equal([]string{"helm.sh/release", "helm.sh/release.v1"}))
		})

		It("should read Masked.KindFields correctly", func() {
			Ω(cfg.Masked.KindFields).Should(HaveKey("Secret"))
			Ω(cfg.Masked.KindFields["Secret"]).Should(Equal([][]string{{"stringData"}}))
			Ω(cfg.Masked.Replacement).Should(Equal("***"))
			Ω(cfg.Masked.Checksum).Should(Equal("md5"))
		})

		It("should read Encrypted.KindFields correctly", func() {
			Ω(cfg.Encrypted.KindFields).Should(HaveKey("Secret"))
			Ω(cfg.Encrypted.KindFields["Secret"]).Should(Equal([][]string{{"data"}}))
			Ω(cfg.Encrypted.AesKey).Should(Equal("12345678901234567890123456789012"))
		})
	})

	Context("KindFields", func() {
		Context("Diff", func() {
			It("The diff should not contain fields covered by the source", func() {
				source := types.KindFields{
					"Secret": [][]string{{"data"}},
					"Pod":    [][]string{{"metadata", "labels", "foo"}},
				}
				other := types.KindFields{
					"Secret":     [][]string{{"data", "key"}},
					"Pod":        [][]string{{"metadata", "labels"}},
					"Deployment": [][]string{{"metadata", "annotations"}},
				}

				diff := source.Diff(other)
				Ω(diff).Should(HaveLen(2))
				Ω(diff["Pod"][0]).Should(Equal([]string{"metadata", "labels"}))
				Ω(diff["Deployment"][0]).Should(Equal([]string{"metadata", "annotations"}))
			})
		})
	})
})
