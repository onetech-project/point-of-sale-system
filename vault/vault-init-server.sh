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
# looping for staging and prodcution key creation

for ENV in staging production; do
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
echo "Transit engine: http://vault:8200/v1/transit/"
echo "Root token stored in: /vault/data/init-keys.txt"
for ENV in staging production; do
  KEY_NAME="pos-$ENV-key"
  echo "Encryption key info for $KEY_NAME:"
  vault read "transit/keys/$KEY_NAME" | grep -E "name|type|latest_version"
done
echo "====================================="
