#!/usr/bin/env bash
# Verify that a kubexporter OTLP metrics dump contains every metric that
# kubexporter is expected to emit.
#
# Usage: verify-metrics.sh <metrics-json-file>
#
# The list below must stay in sync with allMetrics in
# pkg/export/otlp-metrics.go.
set -euo pipefail

file="${1:?metrics json file required}"

if [[ ! -s "$file" ]]; then
  echo "❌ metrics file '${file}' is missing or empty"
  exit 1
fi

# All metrics kubexporter emits (see pkg/export/otlp-metrics.go).
expected_metrics=(
  # metrics-doc-start
  kubexporter.duration_seconds
  kubexporter.errors
  kubexporter.exported_resources
  kubexporter.exported_size_bytes
  kubexporter.kinds
  kubexporter.namespaces
  kubexporter.query_pages
  kubexporter.resource.export_duration_seconds
  kubexporter.resource.exported_instances
  kubexporter.resource.exported_size_bytes
  kubexporter.resource.instances
  kubexporter.resource.query_duration_seconds
  kubexporter.resource.query_pages
  # metrics-doc-end
)

fail=0

echo "Verifying metrics in '${file}'"

for metric in "${expected_metrics[@]}"; do
  if grep -q "\"name\":\"${metric}\"" "$file"; then
    echo "✅ found metric '${metric}'"
  else
    echo "❌ expected metric '${metric}' is missing"
    fail=1
  fi
done

if [[ "$fail" -ne 0 ]]; then
  echo "Metrics verification FAILED"
  exit 1
fi

echo "Metrics verification PASSED"
