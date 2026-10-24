package export

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	openreportsv1alpha1 "github.com/openreports/reports-api/apis/openreports.io/v1alpha1"
	openreportsversioned "github.com/openreports/reports-api/pkg/client/clientset/versioned"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
}

// SendOpenReports creates an OpenReports Report or ClusterReport resource summarizing the export results.
func SendOpenReports(
	ctx context.Context,
	p openReportsProvider,
	cfg *types.OpenReportsConfig,
	resources []*types.GroupResource,
) error {
	stats := p.Stats()
	l := p.Logger()

	kind := "Report"
	if cfg.ClusterReport {
		kind = "ClusterReport"
	}
	l.Printf("\n    Creating OpenReports %s...\n", kind)

	client := p.OpenReportsClient()
	target := p.Config().Target

	var (
		name    string
		updated bool
		err     error
	)
	if cfg.ClusterReport {
		name, updated, err = upsertReport(ctx, client.OpenreportsV1alpha1().ClusterReports(),
			buildClusterReport(stats, p.Start(), resources, target))
	} else {
		name, updated, err = upsertReport(ctx, client.OpenreportsV1alpha1().Reports(cfg.Namespace),
			buildReport(stats, p.Start(), resources, target))
	}
	if err != nil {
		return fmt.Errorf("failed to create OpenReports %s: %w", kind, err)
	}

	action := "created"
	if updated {
		action = "updated"
	}
	l.Printf("    OpenReports %s %s: %s\n", kind, action, name)
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

func buildClusterReport(
	stats *worker.Stats,
	start time.Time,
	resources []*types.GroupResource,
	target string,
) *openreportsv1alpha1.ClusterReport {
	now := time.Now()
	results, summary := buildReportResults(stats, start, resources, now)

	return &openreportsv1alpha1.ClusterReport{
		APIVersion: openreportsv1alpha1.SchemeGroupVersion.String(),
		Kind:       "ClusterReport",
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
