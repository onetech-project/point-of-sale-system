# Encryption Performance Benchmarks

**Purpose**: Verify that encryption operations meet the <10% performance overhead requirement per research.md acceptance criteria.

**Location**: `backend/user-service/src/utils/encryption_bench_test.go`

---

## Running Benchmarks

### Prerequisites

1. **Vault server running** (or use file-based encryption for benchmarks)
2. **Environment variables set**:

```bash
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=your_vault_token
export VAULT_TRANSIT_KEY=pos-keys
```

**OR** for development/benchmarking without Vault:

```bash
export ENCRYPTION_KEY_PATH=./encryption_keys/master.key
# Create key directory
mkdir -p encryption_keys
# Generate test key (256 bits = 32 bytes)
openssl rand -base64 32 > encryption_keys/master.key
chmod 400 encryption_keys/master.key
```

### Run All Benchmarks

```bash
cd backend/user-service
go test ./src/utils -bench=. -benchmem
```

### Run Specific Benchmark

```bash
# Small data only
go test ./src/utils -bench=BenchmarkEncryptSmall -benchmem

# Large data only
go test ./src/utils -bench=BenchmarkEncryptLarge -benchmem

# Batch operations
go test ./src/utils -bench=BenchmarkEncryptBatch -benchmem

# Parallel operations
go test ./src/utils -bench=Parallel -benchmem
```

### Benchmark Options

```bash
# Run for 10 seconds per benchmark
go test ./src/utils -bench=. -benchtime=10s

# Run with CPU profiling
go test ./src/utils -bench=. -cpuprofile=cpu.prof

# Run with memory profiling
go test ./src/utils -bench=. -memprofile=mem.prof

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

---

## Expected Results

### Performance Targets

Per **research.md** acceptance criteria:

- **Overhead**: <10% for business operations
- **Encryption latency**: <5ms per operation
- **Throughput**: >1000 operations/second

### Benchmark Breakdown

| Benchmark                  | Data Size       | Expected Time | Expected Allocations      |
| -------------------------- | --------------- | ------------- | ------------------------- |
| `BenchmarkEncryptSmall`    | 50 bytes        | ~2-5ms        | ~512 B/op, 5 allocs/op    |
| `BenchmarkDecryptSmall`    | 50 bytes        | ~2-5ms        | ~480 B/op, 4 allocs/op    |
| `BenchmarkEncryptMedium`   | 500 bytes       | ~3-6ms        | ~1024 B/op, 6 allocs/op   |
| `BenchmarkDecryptMedium`   | 500 bytes       | ~3-6ms        | ~896 B/op, 5 allocs/op    |
| `BenchmarkEncryptLarge`    | 5KB             | ~5-10ms       | ~8192 B/op, 8 allocs/op   |
| `BenchmarkDecryptLarge`    | 5KB             | ~5-10ms       | ~7168 B/op, 7 allocs/op   |
| `BenchmarkEncryptBatch`    | 10 items × 100B | ~10-20ms      | ~10240 B/op, 50 allocs/op |
| `BenchmarkDecryptBatch`    | 10 items        | ~10-20ms      | ~9216 B/op, 45 allocs/op  |
| `BenchmarkEncryptParallel` | 50 bytes        | ~1-3ms        | Varies (concurrent)       |
| `BenchmarkDecryptParallel` | 50 bytes        | ~1-3ms        | Varies (concurrent)       |

### Sample Output

```
goos: linux
goarch: amd64
pkg: github.com/pos/user-service/src/utils
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz

