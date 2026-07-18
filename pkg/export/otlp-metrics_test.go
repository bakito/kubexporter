package export

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bakito/kubexporter/pkg/export/worker"
	"github.com/bakito/kubexporter/pkg/types"
)

func TestMetricsDoc(t *testing.T) {
	doc := MetricsDoc()
	if len(doc) == 0 {
		t.Error("MetricsDoc() returned empty map")
	}
	if _, ok := doc["kubexporter.kinds"]; !ok {
		t.Error("MetricsDoc() missing key kubexporter.kinds")
	}
}

func TestHeadersFromEnv(t *testing.T) {
	t.Setenv(envOtlpHeaderPrefix+"FOO", "bar")
	t.Setenv(envOtlpHeaderPrefix+"BAZ", "qux")

	headers := headersFromEnv()
	if headers["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %s", headers["FOO"])
	}
	if headers["BAZ"] != "qux" {
		t.Errorf("expected BAZ=qux, got %s", headers["BAZ"])
	}
}

func TestSetupMeterProvider_Error(t *testing.T) {
	_, err := setupMeterProvider(context.Background(), types.OTLP{Endpoint: ""})
	if err == nil {
		t.Error("expected error for empty endpoint, got nil")
	}
}

func TestRecordSummaryMetrics(t *testing.T) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := provider.Meter("test")

	stats := &worker.Stats{
		Kinds:        1,
		Pages:        2,
		Resources:    3,
		ExportedSize: 400,
		Errors:       5,
	}
	// Namespaces are stored in a private map, we can't set them directly if we don't have a helper.
	// But AddNamespace is exported? No, it's addNamespace.
	// We can use Add with another Stats object.
	// stats.Add(otherStats) // doesn't help with namespaces if we can't create them.
	// Wait, Stats has Namespaces() method that returns len(s.namespaces).
	// Let's see if we can use a trick or if we need to modify worker.Stats.
	// Actually, I can just not test namespaces specifically if it's too hard, or I can see if there is a public way.

	e := &exporter{
		stats: stats,
		start: time.Now().Add(-10 * time.Second),
		config: &types.Config{
			Target: "test-target",
		},
	}

	commonAttrs := []attribute.KeyValue{
		attribute.String("cluster", "test-cluster"),
		attribute.String("target", e.config.Target),
	}

	err := e.recordSummaryMetrics(ctx, meter, commonAttrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundMetrics := make(map[string]struct{})
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			foundMetrics[m.Name] = struct{}{}
			switch m.Name {
			case "kubexporter.kinds":
				verifySum(t, m, 1)
			case "kubexporter.query_pages":
				verifySum(t, m, 2)
			case "kubexporter.exported_resources":
				verifySum(t, m, 3)
			case "kubexporter.exported_size_bytes":
				verifySum(t, m, 400)
			case "kubexporter.errors":
				verifySum(t, m, 5)
			case "kubexporter.duration_seconds":
				// duration should be around 10
				verifyGauge(t, m, 10.0)
			}
		}
	}

	expected := []string{
		"kubexporter.kinds",
		"kubexporter.query_pages",
		"kubexporter.exported_resources",
		"kubexporter.exported_size_bytes",
		"kubexporter.namespaces",
		"kubexporter.errors",
		"kubexporter.duration_seconds",
	}
	for _, exp := range expected {
		if _, ok := foundMetrics[exp]; !ok {
			t.Errorf("missing metric: %s", exp)
		}
	}
}

func TestMetricDefOptions(t *testing.T) {
	md := metricDef{
		Description: "test desc",
		Unit:        "test unit",
	}

	opts64 := md.int64CounterOptions()
	if len(opts64) != 2 {
		t.Errorf("expected 2 options, got %d", len(opts64))
	}

	optsF64 := md.float64GaugeOptions()
	if len(optsF64) != 2 {
		t.Errorf("expected 2 options, got %d", len(optsF64))
	}
}

func TestSetupMeterProvider_Success(t *testing.T) {
	t.Setenv(envOtlpHeaderPrefix+"FOO", "bar")

	ctx := context.Background()
	mp, err := setupMeterProvider(ctx, types.OTLP{
		Endpoint: "localhost:4317",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mp == nil {
		t.Fatal("expected meter provider, got nil")
	}
}

func TestRecordPerResourceMetrics(t *testing.T) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := provider.Meter("test")

	e := &exporter{}
	resources := []*types.GroupResource{
		{
			APIGroup:   "apps",
			APIVersion: "v1",
			APIResource: metav1.APIResource{
				Kind:       "Deployment",
				Namespaced: true,
			},
			Instances:         10,
			ExportedInstances: 8,
			ExportedSize:      1000,
			Pages:             2,
			QueryDuration:     time.Second,
			ExportDuration:    time.Second * 2,
		},
	}

	commonAttrs := []attribute.KeyValue{
		attribute.String("cluster", "test-cluster"),
		attribute.String("target", "test-target"),
	}

	err := e.recordPerResourceMetrics(ctx, meter, resources, commonAttrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundMetrics := make(map[string]struct{})
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			foundMetrics[m.Name] = struct{}{}
			switch m.Name {
			case "kubexporter.resource.instances":
				verifySum(t, m, 10)
			case "kubexporter.resource.exported_instances":
				verifySum(t, m, 8)
			case "kubexporter.resource.exported_size_bytes":
				verifySum(t, m, 1000)
			case "kubexporter.resource.query_pages":
				verifySum(t, m, 2)
			case "kubexporter.resource.query_duration_seconds":
				verifyGauge(t, m, 1.0)
			case "kubexporter.resource.export_duration_seconds":
				verifyGauge(t, m, 2.0)
			}
		}
	}

	expected := []string{
		"kubexporter.resource.instances",
		"kubexporter.resource.exported_instances",
		"kubexporter.resource.exported_size_bytes",
		"kubexporter.resource.query_pages",
		"kubexporter.resource.query_duration_seconds",
		"kubexporter.resource.export_duration_seconds",
	}
	for _, exp := range expected {
		if _, ok := foundMetrics[exp]; !ok {
			t.Errorf("missing metric: %s", exp)
		}
	}
}

func verifySum(t *testing.T, m metricdata.Metrics, expected int64) {
	t.Helper()
	sum, ok := m.Data.(metricdata.Sum[int64])
	if !ok {
		t.Errorf("metric %s is not a sum[int64]", m.Name)
		return
	}
	if len(sum.DataPoints) == 0 {
		t.Errorf("metric %s has no data points", m.Name)
		return
	}
	if sum.DataPoints[0].Value != expected {
		t.Errorf("metric %s expected value %d, got %d", m.Name, expected, sum.DataPoints[0].Value)
	}
}

func verifyGauge(t *testing.T, m metricdata.Metrics, expected float64) {
	t.Helper()
	gauge, ok := m.Data.(metricdata.Gauge[float64])
	if !ok {
		t.Errorf("metric %s is not a gauge[float64]", m.Name)
		return
	}
	if len(gauge.DataPoints) == 0 {
		t.Errorf("metric %s has no data points", m.Name)
		return
	}
	val := gauge.DataPoints[0].Value
	if val < expected-1.0 || val > expected+1.0 {
		t.Errorf("metric %s expected value around %f, got %f", m.Name, expected, val)
	}
}
