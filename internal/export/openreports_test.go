package export

import (
	"regexp"
	"strconv"
	"testing"
	"time"

	openreportsv1alpha1 "github.com/openreports/reports-api/apis/openreports.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bakito/kubexporter/internal/export/worker"
	"github.com/bakito/kubexporter/internal/types"
)

var reportNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func TestBuildResults(t *testing.T) {
	start := time.Now().Add(-10 * time.Second)
	ts := metav1.Timestamp{
		Seconds: time.Now().Unix(),
	}

	stats := &worker.Stats{
		Kinds:        3,
		Pages:        5,
		Resources:    10,
		ExportedSize: 1024,
		Errors:       0,
	}

	resources := []*types.GroupResource{
		{
			APIGroup:          "",
			APIVersion:        "v1",
			APIResource:       metav1.APIResource{Kind: "Pod"},
			Instances:         5,
			ExportedInstances: 5,
			Pages:             1,
			ExportedSize:      512,
			QueryDuration:     100 * time.Millisecond,
			ExportDuration:    50 * time.Millisecond,
		},
		{
			APIGroup:          "apps",
			APIVersion:        "v1",
			APIResource:       metav1.APIResource{Kind: "Deployment"},
			Instances:         3,
			ExportedInstances: 3,
			Pages:             1,
			ExportedSize:      256,
			QueryDuration:     80 * time.Millisecond,
			ExportDuration:    30 * time.Millisecond,
		},
	}

	results := buildResults(stats, start, ts, resources)

	// 7 summary items + 2 resource items
	if len(results) != 9 {
		t.Fatalf("expected 9 results, got %d", len(results))
	}

	// Check summary items
	summaryRules := make(map[string]bool)
	for _, r := range results[:7] {
		if r.Policy != "kubexporter/summary" {
			t.Errorf("expected summary policy, got %s", r.Policy)
		}
		if r.Result != openreportsv1alpha1.Result("pass") {
			t.Errorf("expected pass result for summary, got %s", r.Result)
		}
		if r.Category != categoryExport {
			t.Errorf("expected export category, got %q", r.Category)
		}
		if r.Severity != severityInfo {
			t.Errorf("expected info severity, got %q", r.Severity)
		}
		summaryRules[r.Rule] = true
	}
	if results[3].Description != "exported size (bytes)" {
		t.Errorf("expected size message to specify bytes, got %q", results[3].Description)
	}
	if _, err := time.ParseDuration(results[6].Properties["value"]); err != nil {
		t.Errorf("summary duration %q is not a Go duration: %v", results[6].Properties["value"], err)
	}

	// Verify all summary rules are present
	expectedRules := []string{"kinds", "pages", "resources", "size", "namespaces", "errors", "duration"}
	for _, rule := range expectedRules {
		if !summaryRules[rule] {
			t.Errorf("missing summary rule: %s", rule)
		}
	}

	// Check per-resource items
	for i, res := range resources {
		r := results[7+i]
		if r.Policy != "kubexporter/resource" {
			t.Errorf("expected resource policy, got %s", r.Policy)
		}
		if r.Rule != res.GroupKind() {
			t.Errorf("expected rule %q, got %q", res.GroupKind(), r.Rule)
		}
		if r.Result != openreportsv1alpha1.Result("pass") {
			t.Errorf("expected pass result for resource, got %s", r.Result)
		}
		if r.Category != categoryExport {
			t.Errorf("expected export category, got %q", r.Category)
		}
		if r.Severity != severityInfo {
			t.Errorf("expected info severity, got %q", r.Severity)
		}
		if r.Properties["instances"] != strconv.Itoa(res.Instances) {
			t.Errorf("expected %d instances, got %q", res.Instances, r.Properties["instances"])
		}
		if r.Properties["exportedInstances"] != strconv.Itoa(res.ExportedInstances) {
			t.Errorf("expected %d exported instances, got %q", res.ExportedInstances, r.Properties["exportedInstances"])
		}
		if r.Properties["exportedSizeBytes"] != strconv.FormatInt(res.ExportedSize, 10) {
			t.Errorf("expected %d exported bytes, got %q", res.ExportedSize, r.Properties["exportedSizeBytes"])
		}
		if r.Properties["queryDuration"] != types.FormatDuration(res.QueryDuration) {
			t.Errorf(
				"expected query duration %q, got %q",
				types.FormatDuration(res.QueryDuration),
				r.Properties["queryDuration"],
			)
		}
		if r.Properties["exportDuration"] != types.FormatDuration(res.ExportDuration) {
			t.Errorf(
				"expected export duration %q, got %q",
				types.FormatDuration(res.ExportDuration),
				r.Properties["exportDuration"],
			)
		}
	}
}

