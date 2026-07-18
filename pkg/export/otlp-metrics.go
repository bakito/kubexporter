package export

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/bakito/kubexporter/pkg/types"
	"github.com/bakito/kubexporter/version"
)

const (
	meterName           = "github.com/bakito/kubexporter"
	envOtlpHeaderPrefix = "KUBEXPORTER_METRICS_OTLP_HEADER_"
)

// metricDef describes a single OTLP metric emitted by kubexporter.
// Keeping all metric metadata in one place enables easy and automated
// documentation of every exported metric.
type metricDef struct {
	Key         string
	Description string
	Unit        string
}

// int64CounterOptions returns instrument options typed for Int64Counter.
func (m metricDef) int64CounterOptions() []otelmetric.Int64CounterOption {
	opts := []otelmetric.Int64CounterOption{otelmetric.WithDescription(m.Description)}
	if m.Unit != "" {
		opts = append(opts, otelmetric.WithUnit(m.Unit))
	}
	return opts
}

// float64GaugeOptions returns instrument options typed for Float64Gauge.
func (m metricDef) float64GaugeOptions() []otelmetric.Float64GaugeOption {
	opts := []otelmetric.Float64GaugeOption{otelmetric.WithDescription(m.Description)}
	if m.Unit != "" {
		opts = append(opts, otelmetric.WithUnit(m.Unit))
	}
	return opts
}

// Summary-level metric definitions.
var (
	metricKinds = metricDef{
		Key:         "kubexporter.kinds",
		Description: "Number of kinds processed",
	}
	metricQueryPages = metricDef{
		Key:         "kubexporter.query_pages",
		Description: "Total number of query pages requested",
	}
	metricExportedResources = metricDef{
		Key:         "kubexporter.exported_resources",
		Description: "Total number of exported resources",
	}
	metricExportedSizeBytes = metricDef{
		Key:         "kubexporter.exported_size_bytes",
		Description: "Total size of exported resources in bytes",
		Unit:        "By",
	}
	metricNamespaces = metricDef{
		Key:         "kubexporter.namespaces",
		Description: "Number of namespaces containing exported resources",
	}
	metricErrors = metricDef{
		Key:         "kubexporter.errors",
		Description: "Number of errors encountered during export",
	}
	metricDurationSeconds = metricDef{
		Key:         "kubexporter.duration_seconds",
		Description: "Total export duration in seconds",
		Unit:        "s",
	}
)

// Per-resource metric definitions.
var (
	metricResourceInstances = metricDef{
		Key:         "kubexporter.resource.instances",
		Description: "Number of resource instances found per kind",
	}
	metricResourceExportedInstances = metricDef{
		Key:         "kubexporter.resource.exported_instances",
		Description: "Number of exported resource instances per kind",
	}
	metricResourceExportedSizeBytes = metricDef{
		Key:         "kubexporter.resource.exported_size_bytes",
		Description: "Size of exported resources per kind in bytes",
		Unit:        "By",
	}
	metricResourceQueryPages = metricDef{
		Key:         "kubexporter.resource.query_pages",
		Description: "Number of query pages per kind",
	}
	metricResourceQueryDurationSeconds = metricDef{
		Key:         "kubexporter.resource.query_duration_seconds",
		Description: "Query duration per kind in seconds",
		Unit:        "s",
	}
	metricResourceExportDurationSeconds = metricDef{
		Key:         "kubexporter.resource.export_duration_seconds",
		Description: "Export duration per kind in seconds",
		Unit:        "s",
	}
)

// allMetrics is the single source of truth for all metrics emitted by
// kubexporter. Add new metrics here so they are automatically documented.
var allMetrics = []metricDef{
	// Summary metrics
	metricKinds,
	metricQueryPages,
	metricExportedResources,
	metricExportedSizeBytes,
	metricNamespaces,
	metricErrors,
	metricDurationSeconds,
	// Per-resource metrics
	metricResourceInstances,
	metricResourceExportedInstances,
	metricResourceExportedSizeBytes,
	metricResourceQueryPages,
	metricResourceQueryDurationSeconds,
	metricResourceExportDurationSeconds,
}

// MetricsDoc returns a map of every emitted OTLP metric key to its
// description. This can be used to generate documentation automatically.
func MetricsDoc() map[string]string {
	docs := make(map[string]string, len(allMetrics))
	for _, m := range allMetrics {
		docs[m.Key] = m.Description
	}
	return docs
}

func newInt64Counter(meter otelmetric.Meter, m metricDef) (otelmetric.Int64Counter, error) {
	return meter.Int64Counter(m.Key, m.int64CounterOptions()...)
}

func newFloat64Gauge(meter otelmetric.Meter, m metricDef) (otelmetric.Float64Gauge, error) {
	return meter.Float64Gauge(m.Key, m.float64GaugeOptions()...)
}

func (e *exporter) sendOtlpMetrics(ctx context.Context, metrics types.OTLP, resources []*types.GroupResource) error {
	e.l.Printf("\n    Pushing OTLP metrics to %s...\n", metrics.Endpoint)
	provider, err := setupMeterProvider(ctx, metrics)
	if err != nil {
		return err
	}

	// Flush any buffered data and cleanly shut down the exporter on exit
	defer func() {
		if sErr := provider.Shutdown(ctx); sErr != nil {
			e.l.Printf("error shutting down meter provider: %v\n", sErr)
		}
	}()

	meter := otel.Meter(meterName)

	clusterHost := ""
	if e.ac != nil && e.ac.RestConfig != nil {
		clusterHost = e.ac.RestConfig.Host
	}
	commonAttrs := []attribute.KeyValue{
		attribute.String("cluster", clusterHost),
		attribute.String("target", e.config.Target),
	}

	if err := e.recordSummaryMetrics(ctx, meter, commonAttrs); err != nil {
		return err
	}

	return e.recordPerResourceMetrics(ctx, meter, resources, commonAttrs)
}

