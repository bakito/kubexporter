#!/usr/bin/env bash
# Verify that a kubexporter export contains exactly the expected e2e test namespaces.
#
# Usage: verify-namespaces.sh <target-dir> <expected-ns> [<expected-ns> ...]
#
# For every expected namespace the ConfigMap, Secret and Deployment must be
# present. Any of the test namespaces (e2e-ns1, e2e-ns2, e2e-ns3) that is not
# expected must NOT be exported.
set -euo pipefail

target="${1:?target dir required}"
shift
expected=("$@")

# All namespaces that this test suite creates.
all_test_ns=(e2e-ns1 e2e-ns2 e2e-ns3)

# Resource files that must exist for every exported namespace.
required_files=(ConfigMap.e2e-config.yaml Secret.e2e-secret.yaml apps.Deployment.e2e-deploy.yaml)

is_expected() {
  local ns="$1"
  for e in "${expected[@]}"; do
    [[ "$e" == "$ns" ]] && return 0
  done
  return 1
}

fail=0

echo "Verifying export in '${target}' for namespaces: ${expected[*]}"

for ns in "${all_test_ns[@]}"; do
  dir="${target}/${ns}"
  if is_expected "$ns"; then
    if [[ ! -d "$dir" ]]; then
      echo "❌ expected namespace '${ns}' was not exported (missing ${dir})"
      fail=1
      continue
    fi
    for f in "${required_files[@]}"; do
      if [[ ! -f "${dir}/${f}" ]]; then
        echo "❌ namespace '${ns}' is missing exported resource '${f}'"
        fail=1
      else
        echo "✅ found ${ns}/${f}"
      fi
    done
  else
    if [[ -d "$dir" ]]; then
      echo "❌ namespace '${ns}' was exported but should have been filtered out"
      fail=1
    else
      echo "✅ namespace '${ns}' correctly not exported"
    fi
  fi
done

if [[ "$fail" -ne 0 ]]; then
  echo "Namespace verification FAILED"
  exit 1
fi

echo "Namespace verification PASSED"
