package secret

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	gm "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	mockdynamic "github.com/bakito/kubexporter/internal/mocks/client"
	mockmeta "github.com/bakito/kubexporter/internal/mocks/mapper"
)

func Test_toSecret(t *testing.T) {
	ctx := context.TODO()

	s := &corev1.Secret{
		Name:      "test-secret",
		Namespace: "test-ns",
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}
	s.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"})

	u := &unstructured.Unstructured{}
	err := secretScheme.Convert(s, u, ctx)
	if err != nil {
		t.Fatalf("unexpected error converting to unstructured: %v", err)
	}

	res, err := toSecret(ctx, u)
	if err != nil {
		t.Fatalf("unexpected error in toSecret: %v", err)
	}
	if res.Name != s.Name {
		t.Errorf("expected name %s, got %s", s.Name, res.Name)
	}
	if res.Namespace != s.Namespace {
		t.Errorf("expected namespace %s, got %s", s.Namespace, res.Namespace)
	}
	if !bytes.Equal(res.Data["key"], s.Data["key"]) {
		t.Errorf("expected data %s, got %s", string(s.Data["key"]), string(res.Data["key"]))
	}

	// Test error case
	res, err = toSecret(ctx, &corev1.Pod{})
	if err == nil {
		t.Error("expected error in toSecret with Pod, got nil")
	}
	if res != nil {
		t.Errorf("expected nil secret, got %v", res)
	}
}

func Test_secretKeyValue(t *testing.T) {
	s := &corev1.Secret{
		Data: map[string][]byte{
			"key1": []byte("value1"),
		},
	}

	val, err := secretKeyValue(s, "ns", "name", "key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "value1" {
		t.Errorf("expected value1, got %s", val)
	}

	val, err = secretKeyValue(s, "ns", "name", "key2")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if val != "" {
		t.Errorf("expected empty value, got %s", val)
	}
	if !strings.Contains(err.Error(), "key \"key2\" not found in secret ns/name") {
		t.Errorf("expected error message to contain key not found, got %v", err)
	}
}

func Test_readKey(t *testing.T) {
	ctrl := gm.NewController(t)
	defer ctrl.Finish()

	mockClient := mockdynamic.NewMockInterface(ctrl)
	mockMapper := mockmeta.NewMockRESTMapper(ctrl)
	mockNamespaceable := mockdynamic.NewMockNamespaceableResourceInterface(ctrl)
	mockResource := mockdynamic.NewMockResourceInterface(ctrl)

	ctx := context.TODO()
	namespace := "test-ns"
	name := "test-secret"
	key := "test-key"
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}

	t.Run("successful", func(t *testing.T) {
		mapping := &meta.RESTMapping{
			Resource: gvr,
		}
		mockMapper.EXPECT().RESTMapping(schema.GroupKind{Kind: secretKind}, secretVersion).Return(mapping, nil)
		mockClient.EXPECT().Resource(gvr).Return(mockNamespaceable)
		mockNamespaceable.EXPECT().Namespace(namespace).Return(mockResource)

		s := &corev1.Secret{
			Data: map[string][]byte{
				key: []byte("secret-value"),
			},
		}
		s.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"})
		u := &unstructured.Unstructured{}
		_ = secretScheme.Convert(s, u, ctx)

		mockResource.EXPECT().Get(ctx, name, metav1.GetOptions{}).Return(u, nil)

		val, err := readKeyInternal(ctx, mockClient, mockMapper, namespace, name, key)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "secret-value" {
			t.Errorf("expected secret-value, got %s", val)
		}
	})

	t.Run("mapper error", func(t *testing.T) {
		mockMapper.EXPECT().
			RESTMapping(schema.GroupKind{Kind: secretKind}, secretVersion).
			Return(nil, errors.New("mapper error"))

		val, err := readKeyInternal(ctx, mockClient, mockMapper, namespace, name, key)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if err.Error() != "mapper error" {
			t.Errorf("expected mapper error, got %v", err)
		}
		if val != "" {
			t.Errorf("expected empty value, got %s", val)
		}
	})

	t.Run("client get error", func(t *testing.T) {
		mapping := &meta.RESTMapping{
			Resource: gvr,
		}
		mockMapper.EXPECT().RESTMapping(schema.GroupKind{Kind: secretKind}, secretVersion).Return(mapping, nil)
		mockClient.EXPECT().Resource(gvr).Return(mockNamespaceable)
		mockNamespaceable.EXPECT().Namespace(namespace).Return(mockResource)
		mockResource.EXPECT().Get(ctx, name, metav1.GetOptions{}).Return(nil, errors.New("get error"))

		val, err := readKeyInternal(ctx, mockClient, mockMapper, namespace, name, key)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if err.Error() != "get error" {
			t.Errorf("expected get error, got %v", err)
		}
		if val != "" {
			t.Errorf("expected empty value, got %s", val)
		}
	})
}
