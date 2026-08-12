package secret

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/bakito/kubexporter/internal/client"
	"github.com/bakito/kubexporter/internal/types"
)

const (
	secretKind    = "Secret"
	secretVersion = "v1"
)

var secretScheme = runtime.NewScheme()

func init() {
	if err := corev1.AddToScheme(secretScheme); err != nil {
		panic(fmt.Errorf("add core v1 types to secret scheme: %w", err))
	}
}

func ReadKey(ctx context.Context, config *types.Config, namespace, name, key string) (string, error) {
	apiClient, err := client.NewAPIClient(config)
	if err != nil {
		return "", err
	}

	return readKeyInternal(ctx, apiClient.Client, apiClient.Mapper, namespace, name, key)
}

func readKeyInternal(
	ctx context.Context,
	dynClient dynamic.Interface,
	mapper meta.RESTMapper,
	namespace, name, key string,
) (string, error) {
	mapping, err := mapper.RESTMapping(schema.GroupKind{Kind: secretKind}, secretVersion)
	if err != nil {
		return "", err
	}

	unstructuredSecret, err := dynClient.Resource(mapping.Resource).
		Namespace(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	secret, err := toSecret(ctx, unstructuredSecret)
	if err != nil {
		return "", err
	}

	return secretKeyValue(secret, namespace, name, key)
}

func toSecret(ctx context.Context, obj runtime.Object) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := secretScheme.Convert(obj, secret, ctx); err != nil {
		return nil, err
	}
	return secret, nil
}

func secretKeyValue(secret *corev1.Secret, namespace, name, key string) (string, error) {
	data, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %s/%s", key, namespace, name)
	}
	return string(data), nil
}
