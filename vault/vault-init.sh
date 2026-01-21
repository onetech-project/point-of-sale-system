#!/bin/sh
set -e

echo "====================================="
echo "Vault Initialization Script"
echo "====================================="

# Set Vault address (from environment variable or default)
export VAULT_ADDR="https://vault:8200"
export VAULT_CACERT="/vault/tls/ca.crt" # Path to CA certificate in container

# Wait for Vault server to start listening
echo "Waiting for Vault server to start..."
sleep 5

# Check if Vault is initialized (exit code 2 = not initialized, 0 = initialized and unsealed, 1 = error)
echo "Checking Vault initialization status..."
if vault status 2>&1 | grep -q "Initialized.*false"; then
  echo "Initializing Vault for first time..."
  vault operator init -key-shares=3 -key-threshold=3 > /vault/data/init-keys.txt

  ROOT_TOKEN=$(grep 'Initial Root Token:' /vault/data/init-keys.txt | awk '{print $NF}')

  export VAULT_TOKEN="$ROOT_TOKEN"
  echo "✓ Vault initialized"
else
  echo "✓ Vault already initialized"
  
  # Use the root token from init file
  if [ -f /vault/data/init-keys.txt ]; then
    ROOT_TOKEN=$(grep 'Initial Root Token:' /vault/data/init-keys.txt | awk '{print $NF}')
    export VAULT_TOKEN="$ROOT_TOKEN"
  fi
fi

# Unseal if sealed
if vault status 2>&1 | grep -q "Sealed.*true"; then
  echo "Unsealing Vault..."
  UNSEAL_KEY1=$(grep 'Unseal Key 1:' /vault/data/init-keys.txt | awk '{print $NF}')
  UNSEAL_KEY2=$(grep 'Unseal Key 2:' /vault/data/init-keys.txt | awk '{print $NF}')
  UNSEAL_KEY3=$(grep 'Unseal Key 3:' /vault/data/init-keys.txt | awk '{print $NF}')

  echo "Unsealing with key 1..."
  vault operator unseal "$UNSEAL_KEY1"
  if vault status | grep -q "Sealed.*true"; then
    echo "Vault still sealed after first key, proceeding with second key..."
  fi
  
  echo "Unsealing with key 2..."
  vault operator unseal "$UNSEAL_KEY2"
  if vault status | grep -q "Sealed.*true"; then
    echo "Vault still sealed after second key, proceeding with third key..."
  fi
  
  echo "Unsealing with key 3..."
  vault operator unseal "$UNSEAL_KEY3"
  if vault status | grep -q "Sealed.*false"; then
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
for ENV in "$@"; do
  KEY_NAME="pos-$ENV-key"
  if ! vault read "transit/keys/$KEY_NAME" >/dev/null 2>&1; then
    echo "Creating $KEY_NAME with convergent encryption..."
    vault write -f "transit/keys/$KEY_NAME" \
      type=aes256-gcm96 \
      convergent_encryption=true \
      derived=true
    echo "✓ Encryption key $KEY_NAME created with convergent encryption enabled"
  else
    echo "✓ Encryption key $KEY_NAME already exists"
    # Update existing key to enable convergent encryption if not already enabled
    IS_CONVERGENT=$(vault read -field=convergent_encryption "transit/keys/$KEY_NAME")
    IS_DERIVED=$(vault read -field=derived "transit/keys/$KEY_NAME")
    if [ "$IS_CONVERGENT" != "true" ] || [ "$IS_DERIVED" != "true" ]; then
      echo "Updating $KEY_NAME to enable convergent encryption..."
      vault write -f "transit/keys/$KEY_NAME" \
        type=aes256-gcm96 \
        convergent_encryption=true \
        derived=true
      echo "✓ Encryption key $KEY_NAME updated to enable convergent encryption"
    fi
  fi
done

# Display key info
echo ""
echo "====================================="
echo "Vault Configuration Complete"
echo "====================================="
echo "Transit engine: https://vault:8200/v1/transit/"
for ENV in "$@"; do
  KEY_NAME="pos-$ENV-key"
  echo "Encryption key: $KEY_NAME"
  echo "Encryption key info for environment: $ENV"
  vault read "transit/keys/$KEY_NAME" | grep -E "name|type|latest_version"
  echo ""
done
echo "====================================="