func TestBuildResults_WithError(t *testing.T) {
	start := time.Now().Add(-10 * time.Second)
	ts := metav1.Timestamp{
		Seconds: time.Now().Unix(),
	}

	stats := &worker.Stats{
		Kinds:        1,
		Pages:        1,
		Resources:    0,
		ExportedSize: 0,
		Errors:       1,
	}

	resources := []*types.GroupResource{
		{
			APIGroup:          "batch",
			APIVersion:        "v1",
			Instances:         0,
			ExportedInstances: 0,
			Pages:             0,
			ExportedSize:      0,
			Error:             "Not Found",
		},
	}

	results := buildResults(stats, start, ts, resources)

	// 7 summary items + 1 resource item
	if len(results) != 8 {
		t.Fatalf("expected 8 results, got %d", len(results))
	}

	if results[5].Rule != "errors" {
		t.Fatalf("expected result 5 to describe errors, got %s", results[5].Rule)
	}
	if results[5].Result != openreportsv1alpha1.Result("fail") {
		t.Errorf("expected fail result for errors metric, got %s", results[5].Result)
	}
	if results[5].Severity != severityHigh {
		t.Errorf("expected high severity for errors metric, got %q", results[5].Severity)
	}
	if results[5].Category != categoryExport {
		t.Errorf("expected export category for errors metric, got %q", results[5].Category)
	}

	// The resource with error should have error result
	resourceResult := results[7]
	if resourceResult.Policy != "kubexporter/resource" {
		t.Errorf("expected resource policy, got %s", resourceResult.Policy)
	}
	if resourceResult.Result != openreportsv1alpha1.Result("error") {
		t.Errorf("expected error result, got %s", resourceResult.Result)
	}
	if resourceResult.Severity != severityHigh {
		t.Errorf("expected high severity for errored resource, got %q", resourceResult.Severity)
	}
	if resourceResult.Category != categoryExport {
		t.Errorf("expected export category for errored resource, got %q", resourceResult.Category)
	}
	if resourceResult.Properties["error"] != "Not Found" {
		t.Errorf("expected error property 'Not Found', got %s", resourceResult.Properties["error"])
	}
}

func TestBuildResults_NoResources(t *testing.T) {
	start := time.Now().Add(-10 * time.Second)
	ts := metav1.Timestamp{
		Seconds: time.Now().Unix(),
	}

	stats := &worker.Stats{
		Kinds:        0,
		Pages:        0,
		Resources:    0,
		ExportedSize: 0,
		Errors:       0,
	}

	results := buildResults(stats, start, ts, nil)

	// Only 7 summary items, no resource items
	if len(results) != 7 {
		t.Fatalf("expected 7 results, got %d", len(results))
	}
}

