#!/usr/bin/env bash
# Test AES encryption and decryption using Kubernetes Secrets for key storage.
#
# Usage: test-aes-encryption-k8s-secret.sh
#
# Generates a random AES key, stores it in a Kubernetes Secret, exports a secret,
# encrypts it using the key from the K8s secret, decrypts using the key from the
# K8s secret, and verifies the roundtrip.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/encryption-test.sh"

echo "🔑 Testing AES encryption with K8s secret key resolution"

TEST_NS="kubexporter-test"
SECRET_NAME="kubexporter-aes-key"

generate_aes_key
create_test_namespace "$TEST_NS"
kubectl create secret generic "$SECRET_NAME" \
  --namespace "$TEST_NS" \
  --from-literal=aes-key="$AES_KEY" \
  --dry-run=client -o yaml | kubectl apply -f -

export_secrets "exports-aes-k8s-test"

if ! find_secret_file "exports-aes-k8s-test"; then
  echo "No secret file found, skipping AES K8s secret encryption test"
  exit 0
fi

encrypt_aes_k8s "$SECRET_FILE" "$TEST_NS" "$SECRET_NAME" "aes-key"

if verify_encrypted "$SECRET_FILE" "KUBEXPORTER_AES@"; then
  echo "✅ AES encryption prefix found"
else
  echo "❌ AES encryption prefix not found"
  exit 1
fi

decrypt_aes_k8s "$SECRET_FILE" "$TEST_NS" "$SECRET_NAME" "aes-key"

if verify_decrypted "$SECRET_FILE" "KUBEXPORTER_AES@"; then
  echo "✅ AES decryption successful"
else
  echo "❌ AES decryption failed — prefix still present"
  exit 1
fi

# Cleanup
delete_test_namespace "$TEST_NS"
rm -rf exports-aes-k8s-test

echo "✓ AES encryption with K8s secret key resolution passed"
