#!/bin/sh
set -e

# Start Vault in background
echo "Starting Vault server..."
vault server -config=/vault/config/vault-config.hcl &
VAULT_PID=$!

# Wait a moment for Vault to start
sleep 5

# Run initialization script
echo "Running initialization..."
/vault/scripts/vault-init.sh

# Keep container running by waiting for Vault process
wait $VAULT_PID
