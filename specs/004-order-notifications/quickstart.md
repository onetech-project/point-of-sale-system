# Quickstart — Run locally (developer)

This quickstart shows the minimal components to run locally for development and manual verification of the Order Notifications feature.

Prerequisites:
- Docker & Docker Compose
- `make` (optional)

1) Start dependent services (Postgres, Kafka, Redis)

```bash
docker-compose up -d postgres kafka zookeeper redis
```

2) Start `notification-service` (local dev)

```bash
# from repo root
cd backend/notification-service
go run ./main.go
```

3) Start frontend (to connect to SSE)

```bash
cd frontend
npm install
npm run dev
```

4) Produce a test event to Kafka (order_paid)

Use the existing kafka tooling or a simple script to publish a message to `orders.events`. Example JSON payload:

```json
{
  "event_id": "11111111-2222-3333-4444-555555555555",
  "event_type": "order_paid",
  "order_id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
  "tenant_id": "tttttttt-tttt-tttt-tttt-tttttttttttt",
  "timestamp": "2025-12-09T12:34:56Z",
  "payload": { "reference": "INV-00123", "total_amount": 100000 }
}
```

5) Verify
- Open frontend and confirm an in-app toast appears.  
- Check `notification-service` logs to see the event processed and email queued or printed (dev mode).  
- Use `redis-cli` to inspect tenant channels if using Redis for fan-out.
