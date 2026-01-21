# script for generating TLS certificates for Vault
#!/bin/sh
set -e

echo "====================================="
echo "Vault TLS Certificate Generation"
echo "====================================="

# Define certificate parameters
CERT_DIR="./tls"
COMMON_NAME_CA="Vault Internal CA"
CA_KEY="$CERT_DIR/ca.key"
CA_CRT="$CERT_DIR/ca.crt"
CSR_CONFIG="$CERT_DIR/vault_csr.cnf"
VAULT_KEY="$CERT_DIR/vault.key"
VAULT_CRT="$CERT_DIR/vault.crt"
VAULT_CSR="$CERT_DIR/vault.csr"
DAYS_VALID=365
BIT=4096

# Create certificate directory if it doesn't exist
mkdir -p "$CERT_DIR"
echo "Certificate directory created at $CERT_DIR"

# Generate CA key and Self-signed certificate
echo "Generating CA key and self-signed certificate..."
openssl genrsa -out "$CA_KEY" $BIT
openssl req -new -x509 -key "$CA_KEY" -out "$CA_CRT" -days $DAYS_VALID -subj "/CN=$COMMON_NAME_CA"
echo "CA key and certificate generated at $CA_KEY and $CA_CRT"

# Generate Vault certificate
echo "Generating Vault server key and certificate signing request (CSR)..."
openssl genrsa -out "$VAULT_KEY" $BIT
echo "Generate CSR config file..."
cat > "$CSR_CONFIG" <<EOF
[req]
default_bits = $BIT
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = req_ext

[dn]
CN = vault

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = vault
DNS.2 = localhost
IP.1 = 0.0.0.0
IP.2 = 127.0.0.1
EOF

# Generate CSR using the config file
echo "Generating CSR using config file..."
openssl req -new -key "$VAULT_KEY" -out "$VAULT_CSR" -config "$CSR_CONFIG"

# Sign Vault CSR with CA to get Vault certificate
echo "Signing Vault CSR with CA to generate Vault certificate..."
openssl x509 -req -in "$VAULT_CSR" \
  -CA "$CA_CRT" -CAkey "$CA_KEY" -CAcreateserial \
  -out "$VAULT_CRT" -days 825 -sha256 \
  -extensions req_ext -extfile "$CSR_CONFIG"