func TestReportName(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "simple", target: "exports", want: "kubexporter-exports"},
		{name: "absolute path", target: "/home/user/my-exports", want: "kubexporter-my-exports"},
		{name: "special chars", target: "/tmp/Export_2024", want: "kubexporter-export-2024"},
		{name: "empty", target: "", want: "kubexporter-exports"},
		{name: "root", target: "/", want: "kubexporter-exports"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reportName(tt.target); got != tt.want {
				t.Errorf("reportName(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestBuildSummary(t *testing.T) {
	tests := []struct {
		name    string
		results []openreportsv1alpha1.ReportResult
		want    openreportsv1alpha1.ReportSummary
	}{
		{
			name: "all pass",
			results: []openreportsv1alpha1.ReportResult{
				{Result: resultPass},
				{Result: resultPass},
				{Result: resultPass},
			},
			want: openreportsv1alpha1.ReportSummary{Pass: 3},
		},
		{
			name: "mixed statuses",
			results: []openreportsv1alpha1.ReportResult{
				{Result: resultPass},
				{Result: resultPass},
				{Result: resultFail},
				{Result: openreportsv1alpha1.Result("warn")},
				{Result: resultError},
				{Result: resultError},
				{Result: openreportsv1alpha1.Result("skip")},
			},
			want: openreportsv1alpha1.ReportSummary{
				Pass:  2,
				Fail:  1,
				Warn:  1,
				Error: 2,
				Skip:  1,
			},
		},
		{
			name: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildSummary(tt.results); got != tt.want {
				t.Errorf("buildSummary() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBuildReport(t *testing.T) {
	report := buildReport(&worker.Stats{}, time.Now(), nil, "/tmp/My Exports")

	if report.Name != "kubexporter-my-exports" {
		t.Errorf("unexpected name: %q", report.Name)
	}
	if !reportNamePattern.MatchString(report.Name) {
		t.Errorf("name %q is not a valid resource name", report.Name)
	}
	if report.GenerateName != "" {
		t.Errorf("expected empty generateName, got %q", report.GenerateName)
	}
	if _, ok := report.Annotations[lastExportAnnotation]; !ok {
		t.Errorf("expected %s annotation", lastExportAnnotation)
	}
	if report.Labels["app.kubernetes.io/managed-by"] != "kubexporter" {
		t.Errorf("unexpected managed-by label: %q", report.Labels["app.kubernetes.io/managed-by"])
	}
	if report.Labels["app.kubernetes.io/name"] != "kubexporter" {
		t.Errorf("unexpected name label: %q", report.Labels["app.kubernetes.io/name"])
	}
	if report.APIVersion != openreportsv1alpha1.SchemeGroupVersion.String() {
		t.Errorf("unexpected API version: %q", report.APIVersion)
	}
	if report.Kind != "Report" {
		t.Errorf("unexpected kind: %q", report.Kind)
	}
	if report.Source != "kubexporter" {
		t.Errorf("unexpected source: %q", report.Source)
	}
	if report.Summary != buildSummary(report.Results) {
		t.Errorf("summary %+v does not match results", report.Summary)
	}
}

func TestBuildClusterReport(t *testing.T) {
	report := buildClusterReport(&worker.Stats{}, time.Now(), nil, "exports")

	if report.Name != "kubexporter-exports" {
		t.Errorf("unexpected name: %q", report.Name)
	}
	if !reportNamePattern.MatchString(report.Name) {
		t.Errorf("name %q is not a valid resource name", report.Name)
	}
	if report.GenerateName != "" {
		t.Errorf("expected empty generateName, got %q", report.GenerateName)
	}
	if _, ok := report.Annotations[lastExportAnnotation]; !ok {
		t.Errorf("expected %s annotation", lastExportAnnotation)
	}
	if report.Labels["app.kubernetes.io/managed-by"] != "kubexporter" {
		t.Errorf("unexpected managed-by label: %q", report.Labels["app.kubernetes.io/managed-by"])
	}
	if report.Labels["app.kubernetes.io/name"] != "kubexporter" {
		t.Errorf("unexpected name label: %q", report.Labels["app.kubernetes.io/name"])
	}
	if report.APIVersion != openreportsv1alpha1.SchemeGroupVersion.String() {
		t.Errorf("unexpected API version: %q", report.APIVersion)
	}
	if report.Kind != "ClusterReport" {
		t.Errorf("unexpected kind: %q", report.Kind)
	}
	if report.Source != "kubexporter" {
		t.Errorf("unexpected source: %q", report.Source)
	}
	if report.Summary != buildSummary(report.Results) {
		t.Errorf("summary %+v does not match results", report.Summary)
	}
}
