#!/bin/bash

# Script to verify deterministic encryption is working correctly
# This tests that same plaintext + same context = same ciphertext

set -e

echo "=== Deterministic Encryption Verification ==="
echo ""
echo "This script verifies that the Vault convergent encryption is working as expected."
echo "Same plaintext with same context should produce the same ciphertext (deterministic)."
echo ""

# Check if Vault is accessible
if ! curl -s "$VAULT_ADDR/v1/sys/health" > /dev/null 2>&1; then
    echo "❌ ERROR: Vault is not accessible at $VAULT_ADDR"
    echo "Please start Vault first: docker-compose up -d vault"
    exit 1
fi

echo "✓ Vault is accessible at $VAULT_ADDR"
echo ""

# Test encryption with context
TEST_PLAINTEXT="test@example.com"
TEST_CONTEXT="user:email"

echo "Testing encryption with context:"
echo "  Plaintext: $TEST_PLAINTEXT"
echo "  Context: $TEST_CONTEXT"
echo ""

# Encrypt twice with same plaintext and context
ENCRYPTED_1=$(curl -s -X POST \
    -H "X-Vault-Token: $VAULT_TOKEN" \
    -d "{\"plaintext\": \"$(echo -n "$TEST_PLAINTEXT" | base64)\", \"context\": \"$(echo -n "$TEST_CONTEXT" | base64)\"}" \
    "$VAULT_ADDR/v1/transit/encrypt/$VAULT_TRANSIT_KEY" | jq -r '.data.ciphertext')

ENCRYPTED_2=$(curl -s -X POST \
    -H "X-Vault-Token: $VAULT_TOKEN" \
    -d "{\"plaintext\": \"$(echo -n "$TEST_PLAINTEXT" | base64)\", \"context\": \"$(echo -n "$TEST_CONTEXT" | base64)\"}" \
    "$VAULT_ADDR/v1/transit/encrypt/$VAULT_TRANSIT_KEY" | jq -r '.data.ciphertext')

echo "Encryption 1: $ENCRYPTED_1"
echo "Encryption 2: $ENCRYPTED_2"
echo ""

# Verify they are identical (deterministic)
if [ "$ENCRYPTED_1" = "$ENCRYPTED_2" ]; then
    echo "✅ SUCCESS: Deterministic encryption is working!"
    echo "   Same plaintext + same context = same ciphertext"
else
    echo "❌ FAILED: Ciphertexts do not match!"
    echo "   Deterministic encryption is NOT working correctly"
    exit 1
fi

echo ""

# Test with different context produces different ciphertext
TEST_CONTEXT_2="invitation:email"
ENCRYPTED_3=$(curl -s -X POST \
    -H "X-Vault-Token: $VAULT_TOKEN" \
    -d "{\"plaintext\": \"$(echo -n "$TEST_PLAINTEXT" | base64)\", \"context\": \"$(echo -n "$TEST_CONTEXT_2" | base64)\"}" \
    "$VAULT_ADDR/v1/transit/encrypt/$VAULT_TRANSIT_KEY" | jq -r '.data.ciphertext')

echo "Testing with different context:"
echo "  Context: $TEST_CONTEXT_2"
echo "  Encryption 3: $ENCRYPTED_3"
echo ""

if [ "$ENCRYPTED_1" != "$ENCRYPTED_3" ]; then
    echo "✅ SUCCESS: Different contexts produce different ciphertexts"
    echo "   Context isolation is working correctly"
else
    echo "❌ FAILED: Same ciphertext with different contexts!"
    echo "   Context isolation is NOT working"
    exit 1
fi

echo ""
echo "=== All Verification Tests Passed ==="
echo ""
echo "Summary:"
echo "  ✓ Deterministic encryption: WORKING"
echo "  ✓ Context isolation: WORKING"
echo "  ✓ Searchable encrypted fields: ENABLED"
echo ""
echo "You can now use direct encrypted value comparison for searches:"
echo "  - No need for email_hash, token_hash columns"
echo "  - Direct WHERE email = \$encrypted_email queries work"
echo "  - Same plaintext + context always produces same ciphertext"
