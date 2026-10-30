package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
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
	// headlampLinkAnnotation holds dashboard link data for the Headlamp plugin.
	headlampLinkAnnotation = "kubexporter.bakito.ch/headlamp-link"
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

// ValidateOpenReportsConfig checks that the OpenReports namespace is set when not running
// inside a Kubernetes pod (CronJob context), and validates the optional Headlamp link
// annotation. When running in a pod, the namespace is auto-detected from pod metadata.
func ValidateOpenReportsConfig(cfg *types.OpenReportsConfig) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	if cfg.Namespace == "" && getPodNamespace() == "" {
		return errors.New("openreports namespace must be set when not running in a CronJob")
	}
	if value, ok := cfg.Annotations[headlampLinkAnnotation]; ok {
		if err := validateHeadlampLinks(value); err != nil {
			return fmt.Errorf("invalid %s annotation: %w", headlampLinkAnnotation, err)
		}
	}
	return nil
}

type headlampLink struct {
	Title string          `json:"title"`
	URL   string          `json:"url"`
	Icon  json.RawMessage `json:"icon,omitempty"`
}

func validateHeadlampLinks(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("value must not be empty")
	}

	if !strings.HasPrefix(value, "[") {
		if isValidHeadlampURL(value) {
			return nil
		}
		return errors.New("value must be an absolute HTTP(S) URL or a JSON array of links")
	}

	decoder := json.NewDecoder(strings.NewReader(value))
	var links []json.RawMessage
	if err := decoder.Decode(&links); err != nil {
		return fmt.Errorf("invalid headlamp link array: %w", err)
	}
	if len(links) == 0 {
		return errors.New("headlamp link array must contain at least one link")
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err != nil {
			return fmt.Errorf("invalid headlamp link array trailing data: %w", err)
		}
		return errors.New("invalid headlamp link array: trailing data")
	}

	for i, rawLink := range links {
		linkDecoder := json.NewDecoder(strings.NewReader(string(rawLink)))
		linkDecoder.DisallowUnknownFields()

		var link headlampLink
		if err := linkDecoder.Decode(&link); err != nil {
			return fmt.Errorf("headlamp link %d: %w", i, err)
		}
		if len(link.Icon) > 0 {
			var icon string
			if err := json.Unmarshal(link.Icon, &icon); err != nil {
				return fmt.Errorf("headlamp link %d icon must be a string: %w", i, err)
			}
			if strings.TrimSpace(string(link.Icon)) == "null" {
				return fmt.Errorf("headlamp link %d icon must be a string", i)
			}
		}
		if strings.TrimSpace(link.Title) == "" {
			return fmt.Errorf("headlamp link %d title must not be empty", i)
		}
		if !isValidHeadlampURL(link.URL) {
			return fmt.Errorf("headlamp link %d URL must be an absolute HTTP(S) URL with a host", i)
		}
	}

	return nil
}

func isValidHeadlampURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil &&
		parsed.IsAbs() &&
		(strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https")) &&
		parsed.Host != ""
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

	reportNS := cfg.Namespace
	cronJobResult := cronJobOwnerReference(ctx, p.Client(), l)
	if cronJobResult != nil {
		if reportNS != cronJobResult.podNS {
			return fmt.Errorf(
				"report namespace %q does not match pod namespace %q when running in a CronJob",
				reportNS,
				cronJobResult.podNS,
			)
		}
		reportNS = cronJobResult.podNS
	}

	// Fetch the existing report to get previous values for trend comparison
	reportName := reportName(target)
	var prevReport *openreportsv1alpha1.Report
	existing, err := client.OpenreportsV1alpha1().Reports(reportNS).Get(ctx, reportName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get existing OpenReports Report: %w", err)
	}
	if err == nil {
		prevReport = existing
	}

	report := buildReport(stats, p.Start(), resources, target, cfg, prevReport)
	if cronJobResult != nil {
		report.OwnerReferences = []metav1.OwnerReference{*cronJobResult.ownerRef}
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
	cfg *types.OpenReportsConfig,
	prevReport *openreportsv1alpha1.Report,
) *openreportsv1alpha1.Report {
	now := time.Now()
	results, summary := buildReportResults(stats, start, resources, now, prevReport)

	return &openreportsv1alpha1.Report{
		TypeMeta: metav1.TypeMeta{
			APIVersion: openreportsv1alpha1.SchemeGroupVersion.String(),
			Kind:       "Report",
		},
		ObjectMeta: reportMeta(now, target, cfg),
		Source:     "kubexporter",
		Results:    results,
		Summary:    summary,
	}
}

func reportMeta(now time.Time, target string, cfg *types.OpenReportsConfig) metav1.ObjectMeta {
	annotations := map[string]string{
		lastExportAnnotation: now.UTC().Format(time.RFC3339),
	}
	// Merge custom annotations from config
	if cfg != nil {
		maps.Copy(annotations, cfg.Annotations)
	}
	return metav1.ObjectMeta{
		Name: reportName(target),
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "kubexporter",
			"app.kubernetes.io/name":       "kubexporter",
		},
		Annotations: annotations,
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
	prevReport *openreportsv1alpha1.Report,
) ([]openreportsv1alpha1.ReportResult, openreportsv1alpha1.ReportSummary) {
	timestamp := metav1.Timestamp{
		Seconds: now.Unix(),
		Nanos:   int32(now.Nanosecond()),
	}
	prevSummary, prevResources := extractPrevValues(prevReport)
	results := buildResults(stats, start, timestamp, resources, prevSummary, prevResources)
	return results, buildSummary(results)
}

// extractPrevValues extracts the previous values from the existing report for trend comparison.
// It returns maps for summary-level and per-resource-kind previous property values.
func extractPrevValues(
	prevReport *openreportsv1alpha1.Report,
) (summaryPrev map[string]string, resourcePrev map[string]map[string]string) {
	if prevReport == nil {
		return nil, nil
	}
	summaryPrev = make(map[string]string)
	resourcePrev = make(map[string]map[string]string)
	for _, result := range prevReport.Results {
		if result.Policy == "kubexporter/summary" {
			if result.Properties != nil {
				if val, ok := result.Properties["value"]; ok {
					summaryPrev[result.Rule] = val
				}
			}
		} else if result.Policy == "kubexporter/resource" && result.Properties != nil {
			props := make(map[string]string)
			maps.Copy(props, result.Properties)
			resourcePrev[result.Rule] = props
		}
	}
	return summaryPrev, resourcePrev
}

func buildResults(
	stats *worker.Stats,
	start time.Time,
	ts metav1.Timestamp,
	resources []*types.GroupResource,
	prevSummary map[string]string,
	prevResources map[string]map[string]string,
) []openreportsv1alpha1.ReportResult {
	duration := time.Since(start)

	items := []struct {
		rule    string
		message string
		value   string
	}{
		{"kinds", "kinds processed", strconv.Itoa(stats.Kinds)},
		{"resources", "resources exported", strconv.Itoa(stats.Resources)},
		{"size", "exported size (bytes)", strconv.FormatInt(stats.ExportedSize, 10)},
		{"namespaces", "namespaces", strconv.Itoa(stats.Namespaces())},
		{"errors", "errors", strconv.Itoa(stats.Errors)},
		{"duration", "duration", types.FormatDuration(duration)},
	}

	results := make([]openreportsv1alpha1.ReportResult, 0, len(items)+len(resources))
	for _, item := range items {
		props := map[string]string{
			"value": item.value,
		}
		if prevSummary != nil {
			if prevVal, ok := prevSummary[item.rule]; ok {
				props["prevValue"] = prevVal
			}
		}
		result := openreportsv1alpha1.ReportResult{
			Source:      "kubexporter",
			Policy:      "kubexporter/summary",
			Rule:        item.rule,
			Category:    categoryExport,
			Severity:    severityInfo,
			Description: item.message,
			Properties:  props,
			Result:      resultPass,
			Timestamp:   ts,
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
		props := map[string]string{
			"instances":         strconv.Itoa(res.Instances),
			"exportedInstances": strconv.Itoa(res.ExportedInstances),
			"exportedSizeBytes": strconv.FormatInt(res.ExportedSize, 10),
			"queryDuration":     types.FormatDuration(res.QueryDuration),
			"exportDuration":    types.FormatDuration(res.ExportDuration),
		}
		if prevResources != nil {
			if prevProps, ok := prevResources[gk]; ok {
				if prevVal, ok := prevProps["instances"]; ok {
					props["prevInstances"] = prevVal
				}
				if prevVal, ok := prevProps["exportedInstances"]; ok {
					props["prevExportedInstances"] = prevVal
				}
				if prevVal, ok := prevProps["exportedSizeBytes"]; ok {
					props["prevExportedSizeBytes"] = prevVal
				}
			}
		}
		result := openreportsv1alpha1.ReportResult{
			Source:      "kubexporter",
			Policy:      "kubexporter/resource",
			Rule:        gk,
			Category:    categoryExport,
			Severity:    severityInfo,
			Description: "exported " + gk,
			Properties:  props,
			Result:      resultPass,
			Timestamp:   ts,
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
