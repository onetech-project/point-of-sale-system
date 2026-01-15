package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

	DeletedUsersNotifiedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "deleted_users_notified_total",
		Help: "Total number of users notified about upcoming deletion",
	})

	DeletedUsersHardDeletedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "deleted_users_hard_deleted_total",
		Help: "Total number of users permanently deleted",
	})

	CleanupJobDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "cleanup_job_duration_seconds",
		Help:    "Duration of cleanup job execution",
		Buckets: prometheus.DefBuckets,
	})

	CleanupJobErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cleanup_job_errors_total",
		Help: "Total number of cleanup job errors",
	})
)

func init() {
	prometheus.MustRegister(HttpRequestsTotal, HttpRequestDuration)
}