func (e *exporter) recordSummaryMetrics(
	ctx context.Context,
	meter otelmetric.Meter,
	commonAttrs []attribute.KeyValue,
) error {
	kinds, err := newInt64Counter(meter, metricKinds)
	if err != nil {
		return err
	}
	pages, err := newInt64Counter(meter, metricQueryPages)
	if err != nil {
		return err
	}
	exported, err := newInt64Counter(meter, metricExportedResources)
	if err != nil {
		return err
	}
	exportedSize, err := newInt64Counter(meter, metricExportedSizeBytes)
	if err != nil {
		return err
	}
	namespaces, err := newInt64Counter(meter, metricNamespaces)
	if err != nil {
		return err
	}
	errorsCounter, err := newInt64Counter(meter, metricErrors)
	if err != nil {
		return err
	}
	duration, err := newFloat64Gauge(meter, metricDurationSeconds)
	if err != nil {
		return err
	}

	opt := otelmetric.WithAttributes(commonAttrs...)
	kinds.Add(ctx, int64(e.stats.Kinds), opt)
	pages.Add(ctx, int64(e.stats.Pages), opt)
	exported.Add(ctx, int64(e.stats.Resources), opt)
	exportedSize.Add(ctx, e.stats.ExportedSize, opt)
	namespaces.Add(ctx, int64(e.stats.Namespaces()), opt)
	errorsCounter.Add(ctx, int64(e.stats.Errors), opt)

	if !e.start.IsZero() {
		duration.Record(ctx, time.Since(e.start).Seconds(), opt)
	}
	return nil
}

func (*exporter) recordPerResourceMetrics(
	ctx context.Context,
	meter otelmetric.Meter,
	resources []*types.GroupResource,
	commonAttrs []attribute.KeyValue,
) error {
	if len(resources) == 0 {
		return nil
	}

	instances, err := newInt64Counter(meter, metricResourceInstances)
	if err != nil {
		return err
	}
	exportedInstances, err := newInt64Counter(meter, metricResourceExportedInstances)
	if err != nil {
		return err
	}
	resourceSize, err := newInt64Counter(meter, metricResourceExportedSizeBytes)
	if err != nil {
		return err
	}
	resourcePages, err := newInt64Counter(meter, metricResourceQueryPages)
	if err != nil {
		return err
	}
	queryDuration, err := newFloat64Gauge(meter, metricResourceQueryDurationSeconds)
	if err != nil {
		return err
	}
	exportDuration, err := newFloat64Gauge(meter, metricResourceExportDurationSeconds)
	if err != nil {
		return err
	}

	for _, r := range resources {
		attrs := append([]attribute.KeyValue{}, commonAttrs...)
		attrs = append(attrs,
			attribute.String("group", r.APIGroup),
			attribute.String("version", r.APIVersion),
			attribute.String("kind", r.APIResource.Kind),
			attribute.Bool("namespaced", r.APIResource.Namespaced),
		)
		if r.Error != "" {
			attrs = append(attrs, attribute.String("error", r.Error))
		}
		opt := otelmetric.WithAttributes(attrs...)

		instances.Add(ctx, int64(r.Instances), opt)
		exportedInstances.Add(ctx, int64(r.ExportedInstances), opt)
		resourceSize.Add(ctx, r.ExportedSize, opt)
		resourcePages.Add(ctx, int64(r.Pages), opt)
		queryDuration.Record(ctx, r.QueryDuration.Seconds(), opt)
		exportDuration.Record(ctx, r.ExportDuration.Seconds(), opt)
	}
	return nil
}

func setupMeterProvider(ctx context.Context, metrics types.OTLP) (*sdkmetric.MeterProvider, error) {
	if metrics.Endpoint == "" {
		return nil, errors.New("metrics endpoint must not be empty")
	}
	options := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(metrics.Endpoint)}
	if metrics.Insecure {
		options = append(options, otlpmetricgrpc.WithInsecure())
	}
	headers := headersFromEnv()
	if len(headers) > 0 {
		options = append(options, otlpmetricgrpc.WithHeaders(headers))
	}

	exp, err := otlpmetricgrpc.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("kubexporter"),
			semconv.ServiceVersion(version.Version),
		),
	)
	if err != nil {
		return nil, err
	}

	// The interval barely matters here — nothing will export on the
	// PeriodicReader's timer before the task ends. We rely on
	// Shutdown() to force the flush at the right moment.
	reader := sdkmetric.NewPeriodicReader(exp)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)
	return mp, nil
}

func headersFromEnv() map[string]string {
	headers := make(map[string]string)
	for _, e := range os.Environ() {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) != 2 || !strings.HasPrefix(kv[0], envOtlpHeaderPrefix) {
			continue
		}
		name := strings.TrimPrefix(kv[0], envOtlpHeaderPrefix)
		headers[name] = kv[1]
	}
	return headers
}
