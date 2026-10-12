#!/usr/bin/env bash
# Test that kubexporter encrypts Secret fields during export using AES.
#
# Usage: test-aes-encryption-during-export.sh
#
# Generates a random AES key, creates a config file with encryption enabled,
# exports a secret, verifies the exported file contains the KUBEXPORTER_AES@
# prefix, then decrypts it and confirms the prefix is removed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/encryption-test.sh"

echo "🔑 Testing AES encryption during export"

generate_aes_key
tmp_config=$(mktemp /tmp/kubexporter-config-XXXXXX.yaml)
trap "rm -f $tmp_config" EXIT

create_aes_export_config "$AES_KEY" "$tmp_config"
export_secrets_with_config "exports-aes-during-export-test" "$tmp_config"

if ! find_secret_file "exports-aes-during-export-test"; then
  echo "No secret file found, skipping AES during-export encryption test"
  exit 0
fi

if verify_encrypted "$SECRET_FILE" "KUBEXPORTER_AES@"; then
  echo "✅ AES encryption prefix found in exported file"
else
  echo "❌ AES encryption prefix not found in exported file"
  cat "$SECRET_FILE"
  exit 1
fi

# Decrypt the exported file and verify the prefix is removed
decrypt_aes "$SECRET_FILE"

if verify_decrypted "$SECRET_FILE" "KUBEXPORTER_AES@"; then
  echo "✅ AES decryption successful"
else
  echo "❌ AES decryption failed — prefix still present"
  exit 1
fi

# Cleanup
rm -rf exports-aes-during-export-test

echo "✓ AES encryption during export passed"
