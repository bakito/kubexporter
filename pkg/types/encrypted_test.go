package types

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("Config-Encrypted", func() {
	var (
		enc    *Encrypted
		config *Config
	)
	BeforeEach(func() {
		enc = &Encrypted{
			AesKey: "1234567890123456",
			KindFields: map[string][][]string{
				"Secret": {{"data"}},
			},
		}
		config = &Config{
			Encrypted: enc,
		}
		_ = os.Unsetenv(EnvAesKey)
	})
	AfterEach(func() {
		_ = os.Unsetenv(EnvAesKey)
	})
	Context("Setup", func() {
		It("should correctly setup the encoder", func() {
			err := enc.Setup()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(enc.gcm).ShouldNot(BeNil())
			Ω(enc.nonce).ShouldNot(BeNil())
		})

		It("should fail if no key is set", func() {
			enc.AesKey = ""
			err := enc.Setup()
			Ω(err).Should(HaveOccurred())
			Ω(enc.gcm).Should(BeNil())
			Ω(enc.nonce).Should(BeNil())
		})

		It("should use key from env", func() {
			_ = os.Setenv(EnvAesKey, enc.AesKey)
			enc.AesKey = ""
			err := enc.Setup()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(enc.gcm).ShouldNot(BeNil())
			Ω(enc.nonce).ShouldNot(BeNil())
		})

		DescribeTable("Key length mus be supported",
			func(key string, ok bool) {
				enc.AesKey = key
				err := enc.Setup()
				if ok {
					Ω(err).ShouldNot(HaveOccurred())
				} else {
					Ω(err).Should(HaveOccurred())
				}
			},
			Entry("16 should be ok", "1234567890123456", true),
			Entry("24 should be ok", "123456789012345678901234", true),
			Entry("32 should be ok", "12345678901234567890123456789012", true),
		)
	})
	Context("Encrypt/Decrypt", func() {
		BeforeEach(func() {
			_ = enc.Setup()
		})
		Context("EncryptFields", func() {
			It("should encrypt the Secret data", func() {
				us := unstructured.Unstructured{Object: map[string]any{
					"data": map[string]any{
						"secret": "don't tell anyone!",
					},
				}}
				config.EncryptFields(&GroupResource{APIResource: metav1.APIResource{Kind: "Secret"}}, us)
				secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
				Ω(secret).Should(HavePrefix(prefix))
			})
			It("should return an empty string if no key is set", func() {
				enc.AesKey = ""
				us := unstructured.Unstructured{Object: map[string]any{
					"data": map[string]any{
						"secret": "don't tell anyone!",
					},
				}}
				config.EncryptFields(&GroupResource{APIResource: metav1.APIResource{Kind: "Secret"}}, us)
				secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
				Ω(secret).Should(BeEmpty())
			})
			Context("decryptFields", func() {
				It("should decrypt the value correctly", func() {
					us := unstructured.Unstructured{Object: map[string]any{
						"data": map[string]any{
							"secret": "KUBEXPORTER_AES@wKCCGma3NhnvzLMbMCrPK7nq7cQV6hF385YuqLjSk+UXCRgaQATO3PPUsfoheg==",
						},
					}}
					cnt, err := decryptFields(us.Object, enc.gcm, len(enc.nonce))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(cnt).Should(Equal(1))
					secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
					Ω(secret).Should(Equal("don't tell anyone!"))
				})
				It("should not decrypt if not decrypted", func() {
					us := unstructured.Unstructured{Object: map[string]any{
						"data": map[string]any{
							"secret": "don't tell anyone!",
						},
					}}
					cnt, err := decryptFields(us.Object, enc.gcm, len(enc.nonce))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(cnt).Should(Equal(0))
					secret, _, _ := unstructured.NestedString(us.Object, "data", "secret")
					Ω(secret).Should(Equal("don't tell anyone!"))
				})
			})
		})
	})
})
