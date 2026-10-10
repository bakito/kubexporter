#!/usr/bin/env bash
# Test age encryption and decryption roundtrip for kubexporter secrets.
#
# Usage: test-age-encryption.sh
#
# Generates an age keypair, exports a secret, encrypts it with the public key,
# verifies the KUBEXPORTER_AGE@ prefix, decrypts with the identity, and confirms
# the prefix is removed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/encryption-test.sh"

echo "🔐 Testing age encryption roundtrip"

generate_age_keypair
export_secrets "exports-age-test"

if ! find_secret_file "exports-age-test"; then
  echo "No secret file found, skipping age encryption test"
  exit 0
fi

encrypt_age "$SECRET_FILE"

if verify_encrypted "$SECRET_FILE" "KUBEXPORTER_AGE@"; then
  echo "✅ Age encryption prefix found"
else
  echo "❌ Age encryption prefix not found"
  exit 1
fi

decrypt_age "$SECRET_FILE"

if verify_decrypted "$SECRET_FILE" "KUBEXPORTER_AGE@"; then
  echo "✅ Age decryption successful"
else
  echo "❌ Age decryption failed — prefix still present"
  exit 1
fi

echo "✓ Age encryption and decryption roundtrip passed"
