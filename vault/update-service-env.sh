#!/bin/sh
set -e

VAULT_URL="https://localhost:8200"
BACKEND_DIR="../backend/"
CACERT_PATH="../../vault/tls/ca.crt"
TRANSIT_KEY="pos-development-key"

upsert_env() {
  local file="$1"
  local key="$2"
  local value="$3"

  if grep -q "^${key}=" "$file"; then
    sed -i "s|^${key}=.*|${key}=${value}|" "$file"
  else
    sed -i -e '$a\' "$file"
    echo "${key}=${value}" >> "$file"
  fi
}


echo "====================================="
echo "Vault URL: $VAULT_URL"
echo "Backend Directory: $BACKEND_DIR"
echo "CA Certificate Path: $CACERT_PATH"
echo "Transit Key Name: $TRANSIT_KEY"
echo "====================================="

echo "====================================="
echo "Updating Backend .env Files with Vault Configuration"
echo "====================================="

# Update all .env files with the current Vault configuration
TOKEN=$(docker exec pos-vault cat /vault/data/init-keys.txt | grep 'Initial Root Token' | awk '{print $NF}')
find "$BACKEND_DIR" -type f -name ".env" | while read -r file; do
  upsert_env "$file" "VAULT_ADDR" "$VAULT_URL"
  upsert_env "$file" "VAULT_TOKEN" "$TOKEN"
  upsert_env "$file" "VAULT_CACERT" "$CACERT_PATH"
  upsert_env "$file" "VAULT_TRANSIT_KEY" "$TRANSIT_KEY"
done

echo "====================================="
echo "✓ Backend .env files updated with Vault configuration"
echo "====================================="