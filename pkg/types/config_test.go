package types_test

import (
	"path/filepath"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/ptr"

	"github.com/bakito/kubexporter/pkg/types"
)

func setupConfig() (*types.Config, *genericclioptions.PrintFlags, *types.GroupResource) {
	pf := &genericclioptions.PrintFlags{
		OutputFormat:       ptr.To(types.DefaultFormat),
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
	}
	config := types.NewConfig(nil, pf)
	res := &types.GroupResource{
		APIGroup: "group",
		APIResource: metav1.APIResource{
			Kind: "kind",
		},
	}
	return config, pf, res
}

func TestConfig_IsExcluded(t *testing.T) {
	t.Run("should not be excluded if no includes and excludes", func(t *testing.T) {
		config, _, res := setupConfig()
		if config.IsExcluded(res) {
			t.Error("expected not excluded")
		}
	})
	t.Run("should not be excluded if kind is included", func(t *testing.T) {
		config, _, res := setupConfig()
		config.Included.Kinds = []string{"group.kind"}
		if config.IsExcluded(res) {
			t.Error("expected not excluded")
		}
	})
	t.Run("should be excluded if kind is not in included", func(t *testing.T) {
		config, _, res := setupConfig()
		config.Included.Kinds = []string{"group.kind2"}
		if !config.IsExcluded(res) {
			t.Error("expected excluded")
		}
	})
	t.Run("should be excluded if kind is excluded", func(t *testing.T) {
		config, _, res := setupConfig()
		config.Excluded.Kinds = []string{"group.kind"}
		if !config.IsExcluded(res) {
			t.Error("expected excluded")
		}
	})
	t.Run("should not be excluded if kind is not excluded", func(t *testing.T) {
		config, _, res := setupConfig()
		config.Excluded.Kinds = []string{"group.kind2"}
		if config.IsExcluded(res) {
			t.Error("expected not excluded")
		}
	})
}

func TestConfig_IsInstanceExcluded(t *testing.T) {
	config, _, res := setupConfig()
	config.Excluded = types.Excluded{
		KindsByField: map[string][]types.FieldValue{
			"group.kind": {
				{
					Field:  []string{"metadata", "name"},
					Values: []string{"name1", "name2"},
				},
				{
					Field:  []string{"metadata", "namespace"},
					Values: []string{"namespace1"},
				},
			},
		},
		Kinds: []string{"foo.Bar"},
	}
	us := unstructured.Unstructured{
		Object: map[string]any{
			"kind": "kind",
			"metadata": map[string]any{
				"namespace": "namespace",
				"name":      "name",
			},
		},
	}

	t.Run("should not be excluded if no match", func(t *testing.T) {
		if config.IsInstanceExcluded(res, us) {
			t.Error("expected not excluded")
		}
	})

	t.Run("should be excluded if name matches", func(t *testing.T) {
		c := us.DeepCopy()
		_ = unstructured.SetNestedField(c.Object, "name1", "metadata", "name")
		if !config.IsInstanceExcluded(res, *c) {
			t.Error("expected excluded")
		}

		_ = unstructured.SetNestedField(c.Object, "name2", "metadata", "name")
		if !config.IsInstanceExcluded(res, *c) {
			t.Error("expected excluded")
		}
	})

	t.Run("should be excluded if namespace matches", func(t *testing.T) {
		c := us.DeepCopy()
		_ = unstructured.SetNestedField(c.Object, "namespace1", "metadata", "namespace")
		if !config.IsInstanceExcluded(res, *c) {
			t.Error("expected excluded")
		}
	})

	t.Run("ConsiderOwnerReferences", func(t *testing.T) {
		t.Run("if enabled it should be excluded if the owner is excluded", func(t *testing.T) {
			c := us.DeepCopy()
			c.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "foo/v1", Kind: "Bar"}})
			config.ConsiderOwnerReferences = true
			if !config.IsInstanceExcluded(res, *c) {
				t.Error("expected excluded")
			}
		})
		t.Run("if enabled it should not be excluded if the owner is not excluded", func(t *testing.T) {
			c := us.DeepCopy()
			c.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "foofoo/v1", Kind: "Bar"}})
			config.ConsiderOwnerReferences = true
			if config.IsInstanceExcluded(res, *c) {
				t.Error("expected not excluded")
			}
		})
		t.Run("if disabled it should be not excluded if the owner is excluded", func(t *testing.T) {
			c := us.DeepCopy()
			c.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "foo/v1", Kind: "Bar"}})
			config.ConsiderOwnerReferences = false
			if config.IsInstanceExcluded(res, *c) {
				t.Error("expected not excluded")
			}
		})
	})
}

