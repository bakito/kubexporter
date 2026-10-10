#!/usr/bin/env bash
# Test AES encryption and decryption roundtrip for kubexporter secrets.
#
# Usage: test-aes-encryption.sh
#
# Generates a random AES key, exports a secret, encrypts it, verifies the
# KUBEXPORTER_AES@ prefix, decrypts it, and confirms the prefix is removed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/encryption-test.sh"

echo "🔑 Testing AES encryption roundtrip"

generate_aes_key
export_secrets "exports-aes-test"

if ! find_secret_file "exports-aes-test"; then
  echo "No secret file found, skipping AES encryption test"
  exit 0
fi

encrypt_aes "$SECRET_FILE"

if verify_encrypted "$SECRET_FILE" "KUBEXPORTER_AES@"; then
  echo "✅ AES encryption prefix found"
else
  echo "❌ AES encryption prefix not found"
  exit 1
fi

decrypt_aes "$SECRET_FILE"

if verify_decrypted "$SECRET_FILE" "KUBEXPORTER_AES@"; then
  echo "✅ AES decryption successful"
else
  echo "❌ AES decryption failed — prefix still present"
  exit 1
fi

echo "✓ AES encryption and decryption roundtrip passed"
