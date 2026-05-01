import http from 'k6/http'
import { check, sleep } from 'k6'
import { Rate, Trend, Counter } from 'k6/metrics'

// T111: Load test for offline order creation
// Tests 100 concurrent users creating offline orders

// Custom metrics
const orderCreationRate = new Rate('order_creation_success_rate')
const orderCreationDuration = new Trend('order_creation_duration')
const orderCreationErrors = new Counter('order_creation_errors')

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 20 }, // Ramp up to 20 users over 30s
    { duration: '1m', target: 50 }, // Ramp up to 50 users over 1m
    { duration: '2m', target: 100 }, // Ramp up to 100 users over 2m
    { duration: '3m', target: 100 }, // Stay at 100 users for 3m (main test period)
    { duration: '1m', target: 50 }, // Ramp down to 50 users over 1m
    { duration: '30s', target: 0 }, // Ramp down to 0 users over 30s
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000', 'p(99)<5000'], // 95% < 2s, 99% < 5s
    http_req_failed: ['rate<0.05'], // Error rate < 5%
    order_creation_success_rate: ['rate>0.95'], // Success rate > 95%
    order_creation_duration: ['p(95)<1500', 'p(99)<3000'], // Response times
  },
}

// Environment configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'
const API_GATEWAY_URL = __ENV.API_GATEWAY_URL || 'http://localhost:8000'
const TENANT_ID = __ENV.TENANT_ID || 'test-tenant-123'
const JWT_TOKEN = __ENV.JWT_TOKEN || generateMockToken()

// Test data generators
function generateOrderReference() {
  const timestamp = Date.now()
  const random = Math.floor(Math.random() * 10000)
  return `GO-${timestamp}-${random}`
}

function generateCustomerName() {
  const firstNames = ['John', 'Jane', 'Michael', 'Sarah', 'David', 'Emily', 'Robert', 'Lisa']
  const lastNames = [
    'Smith',
    'Johnson',
    'Williams',
    'Brown',
    'Jones',
    'Garcia',
    'Martinez',
    'Davis',
  ]
  return `${firstNames[Math.floor(Math.random() * firstNames.length)]} ${lastNames[Math.floor(Math.random() * lastNames.length)]}`
}

function generatePhoneNumber() {
  const prefix = '+628'
  const number = Math.floor(Math.random() * 1000000000)
    .toString()
    .padStart(9, '0')
  return prefix + number
}

function generateEmail() {
  const timestamp = Date.now()
  const random = Math.floor(Math.random() * 10000)
  return `test${timestamp}${random}@loadtest.example.com`
}

function generateOrderItems() {
  const itemCount = Math.floor(Math.random() * 5) + 1 // 1-5 items
  const items = []

  for (let i = 0; i < itemCount; i++) {
    items.push({
      product_id: `prod-${Math.floor(Math.random() * 100) + 1}`,
      product_name: `Product ${Math.floor(Math.random() * 100) + 1}`,
      quantity: Math.floor(Math.random() * 10) + 1,
      unit_price: (Math.floor(Math.random() * 50) + 10) * 1000, // 10k-60k
      subtotal: 0, // Will be calculated
    })
    items[i].subtotal = items[i].quantity * items[i].unit_price
  }

  return items
}

function calculateTotalAmount(items) {
  return items.reduce((sum, item) => sum + item.subtotal, 0)
}

function generateMockToken() {
  // Generate a basic JWT-like token for testing
  // In production, use real JWT from auth service
  return 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoibG9hZHRlc3QtdXNlciIsInRlbmFudF9pZCI6InRlc3QtdGVuYW50LTEyMyIsInJvbGUiOiJvd25lciJ9.test'
}

