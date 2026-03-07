package worker

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	gm "go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	amtypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/ptr"

	"github.com/bakito/kubexporter/pkg/client"
	"github.com/bakito/kubexporter/pkg/export/progress/nop"
	mock "github.com/bakito/kubexporter/pkg/mocks/client"
	"github.com/bakito/kubexporter/pkg/types"
)

func setupWorker(t *testing.T) (*worker, string) {
	t.Helper()
	tmpDir := t.TempDir()
	mockCtrl := gm.NewController(t)
	mockClient := mock.NewMockInterface(mockCtrl)
	config := types.NewConfig(nil, &genericclioptions.PrintFlags{
		OutputFormat:       ptr.To(types.DefaultFormat),
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
	})
	config.Target = tmpDir

	w := &worker{
		config: config,
		ac:     &client.APIClient{Client: mockClient},
		prog:   nop.NewProgress(),
	}
	return w, tmpDir
}

func getTestData() (*types.GroupResource, *unstructured.UnstructuredList) {
	res := &types.GroupResource{
		APIGroup:        "",
		APIVersion:      "v1",
		APIGroupVersion: "v1",
		APIResource: metav1.APIResource{
			Kind:       "Deployment",
			Namespaced: true,
		},
	}

	dl := &appsv1.DeploymentList{
		TypeMeta: metav1.TypeMeta{
			Kind: "DeploymentList",
		},
		Items: []appsv1.Deployment{
			deployment(1, 1),
			deployment(1, 2),
			deployment(2, 1),
		},
	}

	ulc, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(dl)
	ul := &unstructured.UnstructuredList{}
	ul.SetUnstructuredContent(ulc)
	return res, ul
}

