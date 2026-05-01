# Performance Tests

This directory contains performance and load testing scripts for the Point of Sale system using [k6](https://k6.io/).

## Prerequisites

### Install k6

**macOS (Homebrew):**

```bash
brew install k6
```

**Linux (Debian/Ubuntu):**

```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Windows (Chocolatey):**

```powershell
choco install k6
```

**Docker (Alternative):**

```bash
docker pull grafana/k6:latest
```

## Available Tests

### 1. Offline Order Load Test

**File:** `offline_order_load_test.js`  
**Purpose:** Tests offline order creation under high concurrent load (100 users)  
**Task:** T111

**Test Profile:**

- Ramp up to 100 concurrent users over 3.5 minutes
- Sustain 100 users for 3 minutes
- Ramp down over 1.5 minutes
- Total duration: ~8 minutes

**Success Criteria:**

- p95 response time < 2 seconds
- p99 response time < 5 seconds
- Error rate < 5%
- Success rate > 95%

**Usage:**

```bash
# Basic run with default settings
k6 run offline_order_load_test.js

# Run with custom environment variables
k6 run \
  --env BASE_URL=http://localhost:8080 \
  --env API_GATEWAY_URL=http://localhost:8000 \
  --env TENANT_ID=your-tenant-id \
  --env JWT_TOKEN=your-jwt-token \
  offline_order_load_test.js

# Run with results output to JSON
k6 run --out json=results.json offline_order_load_test.js

# Run with Grafana Cloud (optional - for detailed metrics)
k6 run --out cloud offline_order_load_test.js

# Run in Docker
docker run --rm -i \
  -e BASE_URL=http://host.docker.internal:8080 \
  -e API_GATEWAY_URL=http://host.docker.internal:8000 \
  grafana/k6:latest run - <offline_order_load_test.js
```

**Before Running:**

1. **Start all services:**

   ```bash
   cd /home/faris/code/point-of-sale-system
   docker compose up -d
   cd backend/order-service && go run . &
   cd backend/api-gateway && go run . &
   ```

2. **Get a valid JWT token:**

   ```bash
   # Login to get JWT
   curl -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{
       "email": "owner@test.com",
       "password": "password123"
     }' | jq -r '.token'
   ```

3. **Set environment variables:**

   ```bash
   export JWT_TOKEN="your-jwt-token-here"
   export TENANT_ID="your-tenant-id-here"
   ```

4. **Run the test:**
   ```bash
   k6 run offline_order_load_test.js
   ```

**After Running:**

1. **Review test output:**
   - Check the summary statistics printed by k6
   - Verify all thresholds passed (✓ green checkmarks)
   - Review custom metrics: order_creation_success_rate, order_creation_errors

2. **Check system metrics:**
   - Open Grafana: http://localhost:3000
   - View "Offline Orders Dashboard"
   - Check for:
     - CPU/memory usage spikes
     - Database connection pool saturation
     - Kafka consumer lag
     - Vault API response times

3. **Check application logs:**

   ```bash
   docker compose logs order-service | grep ERROR
   docker compose logs postgres | grep "ERROR"
   ```

4. **Clean up test data (optional):**
   ```bash
   # Delete test orders created during load test
   psql -h localhost -U postgres -d pos_db -c "
     DELETE FROM offline_orders
     WHERE customer_email LIKE '%@loadtest.example.com'
     AND created_at > NOW() - INTERVAL '1 hour';
   "
   ```

## Test Results Interpretation

### Response Time Percentiles

- **p50 (median):** 50% of requests completed in this time or less
- **p95:** 95% of requests completed in this time or less (critical for SLA)
- **p99:** 99% of requests completed in this time or less (edge cases)

**Good:**

- p95 < 1 second: Excellent
- p95 < 2 seconds: Good
- p95 < 5 seconds: Acceptable

**Bad:**

- p95 > 5 seconds: Needs optimization
- p99 > 10 seconds: Critical issue

### Error Rate

- **< 0.1%:** Excellent (fewer than 1 in 1000 requests fail)
- **< 1%:** Good
- **< 5%:** Acceptable (threshold in test)
- **> 5%:** Failing - investigate immediately

### Throughput (Requests per second)

- **Target:** ~50 rps (100 users × 1 order / 2 seconds think time)
- **Minimum:** 25 rps (indicates system can handle peak load)
- **Critical:** < 10 rps (system is severely degraded)

## Troubleshooting

### Test Fails with "Connection Refused"

**Symptom:** `WARN[0001] Request Failed error="Post \"http://localhost:8000/api/v1/offline-orders\": dial tcp [::1]:8000: connect: connection refused"`

**Solution:**

1. Verify services are running: `docker compose ps`
2. Check API gateway is listening: `curl http://localhost:8000/health`
3. Use correct host: `http://host.docker.internal:8000` if running k6 in Docker

### Test Fails with "401 Unauthorized"

**Symptom:** `status is 201.............: 0.00%   ✗ 0 ✓ 1234`

**Solution:**

1. Get a fresh JWT token (tokens expire)
2. Ensure token has correct permissions (owner/manager role)
3. Verify tenant_id matches the token's tenant

### Test Fails with "500 Internal Server Error"

**Symptom:** High error rate, response time spikes

**Solution:**

1. Check application logs: `docker compose logs order-service`
2. Check database connections: `docker compose logs postgres`
3. Check Vault connectivity: `docker compose logs vault`
4. Reduce concurrent users to identify breaking point

### Database Connection Pool Exhausted

**Symptom:** Errors about "too many connections" or "connection pool timeout"

**Solution:**

1. Increase PostgreSQL max_connections:

   ```sql
   ALTER SYSTEM SET max_connections = 200;
   SELECT pg_reload_conf();
   ```

2. Tune connection pool in order-service:

   ```go
   // In database configuration
   MaxOpenConns: 100,
   MaxIdleConns: 25,
   ```

3. Add connection pooling (PgBouncer) if needed

## Best Practices

1. **Run during off-peak hours:** Load tests consume significant resources
2. **Monitor system metrics:** Keep Grafana open during tests
3. **Start with smaller load:** Try 10 users first, then scale up
4. **Clean up test data:** Don't let test orders pollute production metrics
5. **Version control results:** Save test outputs for comparison over time
6. **Test in staging first:** Never run load tests against production

## Advanced Usage

### Custom Load Profiles

Create custom scenarios in the test file:

```javascript
export const options = {
  scenarios: {
    // Stress test: gradually increase load until system breaks
    stress_test: {
      executor: 'ramping-arrival-rate',
      startRate: 1,
      timeUnit: '1s',
      preAllocatedVUs: 500,
      maxVUs: 1000,
      stages: [
        { duration: '2m', target: 10 }, // 10 rps
        { duration: '5m', target: 50 }, // 50 rps
        { duration: '5m', target: 100 }, // 100 rps
        { duration: '5m', target: 200 }, // 200 rps (find breaking point)
      ],
    },

    // Soak test: maintain constant load for extended period
    soak_test: {
      executor: 'constant-vus',
      vus: 50,
      duration: '30m', // 30 minutes
    },
  },
}
```

### CI/CD Integration

Run tests in CI pipeline:

```yaml
# .github/workflows/performance-test.yml
name: Performance Test

on:
  schedule:
    - cron: '0 2 * * *' # Run daily at 2 AM
  workflow_dispatch:

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Start services
        run: docker compose up -d

      - name: Wait for services
        run: sleep 30

      - name: Install k6
        run: |
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6

      - name: Run load test
        run: |
          k6 run --out json=results.json tests/performance/offline_order_load_test.js
        env:
          BASE_URL: http://localhost:8080
          API_GATEWAY_URL: http://localhost:8000
          TENANT_ID: ${{ secrets.TEST_TENANT_ID }}
          JWT_TOKEN: ${{ secrets.TEST_JWT_TOKEN }}

      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: k6-results
          path: results.json

      - name: Check thresholds
        run: |
          # Fail if any threshold was exceeded
          if grep -q '"thresholds":.*"ok":false' results.json; then
            echo "❌ Performance thresholds exceeded!"
            exit 1
          fi
```

## Additional Resources

- [k6 Documentation](https://k6.io/docs/)
- [k6 Examples](https://k6.io/docs/examples/)
- [Grafana k6 Cloud](https://grafana.com/products/cloud/k6/)
- [Performance Testing Best Practices](https://k6.io/docs/testing-guides/load-testing/)