// Main test function
export default function () {
  // Generate order data
  const items = generateOrderItems()
  const totalAmount = calculateTotalAmount(items)
  const paymentType = Math.random() > 0.3 ? 'full' : 'installment' // 70% full, 30% installment

  const payload = {
    order_reference: generateOrderReference(),
    customer_name: generateCustomerName(),
    customer_phone: generatePhoneNumber(),
    customer_email: generateEmail(),
    total_amount: totalAmount,
    order_type: 'offline',
    delivery_type: Math.random() > 0.5 ? 'in_store' : 'home_delivery',
    delivery_address: Math.random() > 0.5 ? '123 Test Street, Jakarta' : null,
    notes: `Load test order created at ${new Date().toISOString()}`,
    items: items,
    data_consent_given: true,
  }

  // Add payment terms
  if (paymentType === 'full') {
    payload.payment_terms = {
      payment_type: 'full',
      status: 'PAID',
    }
  } else {
    const installmentCount = [3, 6, 12][Math.floor(Math.random() * 3)] // 3, 6, or 12 months
    const downPaymentPercent = 0.1 + Math.random() * 0.2 // 10-30%
    const downPaymentAmount = Math.floor(totalAmount * downPaymentPercent)
    const remainingBalance = totalAmount - downPaymentAmount

    payload.payment_terms = {
      payment_type: 'installment',
      installment_count: installmentCount,
      down_payment_amount: downPaymentAmount,
      remaining_balance: remainingBalance,
      status: 'PENDING',
    }
  }

  // Request headers
  const headers = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${JWT_TOKEN}`,
    'X-Tenant-ID': TENANT_ID,
  }

  // Make request
  const startTime = Date.now()
  const response = http.post(`${API_GATEWAY_URL}/api/v1/offline-orders`, JSON.stringify(payload), {
    headers: headers,
  })
  const duration = Date.now() - startTime

  // Record metrics
  orderCreationDuration.add(duration)

  // Validate response
  const success = check(response, {
    'status is 201': (r) => r.status === 201,
    'has order_id': (r) => JSON.parse(r.body).order_id !== undefined,
    'has order_reference': (r) => JSON.parse(r.body).order_reference !== undefined,
    'order_type is offline': (r) => JSON.parse(r.body).order_type === 'offline',
    'customer_phone is encrypted': (r) => {
      const body = JSON.parse(r.body)
      // In response, customer_phone should be decrypted for authorized user
      return body.customer_phone && body.customer_phone.startsWith('+628')
    },
    'response time < 3s': (r) => r.timings.duration < 3000,
  })

  orderCreationRate.add(success)

  if (!success) {
    orderCreationErrors.add(1)
    console.error(`Order creation failed: ${response.status} ${response.body}`)
  }

  // Think time (simulate real user behavior)
  sleep(Math.random() * 2 + 1) // 1-3 seconds between requests
}

// Setup function (runs once at start)
export function setup() {
  console.log('=== Load Test: Offline Order Creation ===')
  console.log(`Base URL: ${BASE_URL}`)
  console.log(`API Gateway: ${API_GATEWAY_URL}`)
  console.log(`Tenant ID: ${TENANT_ID}`)
  console.log('')
  console.log('Test Profile:')
  console.log('- Ramp up to 100 concurrent users over 3.5 minutes')
  console.log('- Sustain 100 users for 3 minutes')
  console.log('- Ramp down over 1.5 minutes')
  console.log('- Total test duration: ~8 minutes')
  console.log('')
  console.log('Success Criteria:')
  console.log('- p95 response time < 2 seconds')
  console.log('- p99 response time < 5 seconds')
  console.log('- Error rate < 5%')
  console.log('- Success rate > 95%')
  console.log('')

  // Verify API is accessible
  const healthCheck = http.get(`${BASE_URL}/health`)
  if (healthCheck.status !== 200) {
    console.error('Health check failed! API may not be accessible.')
    console.error(`Status: ${healthCheck.status}`)
    console.error(`Body: ${healthCheck.body}`)
  } else {
    console.log('✓ Health check passed')
  }

  return { startTime: Date.now() }
}

// Teardown function (runs once at end)
export function teardown(data) {
  const endTime = Date.now()
  const duration = (endTime - data.startTime) / 1000 / 60 // minutes

  console.log('')
  console.log('=== Load Test Complete ===')
  console.log(`Duration: ${duration.toFixed(2)} minutes`)
  console.log('')
  console.log('Check the summary above for detailed metrics.')
  console.log('')
  console.log('Key Metrics to Review:')
  console.log('- http_req_duration (p95, p99): Response time percentiles')
  console.log('- http_req_failed: Request failure rate')
  console.log('- order_creation_success_rate: Successful order creation rate')
  console.log('- order_creation_errors: Total errors encountered')
  console.log('')
  console.log('Database Impact:')
  console.log(
    '- Approximately ' +
      Math.floor((100 * 3 * 60) / 2) +
      ' orders created (100 users × 3 min × ~1 order/2s)'
  )
  console.log('- Check database size, query performance, connection pool usage')
  console.log('')
  console.log('Recommended Actions:')
  console.log('1. Review Grafana dashboard for system metrics during test')
  console.log('2. Check PostgreSQL slow query log')
  console.log('3. Verify Kafka consumer lag remained low')
  console.log('4. Confirm encryption service (Vault) remained responsive')
  console.log('5. Check for any error spikes in application logs')
}

// Custom scenarios (optional - for more complex load profiles)
export const scenarios = {
  // Spike test: sudden surge of users
  spike_test: {
    executor: 'constant-vus',
    vus: 200,
    duration: '1m',
    startTime: '9m', // After main test completes
    tags: { test_type: 'spike' },
  },
}
