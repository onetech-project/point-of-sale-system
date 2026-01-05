#!/bin/sh
set -e

echo "====================================="
echo "Vault Initialization Script"
echo "====================================="

# Set Vault address (from environment variable or default)
export VAULT_ADDR=${VAULT_ADDR:-http://0.0.0.0:8200}

# Wait for Vault server to start listening
echo "Waiting for Vault server to start..."
sleep 5

# Check if Vault is initialized (exit code 2 = not initialized, 0 = initialized and unsealed, 1 = error)
echo "Checking Vault initialization status..."
if vault status 2>&1 | grep -q "Initialized.*false"; then
  echo "Initializing Vault for first time..."
  vault operator init -key-shares=1 -key-threshold=1 > /vault/data/init-keys.txt
  
  UNSEAL_KEY=$(grep 'Unseal Key 1:' /vault/data/init-keys.txt | awk '{print $NF}')
  ROOT_TOKEN=$(grep 'Initial Root Token:' /vault/data/init-keys.txt | awk '{print $NF}')
  
  echo "Unsealing Vault..."
  vault operator unseal "$UNSEAL_KEY"
  
  echo "✓ Vault initialized and unsealed"
  export VAULT_TOKEN="$ROOT_TOKEN"
else
  echo "✓ Vault already initialized"
  
  # Use the root token from init file
  if [ -f /vault/data/init-keys.txt ]; then
    ROOT_TOKEN=$(grep 'Initial Root Token:' /vault/data/init-keys.txt | awk '{print $NF}')
    export VAULT_TOKEN="$ROOT_TOKEN"
  fi
  
  # Unseal if sealed
  if vault status 2>&1 | grep -q "Sealed.*true"; then
    echo "Unsealing Vault..."
    UNSEAL_KEY=$(grep 'Unseal Key 1:' /vault/data/init-keys.txt | awk '{print $NF}')
    vault operator unseal "$UNSEAL_KEY"
    echo "✓ Vault unsealed"
  fi
fi

# Enable transit engine if not already enabled
echo "Checking transit secrets engine..."
if ! vault secrets list 2>&1 | grep -q "^transit/"; then
  echo "Enabling transit secrets engine..."
  vault secrets enable transit
  echo "✓ Transit engine enabled"
else
  echo "✓ Transit engine already enabled"
fi

# Create encryption key if it doesn't exist
echo "Checking encryption key..."
if ! vault read transit/keys/pos-encryption-key >/dev/null 2>&1; then
  echo "Creating pos-encryption-key..."
  vault write -f transit/keys/pos-encryption-key
  echo "✓ Encryption key created"
else
  echo "✓ Encryption key already exists"
fi

# Display key info
echo ""
echo "====================================="
echo "Vault Configuration Complete"
echo "====================================="
vault read transit/keys/pos-encryption-key | grep -E "name|type|latest_version"
echo ""
echo "Transit engine: http://vault:8200/v1/transit/"
echo "Encryption key: pos-encryption-key"
echo "Root token stored in: /vault/data/init-keys.txt"
echo "====================================="
