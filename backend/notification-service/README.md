# Notification Service

Microservice for handling outbound notifications (email, push, in-app, SMS) via Kafka message queue.

## Features

- **Kafka Integration**: Consumes events from Kafka topic
- **Multi-Channel Support**: Email, Push, In-App, SMS
- **Event-Driven Architecture**: Asynchronous processing
- **Retry Mechanism**: Automatic retry on failures
- **Audit Trail**: All notifications stored in database

## Architecture

```
Other Services → Kafka Topic → Notification Service → Email/Push/SMS Providers
                                        ↓
                                   Database (Audit Log)
```

## Supported Events

### 1. User Registration (`user.registered`)
Sends email verification link when a new user registers.

**Event Data:**
```json
{
  "event_type": "user.registered",
  "tenant_id": "uuid",
  "user_id": "uuid",
  "data": {
    "email": "user@example.com",
    "name": "John Doe",
    "verification_token": "token123"
  }
}
```

### 2. User Login (`user.login`)
Sends login alert notification.

**Event Data:**
```json
{
  "event_type": "user.login",
  "tenant_id": "uuid",
  "user_id": "uuid",
  "data": {
    "email": "user@example.com",
    "name": "John Doe",
    "ip_address": "192.168.1.1",
    "user_agent": "Mozilla/5.0..."
  }
}
```

### 3. Password Reset Request (`password.reset_requested`)
Sends password reset link.

**Event Data:**
```json
{
  "event_type": "password.reset_requested",
  "tenant_id": "uuid",
  "user_id": "uuid",
  "data": {
    "email": "user@example.com",
    "name": "John Doe",
    "reset_token": "token123"
  }
}
```

### 4. Password Changed (`password.changed`)
Sends confirmation that password was changed.

**Event Data:**
```json
{
  "event_type": "password.changed",
  "tenant_id": "uuid",
  "user_id": "uuid",
  "data": {
    "email": "user@example.com",
    "name": "John Doe"
  }
}
```

## Configuration

### Environment Variables

```bash
# Server
PORT=8085

# Database
DATABASE_URL=postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=notification-events
KAFKA_GROUP_ID=notification-service-group

# Email (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@pos-system.com
```

## Running Locally

### Prerequisites
- Go 1.21+
- PostgreSQL
- Kafka & Zookeeper

### Start with Docker Compose
```bash
docker-compose up -d postgres redis zookeeper kafka
```

### Run the service
```bash
cd backend/notification-service
go run main.go
```

## Publishing Events from Other Services

### Example: Publishing user registration event

```go
import "github.com/pos/backend/src/queue"

// Initialize publisher
publisher := queue.NewEventPublisher(
    []string{"localhost:9092"},
    "notification-events",
)
defer publisher.Close()

// Publish event
err := publisher.PublishUserRegistered(
    ctx,
    tenantID,
    userID,
    "user@example.com",
    "John Doe",
    "verification-token-123",
)
```

## Database Schema

```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID,
    type VARCHAR(20) NOT NULL, -- email, push, in_app, sms
    status VARCHAR(20) NOT NULL, -- pending, queued, sent, failed, retrying
    subject VARCHAR(255),
    body TEXT NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    metadata JSONB,
    sent_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    error_msg TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
```

## Email Templates

Templates are defined in `src/services/notification_service.go`. Customize as needed:

- `registration`: Welcome email with verification link
- `login_alert`: New login notification
- `password_reset`: Password reset link
- `password_changed`: Password change confirmation

## Health Checks

```bash
# Check service health
curl http://localhost:8085/health

# Check readiness
curl http://localhost:8085/ready
```

## Development Mode

When `SMTP_USERNAME` is not configured, emails are logged to console instead of being sent:

```
[EMAIL] To: user@example.com, Subject: Welcome!
<email body>
```

## Monitoring

Monitor notification status:

```sql
-- Check notification stats
SELECT status, type, COUNT(*) 
FROM notifications 
WHERE tenant_id = 'your-tenant-id'
GROUP BY status, type;

-- Check failed notifications
SELECT * FROM notifications 
WHERE status = 'failed' 
ORDER BY created_at DESC 
LIMIT 10;
```

## Deployment

### Using Docker

```bash
# Build image
docker build -t pos-notification-service .

# Run container
docker run -d \
  -p 8085:8085 \
  -e DATABASE_URL=postgresql://... \
  -e KAFKA_BROKERS=kafka:9092 \
  -e SMTP_HOST=smtp.gmail.com \
  -e SMTP_USERNAME=your-email \
  -e SMTP_PASSWORD=your-password \
  pos-notification-service
```

### Using Kubernetes

See `k8s/notification-service.yaml` for deployment manifests.

## Troubleshooting

### No messages received from Kafka

1. Check Kafka connectivity:
```bash
docker exec -it pos-kafka kafka-topics --list --bootstrap-server localhost:9092
```

2. Check consumer group lag:
```bash
docker exec -it pos-kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --group notification-service-group \
  --describe
```

### Emails not sending

1. Verify SMTP credentials
2. Check email logs: `tail -f /var/log/notification-service.log`
3. Verify firewall allows outbound SMTP (port 587/465)

## Future Enhancements

- [ ] Support for templating engines (Handlebars, Mustache)
- [ ] Firebase Cloud Messaging (FCM) for push notifications
- [ ] Twilio integration for SMS
- [ ] In-app notification API
- [ ] Notification preferences per user
- [ ] Scheduled notifications
- [ ] Batch sending
- [ ] Rate limiting per recipient