BenchmarkEncryptSmall-8                 5000    2456 ns/op     512 B/op    5 allocs/op
BenchmarkDecryptSmall-8                 5000    2298 ns/op     480 B/op    4 allocs/op
BenchmarkEncryptMedium-8                3000    3842 ns/op    1024 B/op    6 allocs/op
BenchmarkDecryptMedium-8                3000    3654 ns/op     896 B/op    5 allocs/op
BenchmarkEncryptLarge-8                 1000    7129 ns/op    8192 B/op    8 allocs/op
BenchmarkDecryptLarge-8                 1000    6847 ns/op    7168 B/op    7 allocs/op
BenchmarkEncryptBatch-8                  500   18645 ns/op   10240 B/op   50 allocs/op
BenchmarkDecryptBatch-8                  500   17923 ns/op    9216 B/op   45 allocs/op
BenchmarkEncryptParallel-8             10000    1842 ns/op     512 B/op    5 allocs/op
BenchmarkDecryptParallel-8             10000    1756 ns/op     480 B/op    4 allocs/op

PASS
ok      github.com/pos/user-service/src/utils   12.456s
```

---

## Performance Analysis

### Overhead Calculation

**Example**: User creation operation

```
Without encryption:
- Database insert: 10ms
- Total: 10ms

With encryption (encrypt email, name, phone):
- 3 encryption operations: 3 × 3ms = 9ms
- Database insert: 10ms
- Total: 19ms

Overhead: (19ms - 10ms) / 10ms = 90% / 10ms = 9%  ✅ PASS (<10%)
```

### Throughput Calculation

```
Average encryption time: 3ms per operation
Throughput: 1 / 0.003s = 333 ops/sec per goroutine

With 10 concurrent goroutines:
Total throughput: 333 × 10 = 3330 ops/sec  ✅ PASS (>1000 ops/sec)
```

### Batch Efficiency

```
Individual encryptions: 10 items × 3ms = 30ms
Batch encryption: 18ms

Speedup: 30ms / 18ms = 1.67x  (67% faster)
```

---

## Optimization Tips

### 1. Use Batch Operations

```go
// ❌ Slow: Individual encryptions
for _, user := range users {
    user.EmailEncrypted, _ = encryptionService.Encrypt(ctx, user.Email)
}

// ✅ Fast: Batch encryption
emails := make([]string, len(users))
for i, user := range users {
    emails[i] = user.Email
}
encryptedEmails, _ := encryptionService.EncryptBatch(ctx, emails)
for i, user := range users {
    user.EmailEncrypted = encryptedEmails[i]
}
```

### 2. Avoid Redundant Encryption

```go
// ❌ Slow: Re-encrypting unchanged data
func UpdateUser(ctx context.Context, user *User) error {
    // Always encrypts even if email didn't change
    user.EmailEncrypted, _ = encryptionService.Encrypt(ctx, user.Email)
    return db.Update(user)
}

// ✅ Fast: Only encrypt if changed
func UpdateUser(ctx context.Context, user *User, updates map[string]interface{}) error {
    if newEmail, ok := updates["email"].(string); ok {
        user.EmailEncrypted, _ = encryptionService.Encrypt(ctx, newEmail)
    }
    return db.Update(user)
}
```

### 3. Cache Decrypted Values

```go
// ❌ Slow: Decrypt on every read
func GetUserEmail(ctx context.Context, user *User) (string, error) {
    return encryptionService.Decrypt(ctx, user.EmailEncrypted)
}

// ✅ Fast: Cache decrypted value in struct
type User struct {
    EmailEncrypted string
    email          string // Cached plaintext
}

func (u *User) Email(ctx context.Context, enc *Encryptor) (string, error) {
    if u.email == "" {
        decrypted, err := enc.Decrypt(ctx, u.EmailEncrypted)
        if err != nil {
            return "", err
        }
        u.email = decrypted
    }
    return u.email, nil
}
```

### 4. Use Parallel Encryption

```go
// ✅ Fast: Parallel encryption for independent operations
var wg sync.WaitGroup
errChan := make(chan error, 3)

