#!/usr/bin/env bash
# Test that kubexporter encrypts Secret fields during export using age encryption.
#
# Usage: test-age-encryption-during-export.sh
#
# Generates an age keypair, creates a config file with age encryption enabled,
# exports a secret, verifies the exported file contains the KUBEXPORTER_AGE@
# prefix, then decrypts it and confirms the prefix is removed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/encryption-test.sh"

echo "🔐 Testing age encryption during export"

generate_age_keypair
tmp_config=$(mktemp /tmp/kubexporter-config-XXXXXX.yaml)
trap "rm -f $tmp_config" EXIT

create_age_export_config "$PUBLIC_KEY" "$tmp_config"
export_secrets_with_config "exports-age-during-export-test" "$tmp_config"

if ! find_secret_file "exports-age-during-export-test"; then
  echo "No secret file found, skipping age during-export encryption test"
  exit 0
fi

if verify_encrypted "$SECRET_FILE" "KUBEXPORTER_AGE@"; then
  echo "✅ Age encryption prefix found in exported file"
else
  echo "❌ Age encryption prefix not found in exported file"
  cat "$SECRET_FILE"
  exit 1
fi

# Decrypt the exported file and verify the prefix is removed
decrypt_age "$SECRET_FILE"

if verify_decrypted "$SECRET_FILE" "KUBEXPORTER_AGE@"; then
  echo "✅ Age decryption successful"
else
  echo "❌ Age decryption failed — prefix still present"
  exit 1
fi

# Cleanup
rm -rf exports-age-during-export-test

echo "✓ Age encryption during export passed"
