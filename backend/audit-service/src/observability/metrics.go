package observability

import (
	"github.com/prometheus/client_golang/prometheus"
)

// T116: Audit trail metrics for monitoring UU PDP compliance

var (
	// AuditEventsPublishedTotal tracks total audit events published by services to Kafka
	AuditEventsPublishedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audit_events_published_total",
			Help: "Total number of audit events published to Kafka by service",
		},
		[]string{"service", "action", "resource_type"},
	)

	// AuditEventsPersistedTotal tracks total audit events successfully persisted to database
	AuditEventsPersistedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audit_events_persisted_total",
			Help: "Total number of audit events persisted to database",
		},
		[]string{"action", "resource_type", "status"}, // status: success or error
	)

	// AuditEventsPersistErrors tracks failed audit event persistence attempts
	AuditEventsPersistErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audit_events_persist_errors_total",
			Help: "Total number of audit event persistence errors",
		},
		[]string{"error_type"},
	)

	// AuditEventsProcessingDuration measures time to process and persist audit events
	AuditEventsProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "audit_events_processing_duration_seconds",
			Help:    "Time taken to process and persist audit events from Kafka",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"action", "resource_type"},
	)

	// AuditKafkaConsumerLag tracks Kafka consumer lag (messages behind)
	AuditKafkaConsumerLag = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "audit_kafka_consumer_lag",
			Help: "Number of messages behind in Kafka audit topic",
		},
	)

	// AuditKafkaConsumerOffset tracks current consumer offset
	AuditKafkaConsumerOffset = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "audit_kafka_consumer_offset",
			Help: "Current Kafka consumer offset for audit events",
		},
	)

	// AuditPartitionsTotal tracks number of audit_events partitions
	AuditPartitionsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "audit_partitions_total",
			Help: "Total number of audit_events table partitions",
		},
	)

	// HTTP metrics (inherited from other services)
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	// Register all metrics with Prometheus
	prometheus.MustRegister(
		AuditEventsPublishedTotal,
		AuditEventsPersistedTotal,
		AuditEventsPersistErrorsTotal,
		AuditEventsProcessingDuration,
		AuditKafkaConsumerLag,
		AuditKafkaConsumerOffset,
		AuditPartitionsTotal,
		HttpRequestsTotal,
		HttpRequestDuration,
	)
}
