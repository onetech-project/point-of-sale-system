#!/usr/bin/env bash
# Failing contract test stub: SSE auth and tenant-scoping
# This test is intentionally written to fail until SSE auth (T024) is implemented.

set -euo pipefail

BASE_URL="http://localhost:8085"
SSE_ENDPOINT="/api/v1/sse"

echo "Running SSE auth contract test (expected to fail until T024 implemented)"

echo "1) Connect without token -> expect 401"
if curl -sf -N "${BASE_URL}${SSE_ENDPOINT}" -o /dev/null; then
  echo "ERROR: connection without token succeeded unexpectedly"
  exit 2
else
  echo "OK: connection without token rejected (expected)"
fi

# Helper to create a minimal JWT for test purposes. In CI this should be replaced
# with a token generator that matches the project's auth signing key or a test-only key.
make_jwt() {
  # Placeholder: callers should replace this with a real token generator.
  echo "TEST-TOKEN-FOR-$1"
}

echo "2) Connect with token for wrong tenant -> expect 403"
WRONG_TOKEN="$(make_jwt wrong_tenant)"
if curl -sf -N -H "Authorization: Bearer ${WRONG_TOKEN}" "${BASE_URL}${SSE_ENDPOINT}" -o /dev/null; then
  echo "ERROR: connection with wrong-tenant token succeeded unexpectedly"
  exit 3
else
  echo "OK: connection with wrong-tenant token rejected (expected)"
fi

echo "3) Connect with token that has required claims (tenant_id, roles) -> expect 200 and first SSE message or heartbeat"
GOOD_TOKEN="$(make_jwt good_tenant)"
if curl -sf -N -H "Authorization: Bearer ${GOOD_TOKEN}" "${BASE_URL}${SSE_ENDPOINT}" | sed -n '1,5p' ; then
  echo "OK: connection with good token accepted (expected)"
else
  echo "ERROR: connection with good token failed - SSE auth not implemented or token validation failing"
  exit 4
fi

echo "SSE auth contract test completed (observations above)."
