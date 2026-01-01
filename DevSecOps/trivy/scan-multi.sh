#!/bin/sh

IFS=','

for img in $TRIVY_TARGET_IMAGES; do
  echo "Scanning $TRIVY_TARGET_IMAGES_PREFIX$img:latest"
  trivy image "$TRIVY_TARGET_IMAGES_PREFIX$img:latest" --format sarif --output "/reports/trivy-$(echo $img | tr '/' '_' | tr ':' '_').sarif"
done
echo "Scanning source code in $TRIVY_TARGET_SRC"
trivy fs --format sarif -o /reports/trivy-fs.sarif /src