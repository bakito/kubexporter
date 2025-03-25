package types

import (
	"crypto/md5"  // #nosec G501 we are ok with md5
	"crypto/sha1" // #nosec G505 we are ok with sha1
	"crypto/sha256"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Masked masking params.
type Masked struct {
	Replacement string     `json:"replacement" yaml:"replacement"`
	Checksum    string     `json:"checksum"    yaml:"checksum"`
	KindFields  KindFields `json:"kindFields"  yaml:"kindFields"`
	doSum       func(string) string
}

func (m *Masked) Setup() error {
	if m.Checksum != "" {
		switch m.Checksum {
		case "md5":
			m.doSum = func(s string) string {
				// #nosec G401 we are ok with md5
				return fmt.Sprintf("%x", md5.Sum([]byte(s)))
			}
		case "sha1":
			m.doSum = func(s string) string {
				// #nosec G401 we are ok with sha1
				return fmt.Sprintf("%x", sha1.Sum([]byte(s)))
			}
		case "sha256":
			m.doSum = func(s string) string {
				return fmt.Sprintf("%x", sha256.Sum224([]byte(s)))
			}
		default:
			return fmt.Errorf("invalid checksum %q supported are: [md5/sha1/sha256]", m.Checksum)
		}
	}
	if m.Replacement == "" {
		m.Replacement = DefaultMaskReplacement
	}
	return nil
}

func (m *Masked) doMask(val any) string {
	if m.doSum != nil {
		s := fmt.Sprintf("%v", val)
		return m.doSum(s)
	}
	return m.Replacement
}

// MaskFields mask fields for a given resource.
func (c *Config) MaskFields(res *GroupResource, us unstructured.Unstructured) {
	transformNestedFields(c.Masked.KindFields, c.Masked.doMask, res.GroupKind(), us)
}