func TestWorker_exportLists(t *testing.T) {
	tests := []struct {
		name     string
		res      *types.GroupResource
		ul       *unstructured.UnstructuredList
		validate func(t *testing.T, tmpDir string)
	}{
		{
			name: "should do nothing with nil args",
			res:  nil,
			ul:   nil,
		},
		{
			name: "should create two dirs with one file each",
			res: func() *types.GroupResource {
				res, _ := getTestData()
				return res
			}(),
			ul: func() *unstructured.UnstructuredList {
				_, ul := getTestData()
				return ul
			}(),
			validate: func(t *testing.T, tmpDir string) {
				t.Helper()
				dirs := checkDir(t, 2, tmpDir)
				if dirs[0].Name() != "namespace-1" {
					t.Errorf("expected namespace-1, but got %s", dirs[0].Name())
				}
				if dirs[1].Name() != "namespace-2" {
					t.Errorf("expected namespace-2, but got %s", dirs[1].Name())
				}

				ns1 := checkDir(t, 1, tmpDir, dirs[0].Name())
				if ns1[0].Name() != "Deployment.yaml" {
					t.Errorf("expected Deployment.yaml, but got %s", ns1[0].Name())
				}
				l1 := unstructuredListFrom(t, tmpDir, dirs[0].Name(), ns1[0].Name())
				if len(l1.Items) != 2 {
					t.Errorf("expected 2 items, but got %d", len(l1.Items))
				}
				checkDeployment(t, 1, 1, &l1.Items[0])
				checkDeployment(t, 1, 2, &l1.Items[1])

				ns2 := checkDir(t, 1, tmpDir, dirs[1].Name())
				if ns2[0].Name() != "Deployment.yaml" {
					t.Errorf("expected Deployment.yaml, but got %s", ns2[0].Name())
				}
				l2 := unstructuredListFrom(t, tmpDir, dirs[1].Name(), ns2[0].Name())
				if len(l2.Items) != 1 {
					t.Errorf("expected 1 item, but got %d", len(l2.Items))
				}
				checkDeployment(t, 2, 1, &l2.Items[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, tmpDir := setupWorker(t)
			w.exportLists(tt.res, tt.ul)
			if tt.validate != nil {
				tt.validate(t, tmpDir)
			}
		})
	}
}

func TestWorker_exportSingleResources(t *testing.T) {
	tests := []struct {
		name     string
		res      *types.GroupResource
		ul       *unstructured.UnstructuredList
		validate func(t *testing.T, tmpDir string)
	}{
		{
			name: "should do nothing with nil args",
			res:  nil,
			ul:   nil,
		},
		{
			name: "should create two dirs, one with one file, one with two files each",
			res: func() *types.GroupResource {
				res, _ := getTestData()
				return res
			}(),
			ul: func() *unstructured.UnstructuredList {
				_, ul := getTestData()
				return ul
			}(),
			validate: func(t *testing.T, tmpDir string) {
				t.Helper()
				dirs := checkDir(t, 2, tmpDir)

				if dirs[0].Name() != "namespace-1" {
					t.Errorf("expected namespace-1, but got %s", dirs[0].Name())
				}
				if dirs[1].Name() != "namespace-2" {
					t.Errorf("expected namespace-2, but got %s", dirs[1].Name())
				}

				ns1 := checkDir(t, 2, tmpDir, dirs[0].Name())
				if ns1[0].Name() != "Deployment.deployment-1.yaml" {
					t.Errorf("expected Deployment.deployment-1.yaml, but got %s", ns1[0].Name())
				}
				if ns1[1].Name() != "Deployment.deployment-2.yaml" {
					t.Errorf("expected Deployment.deployment-2.yaml, but got %s", ns1[1].Name())
				}
				d11 := unstructuredFrom(t, tmpDir, dirs[0].Name(), ns1[0].Name())
				checkDeployment(t, 1, 1, d11)
				d12 := unstructuredFrom(t, tmpDir, dirs[0].Name(), ns1[1].Name())
				checkDeployment(t, 1, 2, d12)

				ns2 := checkDir(t, 1, tmpDir, dirs[1].Name())
				if ns2[0].Name() != "Deployment.deployment-1.yaml" {
					t.Errorf("expected Deployment.deployment-1.yaml, but got %s", ns2[0].Name())
				}
				d21 := unstructuredFrom(t, tmpDir, dirs[1].Name(), ns2[0].Name())
				checkDeployment(t, 2, 1, d21)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, tmpDir := setupWorker(t)
			w.exportSingleResources(tt.res, tt.ul)
			if tt.validate != nil {
				tt.validate(t, tmpDir)
			}
		})
	}
}

func checkDir(t *testing.T, expectedFiles int, dir ...string) []os.DirEntry {
	t.Helper()
	files, err := os.ReadDir(filepath.Join(dir...))
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}
	if len(files) != expectedFiles {
		t.Fatalf("expected %d files, but got %d", expectedFiles, len(files))
	}
	return files
}

func checkDeployment(t *testing.T, n, d int, u *unstructured.Unstructured) {
	t.Helper()
	if u.GetNamespace() != fmt.Sprintf("namespace-%d", n) {
		t.Errorf("expected namespace-%d, but got %s", n, u.GetNamespace())
	}
	if u.GetName() != fmt.Sprintf("deployment-%d", d) {
		t.Errorf("expected deployment-%d, but got %s", d, u.GetName())
	}
	if _, ok := u.Object["status"]; ok {
		t.Error("expected status to be removed")
	}
	metadata, ok := u.Object["metadata"].(map[string]any)
	if !ok {
		t.Fatal("expected metadata to be a map")
	}
	if _, ok := metadata["uid"]; ok {
		t.Error("expected uid to be removed")
	}
}

func deployment(n, d int) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: fmt.Sprintf("namespace-%d", n),
			Name:      fmt.Sprintf("deployment-%d", d),
			UID:       amtypes.UID(uuid.New().String()),
		},
		Status: appsv1.DeploymentStatus{
			Replicas: 1,
		},
	}
}

func unstructuredListFrom(t *testing.T, path ...string) *unstructured.UnstructuredList {
	t.Helper()
	ul := &unstructured.UnstructuredList{}
	b, err := os.ReadFile(filepath.Join(path...))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	err = yaml.Unmarshal(b, ul)
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}
	return ul
}

func unstructuredFrom(t *testing.T, path ...string) *unstructured.Unstructured {
	t.Helper()
	u := &unstructured.Unstructured{}
	b, err := os.ReadFile(filepath.Join(path...))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	err = yaml.Unmarshal(b, u)
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}
	return u
}
