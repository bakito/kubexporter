package export

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"github.com/bakito/kubexporter/internal/log"
)

var testScheme = runtime.NewScheme()

func Test_cronJobOwnerReference(t *testing.T) {
	cronJobUID := "cronjob-uid-123"
	jobUID := "job-uid-456"
	podUID := "pod-uid-789"

	cronJob := map[string]any{
		"apiVersion": "batch/v1",
		"kind":       "CronJob",
		"metadata": map[string]any{
			"name":      "test-cronjob",
			"namespace": "default",
			"uid":       cronJobUID,
		},
	}

	job := map[string]any{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]any{
			"name":      "test-job",
			"namespace": "default",
			"uid":       jobUID,
			"ownerReferences": []any{
				map[string]any{
					"apiVersion":         "batch/v1",
					"kind":               "CronJob",
					"name":               "test-cronjob",
					"uid":                cronJobUID,
					"blockOwnerDeletion": true,
					"controller":         true,
				},
			},
		},
	}

	pod := map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":      "test-pod",
			"namespace": "default",
			"uid":       podUID,
			"ownerReferences": []any{
				map[string]any{
					"apiVersion":         "batch/v1",
					"kind":               "Job",
					"name":               "test-job",
					"uid":                jobUID,
					"blockOwnerDeletion": true,
					"controller":         true,
				},
			},
		},
	}

	tests := []struct {
		name            string
		podNS           string
		podName         string
		objects         []map[string]any
		wantNil         bool
		wantCronJobName string
		wantPodNS       string
	}{
		{
			name:            "pod owned by job owned by cronjob",
			podNS:           "default",
			podName:         "test-pod",
			objects:         []map[string]any{cronJob, job, pod},
			wantNil:         false,
			wantCronJobName: "test-cronjob",
			wantPodNS:       "default",
		},
		{
			name:    "pod without job owner",
			podNS:   "default",
			podName: "test-pod",
			objects: []map[string]any{
				{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]any{
						"name":      "test-pod",
						"namespace": "default",
						"uid":       podUID,
					},
				},
			},
			wantNil: true,
		},
		{
			name:    "job without cronjob owner",
			podNS:   "default",
			podName: "test-pod",
			objects: []map[string]any{
				{
					"apiVersion": "batch/v1",
					"kind":       "Job",
					"metadata": map[string]any{
						"name":      "test-job",
						"namespace": "default",
						"uid":       jobUID,
					},
				},
				{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]any{
						"name":      "test-pod",
						"namespace": "default",
						"uid":       podUID,
						"ownerReferences": []any{
							map[string]any{
								"apiVersion":         "batch/v1",
								"kind":               "Job",
								"name":               "test-job",
								"uid":                jobUID,
								"blockOwnerDeletion": true,
								"controller":         true,
							},
						},
					},
				},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("POD_NAMESPACE", tt.podNS)
			t.Setenv("POD_NAME", tt.podName)

			// Create fake client with test objects
			client := fake.NewSimpleDynamicClient(testScheme)
			for _, obj := range tt.objects {
				kindVal, ok := obj["kind"].(string)
				if !ok {
					t.Skip("skipping test object without kind")
				}
				kind := kindVal
				var gvr schema.GroupVersionResource
				switch kind {
				case "Job":
					gvr = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
				case "CronJob":
					gvr = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
				default:
					gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
				}
				_, err := client.Resource(gvr).Namespace("default").Create(
					context.Background(),
					&unstructured.Unstructured{Object: obj},
					metav1.CreateOptions{},
				)
				if err != nil {
					t.Fatalf("failed to create test object %q: %v", kind, err)
				}
			}

			l := log.New(true, true)
			got := cronJobOwnerReference(context.Background(), client, l)

			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", *got)
				}
			} else {
				if got == nil {
					t.Error("expected non-nil owner reference")
				} else {
					if got.ownerRef.Name != tt.wantCronJobName {
						t.Errorf("expected CronJob name %q, got %q", tt.wantCronJobName, got.ownerRef.Name)
					}
					if got.podNS != tt.wantPodNS {
						t.Errorf("expected pod namespace %q, got %q", tt.wantPodNS, got.podNS)
					}
				}
			}
		})
	}
}
