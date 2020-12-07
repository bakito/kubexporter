package worker

import (
	"fmt"
	mockdynamic "github.com/bakito/kubexporter/pkg/mocks/client"
	"github.com/bakito/kubexporter/pkg/types"
	"github.com/ghodss/yaml"
	gm "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	amtypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
	"os"
	"path/filepath"
)

var _ = Describe("Worker", func() {
	var (
		w          *worker
		mockCtrl   *gm.Controller
		mockClient *mockdynamic.MockInterface
		config     *types.Config
		res        *types.GroupResource
		ul         *unstructured.UnstructuredList
		tmpDir     string
	)
	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("", "ginkgo-convert")
		Ω(err).ShouldNot(HaveOccurred())
		mockCtrl = gm.NewController(GinkgoT())
		mockClient = mockdynamic.NewMockInterface(mockCtrl)
		config = types.NewConfig(nil, &genericclioptions.PrintFlags{
			OutputFormat:       pointer.StringPtr(types.DefaultFormat),
			JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
		})
		config.Target = tmpDir
		res = &types.GroupResource{
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

		ulc, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dl)
		ul = &unstructured.UnstructuredList{}
		ul.SetUnstructuredContent(ulc)

		w = &worker{
			config: config,
			client: mockClient,
		}
	})
	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
	})

	Context("exportLists", func() {
		It("should do noting with nil args", func() {
			w.exportLists(nil, nil)

		})
		It("should create two dirs with one file each", func() {
			w.exportLists(res, ul)
			dirs := checkDir(2, tmpDir)
			Ω(dirs[0].Name()).Should(Equal("namespace-1"))
			Ω(dirs[1].Name()).Should(Equal("namespace-2"))

			ns1 := checkDir(1, tmpDir, dirs[0].Name())
			Ω(ns1[0].Name()).Should(Equal("Deployment.yaml"))
			l1 := unstructuredListFrom(tmpDir, dirs[0].Name(), ns1[0].Name())
			Ω(l1.Items).Should(HaveLen(2))
			checkDeployment(1, 1, &l1.Items[0])
			checkDeployment(1, 2, &l1.Items[1])

			ns2 := checkDir(1, tmpDir, dirs[1].Name())
			Ω(ns2[0].Name()).Should(Equal("Deployment.yaml"))
			l2 := unstructuredListFrom(tmpDir, dirs[1].Name(), ns1[0].Name())
			Ω(l2.Items).Should(HaveLen(1))
			checkDeployment(2, 1, &l2.Items[0])
		})
	})

	Context("exportSingleResources", func() {
		It("should do noting with nil args", func() {
			w.exportSingleResources(nil, nil)
		})
		It("should create two dirs, one with one file, one with two file each", func() {
			w.exportSingleResources(res, ul)
			dirs := checkDir(2, tmpDir)

			Ω(dirs[0].Name()).Should(Equal("namespace-1"))
			Ω(dirs[1].Name()).Should(Equal("namespace-2"))

			ns1 := checkDir(2, tmpDir, dirs[0].Name())
			Ω(ns1[0].Name()).Should(Equal("Deployment.deployment-1.yaml"))
			Ω(ns1[1].Name()).Should(Equal("Deployment.deployment-2.yaml"))
			d11 := unstructuredFrom(tmpDir, dirs[0].Name(), ns1[0].Name())
			checkDeployment(1, 1, d11)
			d12 := unstructuredFrom(tmpDir, dirs[0].Name(), ns1[1].Name())
			checkDeployment(1, 2, d12)

			ns2 := checkDir(1, tmpDir, dirs[1].Name())
			Ω(ns2[0].Name()).Should(Equal("Deployment.deployment-1.yaml"))
			d21 := unstructuredFrom(tmpDir, dirs[1].Name(), ns2[0].Name())
			checkDeployment(2, 1, d21)
		})
	})
})

func checkDir(expectedFiles int, dir ...string) []os.FileInfo {
	files, err := ioutil.ReadDir(filepath.Join(dir...))
	Ω(err).ShouldNot(HaveOccurred())
	Ω(files).Should(HaveLen(expectedFiles))
	return files
}

func checkDeployment(n, d int, u *unstructured.Unstructured) {
	Ω(u.GetNamespace()).Should(Equal(fmt.Sprintf("namespace-%d", n)))
	Ω(u.GetName()).Should(Equal(fmt.Sprintf("deployment-%d", d)))
	Ω(u.Object).ShouldNot(HaveKey("status"))
	Ω(u.Object["metadata"]).ShouldNot(HaveKey("uid"))
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

func unstructuredListFrom(path ...string) *unstructured.UnstructuredList {
	ul := &unstructured.UnstructuredList{}
	b, err := ioutil.ReadFile(filepath.Join(path...))
	Ω(err).ShouldNot(HaveOccurred())
	err = yaml.Unmarshal(b, ul)
	Ω(err).ShouldNot(HaveOccurred())
	return ul
}

func unstructuredFrom(path ...string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	b, err := ioutil.ReadFile(filepath.Join(path...))
	Ω(err).ShouldNot(HaveOccurred())
	err = yaml.Unmarshal(b, u)
	Ω(err).ShouldNot(HaveOccurred())
	return u
}
