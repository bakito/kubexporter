package worker

import (
	mock_dynamic "github.com/bakito/kubexporter/pkg/mocks/client"
	"github.com/bakito/kubexporter/pkg/types"
	gm "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
)

var _ = Describe("Worker", func() {
	var (
		w          *worker
		mockCtrl   *gm.Controller
		mockClient *mock_dynamic.MockInterface
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
		mockClient = mock_dynamic.NewMockInterface(mockCtrl)
		config = &types.Config{
			FileNameTemplate:     types.DefaultFileNameTemplate,
			ListFileNameTemplate: types.DefaultListFileNameTemplate,
			OutputFormat:         types.DefaultFormat,
			Quiet:                true,
			Target:               tmpDir,
		}
		res = &types.GroupResource{
			APIGroup:        "",
			APIVersion:      "v1",
			APIGroupVersion: "v1",
			APIResource: metav1.APIResource{
				Kind: "Deployment",
			},
		}

		ul = &unstructured.UnstructuredList{
			Object: map[string]interface{}{"kind": "DeploymentList"},
		}
		w = &worker{
			config: config,
			client: mockClient,
		}
	})
	AfterEach(func() {
		err := os.RemoveAll(tmpDir)
		Ω(err).ShouldNot(HaveOccurred())
	})

	Context("exportLists", func() {
		It("do noting with nil args", func() {
			w.exportLists(nil, nil)
		})
		It("do noting with nil args", func() {
			w.exportLists(res, ul)
		})
	})

	Context("exportSingleResources", func() {
		It("do noting with nil args", func() {
			w.exportSingleResources(nil, nil)
		})
		It("do noting with nil args", func() {
			w.exportSingleResources(res, ul)
		})
	})
})
