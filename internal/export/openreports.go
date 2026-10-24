package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	openreportsv1alpha1 "github.com/openreports/reports-api/apis/openreports.io/v1alpha1"
	openreportsversioned "github.com/openreports/reports-api/pkg/client/clientset/versioned"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/bakito/kubexporter/internal/export/worker"
	"github.com/bakito/kubexporter/internal/log"
	"github.com/bakito/kubexporter/internal/types"
)

const (
	resultPass     openreportsv1alpha1.Result         = "pass"
	resultFail     openreportsv1alpha1.Result         = "fail"
	resultError    openreportsv1alpha1.Result         = "error"
	severityInfo   openreportsv1alpha1.ResultSeverity = "info"
	severityHigh   openreportsv1alpha1.ResultSeverity = "high"
	categoryExport                                    = "export"

	// reportNamePrefix is the prefix of the generated report name.
	reportNamePrefix = "kubexporter-"
	// lastExportAnnotation holds the timestamp of the last export the report was updated with.
	lastExportAnnotation = "kubexporter.bakito.net/last-export"
	// maxNameSegmentLength is the max length of the target segment within the report name.
	maxNameSegmentLength = 200
)

// openReportsProvider provides the data needed to create an OpenReports report.
type openReportsProvider interface {
	Stats() *worker.Stats
	Config() *types.Config
	Start() time.Time
	Logger() log.YALI
	OpenReportsClient() *openreportsversioned.Clientset
	Client() dynamic.Interface
}

// SendOpenReports creates an OpenReports Report resource summarizing the export results.
func SendOpenReports(
	ctx context.Context,
	p openReportsProvider,
	cfg *types.OpenReportsConfig,
	resources []*types.GroupResource,
) error {
	stats := p.Stats()
	l := p.Logger()

	l.Printf("\n    Creating OpenReports Report...\n")

	client := p.OpenReportsClient()
	target := p.Config().Target

	report := buildReport(stats, p.Start(), resources, target)
	reportNS := cfg.Namespace
	if result := cronJobOwnerReference(ctx, p.Client(), l); result != nil {
		report.OwnerReferences = []metav1.OwnerReference{*result.ownerRef}
		if reportNS != result.podNS {
			return fmt.Errorf(
				"report namespace %q does not match pod namespace %q when running in a CronJob",
				reportNS,
				result.podNS,
			)
		}
		reportNS = result.podNS
	}
	name, updated, err := upsertReport(ctx, client.OpenreportsV1alpha1().Reports(reportNS), report)
	if err != nil {
		return fmt.Errorf("failed to create OpenReports Report: %w", err)
	}

	action := "created"
	if updated {
		action = "updated"
	}
	l.Printf("    OpenReports Report %s: %s\n", action, name)
	return nil
}

// reportClient is the subset of the generated OpenReports clients needed to upsert a report.
type reportClient[T metav1.Object] interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (T, error)
	Create(ctx context.Context, report T, opts metav1.CreateOptions) (T, error)
	Update(ctx context.Context, report T, opts metav1.UpdateOptions) (T, error)
}

// upsertReport updates the report if it already exists, otherwise it is created.
func upsertReport[T metav1.Object](ctx context.Context, c reportClient[T], report T) (string, bool, error) {
	existing, err := c.Get(ctx, report.GetName(), metav1.GetOptions{})
	switch {
	case apierrors.IsNotFound(err):
		created, cErr := c.Create(ctx, report, metav1.CreateOptions{})
		if cErr != nil {
			return "", false, cErr
		}
		return created.GetName(), false, nil
	case err != nil:
		return "", false, err
	}

	report.SetResourceVersion(existing.GetResourceVersion())
	updated, err := c.Update(ctx, report, metav1.UpdateOptions{})
	if err != nil {
		return "", false, err
	}
	return updated.GetName(), true, nil
}

func buildReport(
	stats *worker.Stats,
	start time.Time,
	resources []*types.GroupResource,
	target string,
) *openreportsv1alpha1.Report {
	now := time.Now()
	results, summary := buildReportResults(stats, start, resources, now)

	return &openreportsv1alpha1.Report{
		APIVersion: openreportsv1alpha1.SchemeGroupVersion.String(),
		Kind:       "Report",
		ObjectMeta: reportMeta(now, target),
		Source:     "kubexporter",
		Results:    results,
		Summary:    summary,
	}
}

func reportMeta(now time.Time, target string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: reportName(target),
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "kubexporter",
			"app.kubernetes.io/name":       "kubexporter",
		},
		Annotations: map[string]string{
			lastExportAnnotation: now.UTC().Format(time.RFC3339),
		},
	}
}

// reportName builds the deterministic report name kubexporter-<target>.
func reportName(target string) string {
	name := sanitizeNameSegment(filepath.Base(target))
	if name == "" {
		name = sanitizeNameSegment(types.DefaultTarget)
	}
	return reportNamePrefix + name
}

// sanitizeNameSegment converts the given value into a valid RFC 1123 name segment.
func sanitizeNameSegment(value string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(value) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-':
			sb.WriteRune(r)
		default:
			sb.WriteRune('-')
		}
	}
	name := strings.Trim(sb.String(), "-")
	if len(name) > maxNameSegmentLength {
		name = strings.Trim(name[:maxNameSegmentLength], "-")
	}
	return name
}

func buildReportResults(
	stats *worker.Stats,
	start time.Time,
	resources []*types.GroupResource,
	now time.Time,
) ([]openreportsv1alpha1.ReportResult, openreportsv1alpha1.ReportSummary) {
	timestamp := metav1.Timestamp{
		Seconds: now.Unix(),
		Nanos:   int32(now.Nanosecond()),
	}
	results := buildResults(stats, start, timestamp, resources)
	return results, buildSummary(results)
}

