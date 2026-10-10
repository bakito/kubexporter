#!/usr/bin/env bash
# Shared helper functions for encryption e2e tests.
#
# Source this file from test scripts to reuse common logic:
#   source testdata/e2e/encryption-test.sh
set -euo pipefail

# Returns the path to the kubexporter binary.
# Uses KUBEXPORTER_BIN environment variable if set, otherwise builds the binary.
get_bin() {
  if [ -n "${KUBEXPORTER_BIN:-}" ]; then
    echo "$KUBEXPORTER_BIN"
    return
  fi

  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  local project_root
  project_root="$(cd "$script_dir/../.." && pwd)"

  # Build the binary if not already present
  if [ ! -f "$project_root/kubexporter-bin" ]; then
    (cd "$project_root" && go build -o kubexporter-bin .)
  fi
  echo "$project_root/kubexporter-bin"
}

# Generates a random 32-character printable ASCII AES key.
# Sets variable AES_KEY.
generate_aes_key() {
  AES_KEY=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
}

# Generates an age keypair using the system age-keygen binary.
# Sets variables PUBLIC_KEY and IDENTITY.
generate_age_keypair() {
  local tmp_key
  tmp_key="/tmp/age-key-$$-$(date +%s%N)"
  age-keygen -o "$tmp_key" >/dev/null 2>&1

  PUBLIC_KEY=$(grep '^# public key:' "$tmp_key" | sed 's/^# public key: //')
  IDENTITY=$(grep '^AGE-SECRET-KEY-' "$tmp_key")
  rm -f "$tmp_key"
}

# Exports secrets from the cluster without encryption.
# Args: $1 = target directory name
export_secrets() {
  local target="$1"
  unset KUBEXPORTER_AES_KEY
  $(get_bin) --progress simple --target "$target" --namespace e2e-ns1 --include-kinds Secret --otlp-metrics=false
}

# Finds the first Secret.*.yaml file in the given export directory.
# Args: $1 = export directory
# Sets variable SECRET_FILE. Returns 1 if no file found.
find_secret_file() {
  local dir="$1"
  SECRET_FILE=$(find "$dir" -name "Secret.*.yaml" | head -1)
  if [ -z "$SECRET_FILE" ]; then
    echo "No secret file found in $dir"
    return 1
  fi
}

# Encrypts a secret file using AES.
# Args: $1 = secret file path
# Uses variable AES_KEY.
encrypt_aes() {
  local file="$1"
  $(get_bin) encrypt --aes-key "$AES_KEY" "$file"
}

# Encrypts a secret file using age.
# Args: $1 = secret file path
# Uses variable PUBLIC_KEY.
encrypt_age() {
  local file="$1"
  $(get_bin) encrypt --age-public-key "$PUBLIC_KEY" "$file"
}

# Encrypts a secret file using AES key from a K8s secret.
# Args: $1 = secret file, $2 = namespace, $3 = secret name, $4 = secret key
encrypt_aes_k8s() {
  local file="$1" ns="$2" name="$3" key="$4"
  $(get_bin) encrypt \
    --aes-key-secret-namespace "$ns" \
    --aes-key-secret-name "$name" \
    --aes-key-secret-key "$key" \
    "$file"
}

# Encrypts a secret file using age public key from a K8s secret.
# Args: $1 = secret file, $2 = namespace, $3 = secret name, $4 = secret key
encrypt_age_k8s() {
  local file="$1" ns="$2" name="$3" key="$4"
  $(get_bin) encrypt \
    --age-public-key-secret-namespace "$ns" \
    --age-public-key-secret-name "$name" \
    --age-public-key-secret-key "$key" \
    "$file"
}

# Decrypts a secret file using AES.
# Args: $1 = secret file path
# Uses variable AES_KEY.
decrypt_aes() {
  local file="$1"
  $(get_bin) decrypt --aes-key "$AES_KEY" "$file"
}

# Decrypts a secret file using age.
# Args: $1 = secret file path
# Uses variable IDENTITY.
decrypt_age() {
  local file="$1"
  $(get_bin) decrypt --age-identity "$IDENTITY" "$file"
}

# Decrypts a secret file using AES key from a K8s secret.
# Args: $1 = secret file, $2 = namespace, $3 = secret name, $4 = secret key
decrypt_aes_k8s() {
  local file="$1" ns="$2" name="$3" key="$4"
  $(get_bin) decrypt \
    --aes-key-secret-namespace "$ns" \
    --aes-key-secret-name "$name" \
    --aes-key-secret-key "$key" \
    "$file"
}

# Decrypts a secret file using age identity from a K8s secret.
# Args: $1 = secret file, $2 = namespace, $3 = secret name, $4 = secret key
decrypt_age_k8s() {
  local file="$1" ns="$2" name="$3" key="$4"
  $(get_bin) decrypt \
    --age-identity-secret-namespace "$ns" \
    --age-identity-secret-name "$name" \
    --age-identity-secret-key "$key" \
    "$file"
}

# Verifies that a file contains the expected encryption prefix.
# Args: $1 = file path, $2 = prefix (e.g. "KUBEXPORTER_AES@" or "KUBEXPORTER_AGE@")
# Returns 0 if prefix found, 1 otherwise.
verify_encrypted() {
  local file="$1" prefix="$2"
  if grep -q "$prefix" "$file"; then
    return 0
  else
    return 1
  fi
}

# Verifies that a file does NOT contain the encryption prefix (decryption check).
# Args: $1 = file path, $2 = prefix
# Returns 0 if prefix is absent, 1 otherwise.
verify_decrypted() {
  local file="$1" prefix="$2"
  if ! grep -q "$prefix" "$file"; then
    return 0
  else
    return 1
  fi
}

# Creates a K8s namespace for test secrets (idempotent).
# Args: $1 = namespace name
create_test_namespace() {
  local ns="$1"
  kubectl create namespace "$ns" --dry-run=client -o yaml | kubectl apply -f -
}

# Deletes a K8s namespace (ignores if already gone).
# Args: $1 = namespace name
delete_test_namespace() {
  local ns="$1"
  kubectl delete namespace "$ns" --ignore-not-found=true
}
