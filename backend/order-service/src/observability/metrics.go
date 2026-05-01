package observability

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
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

	// T112: Business metrics for offline orders
	OfflineOrdersTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "offline_orders_total",
			Help: "Total number of offline orders created by status",
		},
		[]string{"status", "tenant_id"},
	)

	OfflineOrderRevenue = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "offline_order_revenue_total",
			Help: "Total revenue from offline orders in cents",
		},
		[]string{"tenant_id"},
	)

	PaymentInstallmentsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_installments_total",
			Help: "Total number of payment installments created",
		},
		[]string{"tenant_id", "installment_count"},
	)

	OfflineOrderCreationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "offline_order_creation_duration_seconds",
			Help:    "Duration of offline order creation in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"tenant_id"},
	)

	OfflineOrderPaymentsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "offline_order_payments_total",
			Help: "Total number of payments recorded for offline orders",
		},
		[]string{"tenant_id", "payment_type"},
	)

	OfflineOrderUpdatesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "offline_order_updates_total",
			Help: "Total number of offline order updates",
		},
		[]string{"tenant_id"},
	)

	OfflineOrderDeletionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "offline_order_deletions_total",
			Help: "Total number of offline order deletions",
		},
		[]string{"tenant_id", "user_role"},
	)
)

func init() {
	prometheus.MustRegister(
		HttpRequestsTotal,
		HttpRequestDuration,
		// T112: Register offline order business metrics
		OfflineOrdersTotal,
		OfflineOrderRevenue,
		PaymentInstallmentsTotal,
		OfflineOrderCreationDuration,
		OfflineOrderPaymentsTotal,
		OfflineOrderUpdatesTotal,
		OfflineOrderDeletionsTotal,
	)
}

