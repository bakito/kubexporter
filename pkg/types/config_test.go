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

func setupConfig() (*types.Config, *types.GroupResource) {
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
	return config, res
}

func TestConfig_IsExcluded(t *testing.T) {
	tests := []struct {
		name     string
		included []string
		excluded []string
		expected bool
	}{
		{
			name:     "should not be excluded if no includes and excludes",
			expected: false,
		},
		{
			name:     "should not be excluded if kind is included",
			included: []string{"group.kind"},
			expected: false,
		},
		{
			name:     "should be excluded if kind is not in included",
			included: []string{"group.kind2"},
			expected: true,
		},
		{
			name:     "should be excluded if kind is excluded",
			excluded: []string{"group.kind"},
			expected: true,
		},
		{
			name:     "should not be excluded if kind is not excluded",
			excluded: []string{"group.kind2"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, res := setupConfig()
			config.Included.Kinds = tt.included
			config.Excluded.Kinds = tt.excluded
			if got := config.IsExcluded(res); got != tt.expected {
				t.Errorf("IsExcluded() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_IsInstanceExcluded(t *testing.T) {
	config, res := setupConfig()
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

	tests := []struct {
		name                    string
		setup                   func(u *unstructured.Unstructured)
		considerOwnerReferences bool
		expected                bool
	}{
		{
			name:     "should not be excluded if no match",
			expected: false,
		},
		{
			name: "should be excluded if name1 matches",
			setup: func(u *unstructured.Unstructured) {
				_ = unstructured.SetNestedField(u.Object, "name1", "metadata", "name")
			},
			expected: true,
		},
		{
			name: "should be excluded if name2 matches",
			setup: func(u *unstructured.Unstructured) {
				_ = unstructured.SetNestedField(u.Object, "name2", "metadata", "name")
			},
			expected: true,
		},
		{
			name: "should be excluded if namespace matches",
			setup: func(u *unstructured.Unstructured) {
				_ = unstructured.SetNestedField(u.Object, "namespace1", "metadata", "namespace")
			},
			expected: true,
		},
		{
			name: "if enabled it should be excluded if the owner is excluded",
			setup: func(u *unstructured.Unstructured) {
				u.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "foo/v1", Kind: "Bar"}})
			},
			considerOwnerReferences: true,
			expected:                true,
		},
		{
			name: "if enabled it should not be excluded if the owner is not excluded",
			setup: func(u *unstructured.Unstructured) {
				u.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "foofoo/v1", Kind: "Bar"}})
			},
			considerOwnerReferences: true,
			expected:                false,
		},
		{
			name: "if disabled it should be not excluded if the owner is excluded",
			setup: func(u *unstructured.Unstructured) {
				u.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "foo/v1", Kind: "Bar"}})
			},
			considerOwnerReferences: false,
			expected:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := us.DeepCopy()
			if tt.setup != nil {
				tt.setup(c)
			}
			config.ConsiderOwnerReferences = tt.considerOwnerReferences
			if got := config.IsInstanceExcluded(res, *c); got != tt.expected {
				t.Errorf("IsInstanceExcluded() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_FileName(t *testing.T) {
	config, res := setupConfig()
	us := &unstructured.Unstructured{
		Object: map[string]any{
			"kind": "Kind",
			"metadata": map[string]any{
				"namespace": "namespace",
				"name":      "name",
			},
		},
	}

	tests := []struct {
		name     string
		index    int
		group    string
		expected string
	}{
		{
			name:     "should generate a file name with group",
			index:    0,
			group:    "group",
			expected: filepath.Join("namespace", "group.kind.name.yaml"),
		},
		{
			name:     "should generate a file name with group and index",
			index:    1,
			group:    "group",
			expected: filepath.Join("namespace", "group.kind.name_1.yaml"),
		},
		{
			name:     "should generate a file name without group",
			index:    0,
			group:    "",
			expected: filepath.Join("namespace", "kind.name.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := *res
			r.APIGroup = tt.group
			got, _ := config.FileName(&r, us, tt.index)
			if got != tt.expected {
				t.Errorf("FileName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConfig_ListFileName(t *testing.T) {
	config, res := setupConfig()

	tests := []struct {
		name     string
		group    string
		expected string
	}{
		{
			name:     "should generate a file name with group",
			group:    "group",
			expected: filepath.Join("namespace", "group.kind.yaml"),
		},
		{
			name:     "should generate a file name without group",
			group:    "",
			expected: filepath.Join("namespace", "kind.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := *res
			r.APIGroup = tt.group
			got, _ := config.ListFileName(&r, "namespace")
			if got != tt.expected {
				t.Errorf("ListFileName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c *types.Config)
		wantErr  bool
		errStr   string
		validate func(t *testing.T, c *types.Config)
	}{
		{
			name:    "should be valid",
			wantErr: false,
		},
		{
			name: "should have invalid workers",
			setup: func(c *types.Config) {
				c.Worker = 0
			},
			wantErr: true,
			errStr:  "worker must be > 0",
		},
		{
			name: "should have invalid file template",
			setup: func(c *types.Config) {
				c.FileNameTemplate = ""
			},
			wantErr: true,
			errStr:  "file name template must not be empty",
		},
		{
			name: "should have not parsable file template",
			setup: func(c *types.Config) {
				c.FileNameTemplate = "{{dsfa"
			},
			wantErr: true,
			errStr:  "error parsing file name template [{{dsfa]",
		},
		{
			name: "should have invalid list file template",
			setup: func(c *types.Config) {
				c.ListFileNameTemplate = ""
			},
			wantErr: true,
			errStr:  "list file name template must not be empty",
		},
		{
			name: "should have not parsable list file template",
			setup: func(c *types.Config) {
				c.ListFileNameTemplate = "{{dsfa"
			},
			wantErr: true,
			errStr:  "error parsing list file name template [{{dsfa]",
		},
		{
			name: "quiet should switch progress and summary to false",
			setup: func(c *types.Config) {
				c.Quiet = true
				c.Progress = types.ProgressBar
				c.Summary = true
			},
			wantErr: false,
			validate: func(t *testing.T, c *types.Config) {
				t.Helper()
				if c.Progress != types.ProgressNone {
					t.Errorf("expected ProgressNone, but got %v", c.Progress)
				}
				if c.Summary {
					t.Error("expected Summary to be false")
				}
			},
		},
		{
			name: "should set progress default to bar when empty",
			setup: func(c *types.Config) {
				c.Progress = ""
			},
			wantErr: false,
			validate: func(t *testing.T, c *types.Config) {
				t.Helper()
				if c.Progress != types.ProgressBar {
					t.Errorf("expected ProgressBar, but got %v", c.Progress)
				}
			},
		},
		{
			name: "should set progress default to bar when invalid",
			setup: func(c *types.Config) {
				c.Progress = "foo"
			},
			wantErr: false,
			validate: func(t *testing.T, c *types.Config) {
				t.Helper()
				if c.Progress != types.ProgressBar {
					t.Errorf("expected ProgressBar, but got %v", c.Progress)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, _ := setupConfig()
			if tt.setup != nil {
				tt.setup(config)
			}
			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errStr {
				t.Errorf("Validate() error = %q, wantErrStr %q", err.Error(), tt.errStr)
			}
			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestConfig_FilterFields(t *testing.T) {
	config, res := setupConfig()
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

	tests := []struct {
		name     string
		res      *types.GroupResource
		validate func(t *testing.T, u *unstructured.Unstructured)
	}{
		{
			name: "should filter default fields",
			res:  res,
			validate: func(t *testing.T, u *unstructured.Unstructured) {
				t.Helper()
				metadata, ok := u.Object["metadata"].(map[string]any)
				if !ok {
					t.Fatal("expected metadata to be a map")
				}
				if _, ok := metadata["name"]; !ok {
					t.Error("expected name to be present")
				}
				if _, ok := metadata["revision"]; !ok {
					t.Error("expected revision to be present")
				}
				if _, ok := metadata["uid"]; ok {
					t.Error("expected uid to be removed")
				}
				if _, ok := u.Object["status"]; ok {
					t.Error("expected status to be removed")
				}
			},
		},
		{
			name: "should filter slice fields",
			res:  res,
			validate: func(t *testing.T, u *unstructured.Unstructured) {
				t.Helper()
				spec, ok := u.Object["spec"].(map[string]any)
				if !ok {
					t.Fatal("expected spec to be a map")
				}
				if spec["foo"] != "bar" {
					t.Errorf("expected foo=bar, but got %v", spec["foo"])
				}
				sl, _, _ := unstructured.NestedSlice(u.Object, "spec", "slice")
				if len(sl) != 1 {
					t.Fatalf("expected slice length 1, but got %d", len(sl))
				}
				item, ok := sl[0].(map[string]any)
				if !ok {
					t.Fatal("expected item to be a map")
				}
				if _, ok := item["a"]; ok {
					t.Error("expected a to be removed")
				}
				b, ok := item["b"].(map[string]any)
				if !ok {
					t.Fatal("expected b to be a map")
				}
				if b["ba"] != "BA" {
					t.Errorf("expected ba=BA, but got %v", b["ba"])
				}
				if _, ok := b["bb"]; ok {
					t.Error("expected bb to be removed")
				}
			},
		},
		{
			name: "should filter default fields and kindFields",
			res: &types.GroupResource{
				APIGroup: "group",
				APIResource: metav1.APIResource{
					Kind: "kind2",
				},
			},
			validate: func(t *testing.T, u *unstructured.Unstructured) {
				t.Helper()
				metadata, ok := u.Object["metadata"].(map[string]any)
				if !ok {
					t.Fatal("expected metadata to be a map")
				}
				if _, ok := metadata["name"]; !ok {
					t.Error("expected name to be present")
				}
				if _, ok := metadata["revision"]; ok {
					t.Error("expected revision to be removed")
				}
				if _, ok := metadata["uid"]; ok {
					t.Error("expected uid to be removed")
				}
				if _, ok := u.Object["status"]; ok {
					t.Error("expected status to be removed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := us.DeepCopy()
			config.FilterFields(tt.res, *c)
			tt.validate(t, c)
		})
	}
}

func TestConfig_MaskFields(t *testing.T) {
	config, res := setupConfig()
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

	tests := []struct {
		name     string
		checksum string
		validate func(t *testing.T, u *unstructured.Unstructured)
	}{
		{
			name: "should mask status and all data fields",
			validate: func(t *testing.T, u *unstructured.Unstructured) {
				t.Helper()
				if u.Object["status"] != "***" {
					t.Errorf("expected status=***, but got %v", u.Object["status"])
				}
				data, ok := u.Object["data"].(map[string]any)
				if !ok {
					t.Fatal("expected data to be a map")
				}
				if data["a"] != "***" {
					t.Errorf("expected data.a=***, but got %v", data["a"])
				}
				if data["b"] != "***" {
					t.Errorf("expected data.b=***, but got %v", data["b"])
				}
			},
		},
		{
			name:     "should generate the md5 checksum",
			checksum: "md5",
			validate: func(t *testing.T, u *unstructured.Unstructured) {
				t.Helper()
				if u.Object["status"] != "900150983cd24fb0d6963f7d28e17f72" {
					t.Errorf("expected md5 hash, but got %v", u.Object["status"])
				}
			},
		},
		{
			name:     "should generate the sha1 checksum",
			checksum: "sha1",
			validate: func(t *testing.T, u *unstructured.Unstructured) {
				t.Helper()
				if u.Object["status"] != "a9993e364706816aba3e25717850c26c9cd0d89d" {
					t.Errorf("expected sha1 hash, but got %v", u.Object["status"])
				}
			},
		},
		{
			name:     "should generate the sha256 checksum",
			checksum: "sha256",
			validate: func(t *testing.T, u *unstructured.Unstructured) {
				t.Helper()
				if u.Object["status"] != "23097d223405d8228642a477bda255b32aadbce4bda0b3f7e36c9da7" {
					t.Errorf("expected sha256 hash, but got %v", u.Object["status"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := us.DeepCopy()
			config.Masked.Checksum = tt.checksum
			_ = config.Masked.Setup()
			config.MaskFields(res, *c)
			tt.validate(t, c)
		})
	}

	t.Run("should fail with an unknown checksum", func(t *testing.T) {
		config.Masked.Checksum = "foo"
		err := config.Masked.Setup()
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestConfig_SortSliceFields(t *testing.T) {
	config, res := setupConfig()
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

	tests := []struct {
		name     string
		field    string
		expected []any
	}{
		{
			name:     "should sort the string slice",
			field:    "stringSlice",
			expected: []any{"A", "AA", "B", "C"},
		},
		{
			name:     "should sort the int slice",
			field:    "intSlice",
			expected: []any{int64(1), int64(2), int64(3), int64(4)},
		},
		{
			name:     "should sort the float slice",
			field:    "floatSlice",
			expected: []any{1.1, 1.2, 1.3, 1.4},
		},
		{
			name:     "should sort the struct slice",
			field:    "structSlice",
			expected: []any{map[string]any{"field": "val1"}, map[string]any{"field": "val2"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := us.DeepCopy()
			config.SortSlices = map[string][][]string{"group.kind": {{tt.field}}}
			config.SortSliceFields(res, *c)
			if !reflect.DeepEqual(c.Object[tt.field], tt.expected) {
				t.Errorf("expected %v, but got %v", tt.expected, c.Object[tt.field])
			}
		})
	}
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
	config, res := setupConfig()

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
		status, ok := us.Object["status"].(map[string]any)
		if !ok {
			t.Fatal("expected status to be a map")
		}
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
