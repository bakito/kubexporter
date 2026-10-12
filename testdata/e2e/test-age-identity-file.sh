#!/usr/bin/env bash
# Test age encryption and decryption using the age identity as a file path.
#
# Usage: test-age-identity-file.sh
#
# Generates an age keypair, keeps the identity file, exports a secret, encrypts it
# with the public key, and decrypts it by passing the identity file path instead of
# the identity string — via both --age-identity flag and KUBEXPORTER_AGE_IDENTITY env var.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/encryption-test.sh"

echo "🔐 Testing age encryption with identity file path"

generate_age_keypair_file
trap cleanup_age_keypair_file EXIT

export_secrets "exports-age-file-test"

if ! find_secret_file "exports-age-file-test"; then
  echo "No secret file found, skipping age identity file test"
  exit 0
fi

encrypt_age "$SECRET_FILE"

if verify_encrypted "$SECRET_FILE" "KUBEXPORTER_AGE@"; then
  echo "✅ Age encryption prefix found"
else
  echo "❌ Age encryption prefix not found"
  exit 1
fi

# Test 1: decrypt using --age-identity with a file path
decrypt_age_from_file "$SECRET_FILE"

if verify_decrypted "$SECRET_FILE" "KUBEXPORTER_AGE@"; then
  echo "✅ Age decryption via file path (flag) successful"
else
  echo "❌ Age decryption via file path (flag) failed — prefix still present"
  exit 1
fi

# Re-encrypt for the next test
encrypt_age "$SECRET_FILE"

# Test 2: decrypt using KUBEXPORTER_AGE_IDENTITY env var with a file path
decrypt_age_from_file_env "$SECRET_FILE"

if verify_decrypted "$SECRET_FILE" "KUBEXPORTER_AGE@"; then
  echo "✅ Age decryption via file path (env var) successful"
else
  echo "❌ Age decryption via file path (env var) failed — prefix still present"
  exit 1
fi

echo "✓ Age identity file path roundtrip passed"
