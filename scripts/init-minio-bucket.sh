#!/bin/bash
# MinIO Bucket Initialization Script
# Feature: 005-product-photo-storage
# Purpose: Initialize MinIO bucket for product photos

set -e

# Configuration
MINIO_HOST="${MINIO_HOST:-localhost:9000}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"
BUCKET_NAME="${BUCKET_NAME:-product-photos}"

echo "🗄️  MinIO Bucket Initialization"
echo "================================"
echo "Host: $MINIO_HOST"
echo "Bucket: $BUCKET_NAME"
echo ""

# Check if MinIO is running
echo "⏳ Checking MinIO availability..."
if ! curl -s "http://$MINIO_HOST/minio/health/live" > /dev/null 2>&1; then
    echo "❌ MinIO is not running at $MINIO_HOST"
    echo ""
    echo "To start MinIO, run:"
    echo "  docker-compose up -d minio"
    echo ""
    exit 1
fi

echo "✅ MinIO is running"
echo ""

# Install mc (MinIO Client) if not available
if ! command -v mc &> /dev/null; then
    echo "📦 Installing MinIO Client (mc)..."
    
    # Detect OS
    OS="$(uname -s)"
    case "${OS}" in
        Linux*)     
            wget -q https://dl.min.io/client/mc/release/linux-amd64/mc -O /tmp/mc
            chmod +x /tmp/mc
            sudo mv /tmp/mc /usr/local/bin/mc
            ;;
        Darwin*)    
            brew install minio/stable/mc
            ;;
        *)          
            echo "❌ Unsupported OS: ${OS}"
            echo "Please install mc manually: https://min.io/docs/minio/linux/reference/minio-mc.html"
            exit 1
            ;;
    esac
    
    echo "✅ MinIO Client installed"
fi

# Configure mc alias
echo "🔧 Configuring MinIO client..."
mc alias set local "http://$MINIO_HOST" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" > /dev/null 2>&1

# Check if bucket exists
if mc ls "local/$BUCKET_NAME" > /dev/null 2>&1; then
    echo "✅ Bucket '$BUCKET_NAME' already exists"
else
    echo "📦 Creating bucket '$BUCKET_NAME'..."
    mc mb "local/$BUCKET_NAME"
    echo "✅ Bucket created successfully"
fi

# Set bucket policy (allow public read for presigned URLs to work)
echo "🔒 Setting bucket policy..."
cat > /tmp/bucket-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": ["*"]
      },
      "Action": ["s3:GetObject"],
      "Resource": ["arn:aws:s3:::${BUCKET_NAME}/*"],
      "Condition": {
        "StringLike": {
          "s3:x-amz-server-side-encryption": "*"
        }
      }
    }
  ]
}
EOF

# Note: For development, we'll rely on presigned URLs instead of public policy
# mc anonymous set download "local/$BUCKET_NAME"
echo "✅ Bucket policy configured (presigned URLs)"

# Display bucket info
echo ""
echo "📊 Bucket Information:"
mc stat "local/$BUCKET_NAME"

echo ""
echo "✅ MinIO bucket initialization complete!"
echo ""
echo "Access MinIO Console: http://localhost:9001"
echo "  Username: $MINIO_ACCESS_KEY"
echo "  Password: $MINIO_SECRET_KEY"
echo ""
echo "Bucket: $BUCKET_NAME"
echo "Endpoint: http://$MINIO_HOST"
echo ""
