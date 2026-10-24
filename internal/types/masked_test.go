package types

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestMasked_Setup_md5(t *testing.T) {
	m := &Masked{
		Checksum: "md5",
	}
	err := m.Setup()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if m.doSum == nil {
		t.Error("expected doSum to be set")
	}
	if m.Replacement != DefaultMaskReplacement {
		t.Errorf("expected default replacement, got %s", m.Replacement)
	}
}

func TestMasked_Setup_sha1(t *testing.T) {
	m := &Masked{
		Checksum: "sha1",
	}
	err := m.Setup()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if m.doSum == nil {
		t.Error("expected doSum to be set")
	}
}

func TestMasked_Setup_sha256(t *testing.T) {
	m := &Masked{
		Checksum: "sha256",
	}
	err := m.Setup()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if m.doSum == nil {
		t.Error("expected doSum to be set")
	}
}

func TestMasked_Setup_invalid_checksum(t *testing.T) {
	m := &Masked{
		Checksum: "invalid",
	}
	err := m.Setup()
	if err == nil {
		t.Error("expected error for invalid checksum")
	}
}

func TestMasked_Setup_no_checksum(t *testing.T) {
	m := &Masked{
		Replacement: "MASKED",
	}
	err := m.Setup()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if m.doSum != nil {
		t.Error("expected doSum to be nil when no checksum specified")
	}
	if m.Replacement != "MASKED" {
		t.Errorf("expected replacement to be MASKED, got %s", m.Replacement)
	}
}

func TestMasked_doMask_with_checksum(t *testing.T) {
	m := &Masked{
		Checksum: "md5",
	}
	if err := m.Setup(); err != nil {
		t.Fatal(err)
	}

	result := m.doMask("test")
	if result == "" {
		t.Error("expected non-empty result")
	}
	if result == "test" {
		t.Error("expected masked value, not original")
	}
}

func TestMasked_doMask_without_checksum(t *testing.T) {
	m := &Masked{
		Replacement: "MASKED",
	}
	if err := m.Setup(); err != nil {
		t.Fatal(err)
	}

	result := m.doMask("test")
	if result != "MASKED" {
		t.Errorf("expected MASKED, got %s", result)
	}
}

func TestMasked_doMask_consistent_checksum(t *testing.T) {
	m := &Masked{
		Checksum: "md5",
	}
	if err := m.Setup(); err != nil {
		t.Fatal(err)
	}

	result1 := m.doMask("test")
	result2 := m.doMask("test")
	if result1 != result2 {
		t.Error("expected consistent checksum results")
	}
}

func TestMasked_doMask_different_values(t *testing.T) {
	m := &Masked{
		Checksum: "md5",
	}
	if err := m.Setup(); err != nil {
		t.Fatal(err)
	}

	result1 := m.doMask("test1")
	result2 := m.doMask("test2")
	if result1 == result2 {
		t.Error("expected different checksums for different values")
	}
}

func TestConfig_MaskFields(t *testing.T) {
	config := &Config{
		Masked: &Masked{
			Checksum: "md5",
			KindFields: KindFields{
				"Secret": {
					{"data", "password"},
				},
			},
		},
	}
	if err := config.Masked.Setup(); err != nil {
		t.Fatal(err)
	}

	res := &GroupResource{
		APIGroup:   "",
		APIVersion: "v1",
		APIResource: metav1.APIResource{
			Kind: "Secret",
		},
	}

	us := unstructured.Unstructured{}
	us.SetUnstructuredContent(map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"data": map[string]any{
			"password": "secret-value",
		},
	})

	// Should not panic
	config.MaskFields(res, us)
}

func TestConfig_MaskFields_no_matching_kind(t *testing.T) {
	config := &Config{
		Masked: &Masked{
			Checksum: "md5",
			KindFields: KindFields{
				"Secret": {
					{"data", "password"},
				},
			},
		},
	}
	if err := config.Masked.Setup(); err != nil {
		t.Fatal(err)
	}

	res := &GroupResource{
		APIGroup:   "",
		APIVersion: "v1",
		APIResource: metav1.APIResource{
			Kind: "ConfigMap",
		},
	}

	us := unstructured.Unstructured{}
	us.SetUnstructuredContent(map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"data": map[string]any{
			"key": "value",
		},
	})

	// Should not panic
	config.MaskFields(res, us)
}

func TestConfig_MaskFields_empty_kindFields(t *testing.T) {
	config := &Config{
		Masked: &Masked{
			Checksum:   "md5",
			KindFields: KindFields{},
		},
	}
	if err := config.Masked.Setup(); err != nil {
		t.Fatal(err)
	}

	res := &GroupResource{
		APIGroup:   "",
		APIVersion: "v1",
		APIResource: metav1.APIResource{
			Kind: "Secret",
		},
	}

	us := unstructured.Unstructured{}
	us.SetUnstructuredContent(map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"data": map[string]any{
			"password": "secret-value",
		},
	})

	// Should not panic
	config.MaskFields(res, us)
}