func TestConfig_FileName(t *testing.T) {
	config, _, res := setupConfig()
	us := &unstructured.Unstructured{
		Object: map[string]any{
			"kind": "Kind",
			"metadata": map[string]any{
				"namespace": "namespace",
				"name":      "name",
			},
		},
	}

	t.Run("should generate a file name with group", func(t *testing.T) {
		got, _ := config.FileName(res, us, 0)
		expected := filepath.Join("namespace", "group.kind.name.yaml")
		if got != expected {
			t.Errorf("expected %q, but got %q", expected, got)
		}
	})
	t.Run("should generate a file name with group and index", func(t *testing.T) {
		got, _ := config.FileName(res, us, 1)
		expected := filepath.Join("namespace", "group.kind.name_1.yaml")
		if got != expected {
			t.Errorf("expected %q, but got %q", expected, got)
		}
	})
	t.Run("should generate a file name without group", func(t *testing.T) {
		r := *res
		r.APIGroup = ""
		got, _ := config.FileName(&r, us, 0)
		expected := filepath.Join("namespace", "kind.name.yaml")
		if got != expected {
			t.Errorf("expected %q, but got %q", expected, got)
		}
	})

	t.Run("ListFileName", func(t *testing.T) {
		t.Run("should generate a file name with group", func(t *testing.T) {
			got, _ := config.ListFileName(res, "namespace")
			expected := filepath.Join("namespace", "group.kind.yaml")
			if got != expected {
				t.Errorf("expected %q, but got %q", expected, got)
			}
		})
		t.Run("should generate a file name without group", func(t *testing.T) {
			r := *res
			r.APIGroup = ""
			got, _ := config.ListFileName(&r, "namespace")
			expected := filepath.Join("namespace", "kind.yaml")
			if got != expected {
				t.Errorf("expected %q, but got %q", expected, got)
			}
		})
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("should be valid", func(t *testing.T) {
		config, _, _ := setupConfig()
		if err := config.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("should have invalid workers", func(t *testing.T) {
		config, _, _ := setupConfig()
		config.Worker = 0
		err := config.Validate()
		if err == nil {
			t.Error("expected error")
		} else if err.Error() != "worker must be > 0" {
			t.Errorf("expected \"worker must be > 0\", but got %q", err.Error())
		}
	})
	t.Run("should have invalid file template", func(t *testing.T) {
		config, _, _ := setupConfig()
		config.FileNameTemplate = ""
		err := config.Validate()
		if err == nil {
			t.Error("expected error")
		} else if err.Error() != "file name template must not be empty" {
			t.Errorf("expected \"file name template must not be empty\", but got %q", err.Error())
		}
	})
	t.Run("should have not parsable file template", func(t *testing.T) {
		config, _, _ := setupConfig()
		config.FileNameTemplate = "{{dsfa"
		err := config.Validate()
		if err == nil {
			t.Error("expected error")
		} else if err.Error() != "error parsing file name template [{{dsfa]" {
			t.Errorf("expected \"error parsing file name template [{{dsfa]\", but got %q", err.Error())
		}
	})
	t.Run("should have invalid list file template", func(t *testing.T) {
		config, _, _ := setupConfig()
		config.ListFileNameTemplate = ""
		err := config.Validate()
		if err == nil {
			t.Error("expected error")
		} else if err.Error() != "list file name template must not be empty" {
			t.Errorf("expected \"list file name template must not be empty\", but got %q", err.Error())
		}
	})
	t.Run("should have not parsable list file template", func(t *testing.T) {
		config, _, _ := setupConfig()
		config.ListFileNameTemplate = "{{dsfa"
		err := config.Validate()
		if err == nil {
			t.Error("expected error")
		} else if err.Error() != "error parsing list file name template [{{dsfa]" {
			t.Errorf("expected \"error parsing list file name template [{{dsfa]\", but got %q", err.Error())
		}
	})
	t.Run("quiet should switch progress and summary to false", func(t *testing.T) {
		config, _, _ := setupConfig()
		config.Quiet = true
		config.Progress = types.ProgressBar
		config.Summary = true
		err := config.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if config.Progress != types.ProgressNone {
			t.Errorf("expected ProgressNone, but got %v", config.Progress)
		}
		if config.Summary {
			t.Error("expected Summary to be false")
		}
	})
	t.Run("should set progress default to bar", func(t *testing.T) {
		config, _, _ := setupConfig()
		config.Progress = ""
		err := config.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if config.Progress != types.ProgressBar {
			t.Errorf("expected ProgressBar, but got %v", config.Progress)
		}
		config.Progress = "foo"
		err = config.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if config.Progress != types.ProgressBar {
			t.Errorf("expected ProgressBar, but got %v", config.Progress)
		}
	})
}

func TestConfig_FilterFields(t *testing.T) {
	config, _, res := setupConfig()
	config.Excluded = types.Excluded{
		Fields: [][]string{
			{"status"},
			{"metadata", "uid"},
			{"spec", "slice", "a"},
			{"spec", "slice", "b", "bb"},
		},
		KindFields: map[string][][]string{
			"group.kind2": {
				{"metadata", "revision"},
			},
		},
	}
	us := unstructured.Unstructured{
		Object: map[string]any{
			"kind": "kind",
			"metadata": map[string]any{
				"name":     "name",
				"uid":      "uid",
				"revision": "revision",
			},
			"spec": map[string]any{
				"foo": "bar",
				"slice": []any{
					map[string]any{
						"a": "A",
						"b": map[string]any{
							"ba": "BA",
							"bb": "BB",
						},
					},
				},
			},
			"status": "abc",
		},
	}

	t.Run("should filter default fields", func(t *testing.T) {
		c := us.DeepCopy()
		config.FilterFields(res, *c)
		metadata := c.Object["metadata"].(map[string]any)
		if _, ok := metadata["name"]; !ok {
			t.Error("expected name to be present")
		}
		if _, ok := metadata["revision"]; !ok {
			t.Error("expected revision to be present")
		}
		if _, ok := metadata["uid"]; ok {
			t.Error("expected uid to be removed")
		}
		if _, ok := c.Object["status"]; ok {
			t.Error("expected status to be removed")
		}
	})

	t.Run("should filter slice fields", func(t *testing.T) {
		c := us.DeepCopy()
		config.FilterFields(res, *c)
		spec := c.Object["spec"].(map[string]any)
		if spec["foo"] != "bar" {
			t.Errorf("expected foo=bar, but got %v", spec["foo"])
		}
		sl, _, _ := unstructured.NestedSlice(c.Object, "spec", "slice")
		if len(sl) != 1 {
			t.Fatalf("expected slice length 1, but got %d", len(sl))
		}
		item := sl[0].(map[string]any)
		if _, ok := item["a"]; ok {
			t.Error("expected a to be removed")
		}
		b := item["b"].(map[string]any)
		if b["ba"] != "BA" {
			t.Errorf("expected ba=BA, but got %v", b["ba"])
		}
		if _, ok := b["bb"]; ok {
			t.Error("expected bb to be removed")
		}
	})

	t.Run("should filter default fields and kindFields", func(t *testing.T) {
		res2 := &types.GroupResource{
			APIGroup: "group",
			APIResource: metav1.APIResource{
				Kind: "kind2",
			},
		}
		c := us.DeepCopy()
		config.FilterFields(res2, *c)
		metadata := c.Object["metadata"].(map[string]any)
		if _, ok := metadata["name"]; !ok {
			t.Error("expected name to be present")
		}
		if _, ok := metadata["revision"]; ok {
			t.Error("expected revision to be removed")
		}
		if _, ok := metadata["uid"]; ok {
			t.Error("expected uid to be removed")
		}
		if _, ok := c.Object["status"]; ok {
			t.Error("expected status to be removed")
		}
	})
}

func TestConfig_MaskFields(t *testing.T) {
	config, _, res := setupConfig()
	config.Masked = &types.Masked{
		Replacement: "***",
		KindFields: map[string][][]string{
			"group.kind": {
				{"data"},
				{"status"},
			},
		},
	}
	us := unstructured.Unstructured{
		Object: map[string]any{
			"kind": "kind",
			"metadata": map[string]any{
				"name":     "name",
				"uid":      "uid",
				"revision": "revision",
			},
			"data": map[string]any{
				"a": "A",
				"b": "BB",
			},
			"status": "abc",
		},
	}

	t.Run("should mask status and all data fields", func(t *testing.T) {
		c := us.DeepCopy()
		_ = config.Masked.Setup()
		config.MaskFields(res, *c)
		if c.Object["status"] != "***" {
			t.Errorf("expected status=***, but got %v", c.Object["status"])
		}
		data := c.Object["data"].(map[string]any)
		if data["a"] != "***" {
			t.Errorf("expected data.a=***, but got %v", data["a"])
		}
		if data["b"] != "***" {
			t.Errorf("expected data.b=***, but got %v", data["b"])
		}
	})

	t.Run("should generate the md5 checksum of status and all data fields", func(t *testing.T) {
		c := us.DeepCopy()
		config.Masked.Checksum = "md5"
		_ = config.Masked.Setup()
		config.MaskFields(res, *c)
		if c.Object["status"] != "900150983cd24fb0d6963f7d28e17f72" {
			t.Errorf("expected md5 hash, but got %v", c.Object["status"])
		}
	})

	t.Run("should generate the sha1 checksum of status and all data fields", func(t *testing.T) {
		c := us.DeepCopy()
		config.Masked.Checksum = "sha1"
		_ = config.Masked.Setup()
		config.MaskFields(res, *c)
		if c.Object["status"] != "a9993e364706816aba3e25717850c26c9cd0d89d" {
			t.Errorf("expected sha1 hash, but got %v", c.Object["status"])
		}
	})

	t.Run("should generate the sha256 checksum of status and all data fields", func(t *testing.T) {
		c := us.DeepCopy()
		config.Masked.Checksum = "sha256"
		_ = config.Masked.Setup()
		config.MaskFields(res, *c)
		if c.Object["status"] != "23097d223405d8228642a477bda255b32aadbce4bda0b3f7e36c9da7" {
			t.Errorf("expected sha256 hash, but got %v", c.Object["status"])
		}
	})

	t.Run("should fail with an unknown checksum", func(t *testing.T) {
		config.Masked.Checksum = "foo"
		err := config.Masked.Setup()
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestConfig_SortSliceFields(t *testing.T) {
	config, _, res := setupConfig()
	us := unstructured.Unstructured{
		Object: map[string]any{
			"kind":        "kind",
			"stringSlice": []any{"C", "A", "B", "AA"},
			"intSlice":    []any{int64(3), int64(1), int64(2), int64(4)},
			"floatSlice":  []any{1.3, 1.1, 1.2, 1.4},
			"structSlice": []any{
				map[string]any{"field": "val2"},
				map[string]any{"field": "val1"},
			},
		},
	}

	t.Run("should sort the string slice", func(t *testing.T) {
		c := us.DeepCopy()
		config.SortSlices = map[string][][]string{"group.kind": {{"stringSlice"}}}
		config.SortSliceFields(res, *c)
		expected := []any{"A", "AA", "B", "C"}
		if !reflect.DeepEqual(c.Object["stringSlice"], expected) {
			t.Errorf("expected %v, but got %v", expected, c.Object["stringSlice"])
		}
	})

	t.Run("should sort the int slice", func(t *testing.T) {
		c := us.DeepCopy()
		config.SortSlices = map[string][][]string{"group.kind": {{"intSlice"}}}
		config.SortSliceFields(res, *c)
		expected := []any{int64(1), int64(2), int64(3), int64(4)}
		if !reflect.DeepEqual(c.Object["intSlice"], expected) {
			t.Errorf("expected %v, but got %v", expected, c.Object["intSlice"])
		}
	})

	t.Run("should sort the float slice", func(t *testing.T) {
		c := us.DeepCopy()
		config.SortSlices = map[string][][]string{"group.kind": {{"floatSlice"}}}
		config.SortSliceFields(res, *c)
		expected := []any{1.1, 1.2, 1.3, 1.4}
		if !reflect.DeepEqual(c.Object["floatSlice"], expected) {
			t.Errorf("expected %v, but got %v", expected, c.Object["floatSlice"])
		}
	})

	t.Run("should sort the struct slice", func(t *testing.T) {
		c := us.DeepCopy()
		config.SortSlices = map[string][][]string{"group.kind": {{"structSlice"}}}
		config.SortSliceFields(res, *c)
		expected := []any{map[string]any{"field": "val1"}, map[string]any{"field": "val2"}}
		if !reflect.DeepEqual(c.Object["structSlice"], expected) {
			t.Errorf("expected %v, but got %v", expected, c.Object["structSlice"])
		}
	})
}

func TestConfig_ReadConfig(t *testing.T) {
	cfg := types.NewConfig(nil, nil)
	err := types.UpdateFrom(cfg, "../../config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("should read Excluded.Kinds correctly", func(t *testing.T) {
		foundPod := false
		foundJob := false
		for _, k := range cfg.Excluded.Kinds {
			if k == "Pod" {
				foundPod = true
			}
			if k == "batch.Job" {
				foundJob = true
			}
		}
		if !foundPod || !foundJob {
			t.Errorf("expected Pod and batch.Job in Excluded.Kinds, but got %v", cfg.Excluded.Kinds)
		}
	})

	t.Run("should read Excluded.Fields correctly", func(t *testing.T) {
		foundStatus := false
		for _, f := range cfg.Excluded.Fields {
			if reflect.DeepEqual(f, []string{"status"}) {
				foundStatus = true
				break
			}
		}
		if !foundStatus {
			t.Error("expected status in Excluded.Fields")
		}
	})
}

func TestKindFields(t *testing.T) {
	t.Run("Diff", func(t *testing.T) {
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
		if len(diff) != 2 {
			t.Errorf("expected diff length 2, but got %d", len(diff))
		}
		if !reflect.DeepEqual(diff["Pod"][0], []string{"metadata", "labels"}) {
			t.Errorf("expected Pod diff [metadata labels], but got %v", diff["Pod"][0])
		}
	})
	t.Run("String", func(t *testing.T) {
		kf := types.KindFields{
			"Secret":     [][]string{{"data", "key"}},
			"Pod":        [][]string{{"metadata", "labels"}},
			"Deployment": [][]string{{"metadata", "annotations"}},
		}
		expected := "Deployment: [[metadata,annotations]], Pod: [[metadata,labels]], Secret: [[data,key]]"
		if kf.String() != expected {
			t.Errorf("expected %q, but got %q", expected, kf.String())
		}
	})
}

func TestConfig_PreservedFields(t *testing.T) {
	config, _, res := setupConfig()

	t.Run("should preserve specified fields when excluding status", func(t *testing.T) {
		config.Excluded = types.Excluded{
			Fields: [][]string{
				{"status"},
			},
			PreservedFields: types.PreservedFields{
				Fields: [][]string{
					{"status", "loadBalancer", "ingress"},
					{"status", "conditions"},
				},
			},
		}

		us := unstructured.Unstructured{
			Object: map[string]any{
				"kind": "kind",
				"metadata": map[string]any{
					"name": "test-resource",
				},
				"status": map[string]any{
					"phase": "Running",
					"conditions": []any{
						map[string]any{
							"type":   "Ready",
							"status": "True",
						},
					},
					"loadBalancer": map[string]any{
						"ingress": []any{
							map[string]any{
								"ip": "192.168.1.100",
							},
						},
						"other": "should-be-removed",
					},
					"other": "should-be-removed",
				},
			},
		}

		config.FilterFields(res, us)

		if _, ok := us.Object["status"]; !ok {
			t.Fatal("expected status to be present")
		}
		status := us.Object["status"].(map[string]any)
		if _, ok := status["loadBalancer"]; !ok {
			t.Error("expected loadBalancer to be present")
		}
		if _, ok := status["conditions"]; !ok {
			t.Error("expected conditions to be present")
		}
		if _, ok := status["phase"]; ok {
			t.Error("expected phase to be removed")
		}
	})
}
