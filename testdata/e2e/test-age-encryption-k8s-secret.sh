#!/usr/bin/env bash
# Test age encryption and decryption using Kubernetes Secrets for key storage.
#
# Usage: test-age-encryption-k8s-secret.sh
#
# Generates an age keypair, stores it in a Kubernetes Secret, exports a secret,
# encrypts it using the public key from the K8s secret, decrypts using the
# identity from the K8s secret, and verifies the roundtrip.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/encryption-test.sh"

echo "🔐 Testing age encryption with K8s secret key resolution"

TEST_NS="kubexporter-test"
SECRET_NAME="kubexporter-age-key"

generate_age_keypair
create_test_namespace "$TEST_NS"
kubectl create secret generic "$SECRET_NAME" \
  --namespace "$TEST_NS" \
  --from-literal=public-key="$PUBLIC_KEY" \
  --from-literal=identity="$IDENTITY" \
  --dry-run=client -o yaml | kubectl apply -f -

export_secrets "exports-age-k8s-test"

if ! find_secret_file "exports-age-k8s-test"; then
  echo "No secret file found, skipping age K8s secret encryption test"
  exit 0
fi

encrypt_age_k8s "$SECRET_FILE" "$TEST_NS" "$SECRET_NAME" "public-key"

if verify_encrypted "$SECRET_FILE" "KUBEXPORTER_AGE@"; then
  echo "✅ Age encryption prefix found"
else
  echo "❌ Age encryption prefix not found"
  exit 1
fi

decrypt_age_k8s "$SECRET_FILE" "$TEST_NS" "$SECRET_NAME" "identity"

if verify_decrypted "$SECRET_FILE" "KUBEXPORTER_AGE@"; then
  echo "✅ Age decryption successful"
else
  echo "❌ Age decryption failed — prefix still present"
  exit 1
fi

# Cleanup
delete_test_namespace "$TEST_NS"
rm -rf exports-age-k8s-test

echo "✓ Age encryption with K8s secret key resolution passed"