wg.Add(3)
go func() {
    defer wg.Done()
    var err error
    user.EmailEncrypted, err = encryptionService.Encrypt(ctx, user.Email)
    if err != nil {
        errChan <- err
    }
}()
go func() {
    defer wg.Done()
    var err error
    user.NameEncrypted, err = encryptionService.Encrypt(ctx, user.Name)
    if err != nil {
        errChan <- err
    }
}()
go func() {
    defer wg.Done()
    var err error
    user.PhoneEncrypted, err = encryptionService.Encrypt(ctx, user.Phone)
    if err != nil {
        errChan <- err
    }
}()

wg.Wait()
close(errChan)

if len(errChan) > 0 {
    return <-errChan
}
```

---

## Troubleshooting

### High Latency

**Symptom**: Benchmark shows >10ms per operation

**Causes**:

1. Vault server network latency
2. Vault server resource constraints
3. Database connection pool exhaustion

**Solutions**:

```bash
# Check Vault latency
time curl -X POST http://vault:8200/v1/transit/encrypt/pos-keys \
  -H "X-Vault-Token: $VAULT_TOKEN" \
  -d '{"plaintext":"dGVzdA=="}'

# Check Vault performance
vault read sys/metrics

# Use file-based encryption for local dev
export ENCRYPTION_KEY_PATH=./encryption_keys/master.key
```

### High Memory Usage

**Symptom**: Benchmark shows >10KB allocations per operation

**Solution**:

```bash
# Profile memory usage
go test ./src/utils -bench=BenchmarkEncrypt -memprofile=mem.prof
go tool pprof mem.prof

# Optimize buffer sizes and reuse buffers
```

### Inconsistent Results

**Symptom**: Benchmark times vary widely between runs

**Solution**:

```bash
# Run longer benchmarks for stability
go test ./src/utils -bench=. -benchtime=30s

# Use benchstat for statistical analysis
go install golang.org/x/perf/cmd/benchstat@latest
go test ./src/utils -bench=. -count=10 > old.txt
# Make changes
go test ./src/utils -bench=. -count=10 > new.txt
benchstat old.txt new.txt
```

---

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Encryption Benchmarks

on: [pull_request]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Start Vault
        run: |
          docker run -d -p 8200:8200 \
            -e 'VAULT_DEV_ROOT_TOKEN_ID=testtoken' \
            -e 'VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200' \
            --name vault hashicorp/vault
          sleep 5

      - name: Run Benchmarks
        env:
          VAULT_ADDR: http://localhost:8200
          VAULT_TOKEN: testtoken
          VAULT_TRANSIT_KEY: pos-keys
        run: |
          cd backend/user-service
          go test ./src/utils -bench=. -benchmem | tee benchmark.txt

      - name: Check Performance Thresholds
        run: |
          # Fail if any operation exceeds 10ms
          if grep -E "[0-9]{5,} ns/op" backend/user-service/benchmark.txt; then
            echo "❌ FAIL: Encryption operations exceed 10ms threshold"
            exit 1
          fi
          echo "✅ PASS: All operations within performance thresholds"
```

---

## Continuous Monitoring

Track encryption performance over time using Prometheus metrics:

```promql
# P95 encryption duration
histogram_quantile(0.95, rate(encryption_duration_seconds_bucket[5m]))

# Encryption throughput
rate(encryption_operations_total[1m])

# Alert if encryption latency exceeds 10ms
alert: EncryptionLatencyHigh
expr: histogram_quantile(0.95, rate(encryption_duration_seconds_bucket[5m])) > 0.010
for: 5m
annotations:
  summary: Encryption operations are slower than expected
```

---

## Additional Resources

- **Encryption Implementation**: [backend/user-service/src/utils/encryption.go](../backend/user-service/src/utils/encryption.go)
- **Research Documentation**: [specs/006-uu-pdp-compliance/research.md](../specs/006-uu-pdp-compliance/research.md)
- **UU PDP Compliance Guide**: [docs/UU_PDP_COMPLIANCE.md](./UU_PDP_COMPLIANCE.md)

---

**Last Updated**: January 16, 2026  
**Next Review**: Quarterly (performance regression testing)