func buildResults(
	stats *worker.Stats,
	start time.Time,
	ts metav1.Timestamp,
	resources []*types.GroupResource,
) []openreportsv1alpha1.ReportResult {
	duration := time.Since(start)

	items := []struct {
		rule    string
		message string
		value   string
	}{
		{"kinds", "kinds processed", strconv.Itoa(stats.Kinds)},
		{"pages", "query pages", strconv.Itoa(stats.Pages)},
		{"resources", "resources exported", strconv.Itoa(stats.Resources)},
		{"size", "exported size (bytes)", strconv.FormatInt(stats.ExportedSize, 10)},
		{"namespaces", "namespaces", strconv.Itoa(stats.Namespaces())},
		{"errors", "errors", strconv.Itoa(stats.Errors)},
		{"duration", "duration", types.FormatDuration(duration)},
	}

	results := make([]openreportsv1alpha1.ReportResult, 0, len(items)+len(resources))
	for _, item := range items {
		result := openreportsv1alpha1.ReportResult{
			Source:      "kubexporter",
			Policy:      "kubexporter/summary",
			Rule:        item.rule,
			Category:    categoryExport,
			Severity:    severityInfo,
			Description: item.message,
			Properties: map[string]string{
				"value": item.value,
			},
			Result:    resultPass,
			Timestamp: ts,
		}
		if item.rule == "errors" && stats.HasErrors() {
			result.Result = resultFail
			result.Severity = severityHigh
		}
		results = append(results, result)
	}

	// Add a metric for each individual resource kind
	for _, res := range resources {
		gk := res.GroupKind()
		result := openreportsv1alpha1.ReportResult{
			Source:      "kubexporter",
			Policy:      "kubexporter/resource",
			Rule:        gk,
			Category:    categoryExport,
			Severity:    severityInfo,
			Description: "exported " + gk,
			Properties: map[string]string{
				"instances":         strconv.Itoa(res.Instances),
				"exportedInstances": strconv.Itoa(res.ExportedInstances),
				"pages":             strconv.Itoa(res.Pages),
				"exportedSizeBytes": strconv.FormatInt(res.ExportedSize, 10),
				"queryDuration":     types.FormatDuration(res.QueryDuration),
				"exportDuration":    types.FormatDuration(res.ExportDuration),
			},
			Result:    resultPass,
			Timestamp: ts,
		}
		if res.Error != "" {
			result.Result = resultError
			result.Severity = severityHigh
			result.Properties["error"] = res.Error
		}
		results = append(results, result)
	}
	return results
}

func buildSummary(results []openreportsv1alpha1.ReportResult) openreportsv1alpha1.ReportSummary {
	summary := openreportsv1alpha1.ReportSummary{}
	for _, result := range results {
		switch result.Result {
		case resultPass:
			summary.Pass++
		case resultFail:
			summary.Fail++
		case openreportsv1alpha1.Result("warn"):
			summary.Warn++
		case resultError:
			summary.Error++
		case openreportsv1alpha1.Result("skip"):
			summary.Skip++
		}
	}
	return summary
}

// cronJobOwnerResult holds the CronJob owner reference and the pod namespace.
type cronJobOwnerResult struct {
	ownerRef *metav1.OwnerReference
	podNS    string
}

// cronJobOwnerReference detects whether the current process runs inside a pod owned by a CronJob.
// It returns the CronJob owner reference and the pod namespace if found, nil otherwise.
// Errors are logged but not returned, as this is a non-fatal feature.
func cronJobOwnerReference(
	ctx context.Context,
	client dynamic.Interface,
	l log.YALI,
) *cronJobOwnerResult {
	podNS := getPodNamespace()
	if podNS == "" {
		return nil
	}

	podName := getPodName()
	if podName == "" {
		return nil
	}

	podGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	pod, err := client.Resource(podGVR).Namespace(podNS).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		l.Printf("    Could not get pod %q: %v\n", podName, err)
		return nil
	}

	// Find the Job owner reference
	var jobOwner *metav1.OwnerReference
	for _, ref := range pod.GetOwnerReferences() {
		if ref.Kind == "Job" && ref.APIVersion == "batch/v1" {
			jobOwner = &ref
			break
		}
	}
	if jobOwner == nil {
		return nil
	}

	// Get the Job to find its CronJob owner
	jobGVR := schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
	job, err := client.Resource(jobGVR).Namespace(podNS).Get(ctx, jobOwner.Name, metav1.GetOptions{})
	if err != nil {
		l.Printf("    Could not get job %q: %v\n", jobOwner.Name, err)
		return nil
	}

	// Find the CronJob owner reference
	for _, ref := range job.GetOwnerReferences() {
		if ref.Kind == "CronJob" && ref.APIVersion == "batch/v1" {
			ownerRef := metav1.OwnerReference{
				APIVersion:         ref.APIVersion,
				Kind:               ref.Kind,
				Name:               ref.Name,
				UID:                ref.UID,
				BlockOwnerDeletion: ref.BlockOwnerDeletion,
				Controller:         ref.Controller,
			}
			l.Printf("    Found CronJob owner reference: %s\n", ref.Name)
			return &cronJobOwnerResult{
				ownerRef: &ownerRef,
				podNS:    podNS,
			}
		}
	}
	return nil
}

// getPodNamespace returns the namespace of the current pod.
func getPodNamespace() string {
	ns := os.Getenv("POD_NAMESPACE")
	if ns != "" {
		return ns
	}
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// getPodName returns the name of the current pod.
func getPodName() string {
	name := os.Getenv("POD_NAME")
	if name != "" {
		return name
	}
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	return hostname
}
