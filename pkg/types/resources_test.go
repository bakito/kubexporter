package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bakito/kubexporter/pkg/types"
)

var _ = Describe("Resources", func() {
	var res *types.GroupResource
	BeforeEach(func() {
		res = &types.GroupResource{
			APIResource: metav1.APIResource{
				Kind: "kind",
			},
		}
	})

	Context("GroupKind", func() {
		It("should return the kind only", func() {
			Ω(res.GroupKind()).Should(Equal("kind"))
		})
		It("should return the group.kind only", func() {
			res.APIGroup = "group"
			Ω(res.GroupKind()).Should(Equal("group.kind"))
		})
	})
})
