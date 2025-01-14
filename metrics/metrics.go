package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Transaction metrics
	TransactionCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nmi_transactions_total",
			Help: "Total number of transactions processed",
		},
		[]string{"type", "status"},
	)

	TransactionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nmi_transaction_duration_seconds",
			Help:    "Transaction processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	// Error metrics
	ErrorCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nmi_errors_total",
			Help: "Total number of errors encountered",
		},
		[]string{"type", "error_type"},
	)

	// API request metrics
	RequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nmi_requests_in_flight",
			Help: "Current number of API requests being processed",
		},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nmi_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	ResponseStatus = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nmi_http_responses_total",
			Help: "Total number of HTTP responses sent, by status code",
		},
		[]string{"code"},
	)
)

// RecordTransactionMetrics records metrics for a transaction
func RecordTransactionMetrics(txType, status string, duration float64) {
	TransactionCounter.WithLabelValues(txType, status).Inc()
	TransactionDuration.WithLabelValues(txType).Observe(duration)
}

// RecordErrorMetrics records error metrics
func RecordErrorMetrics(txType, errorType string) {
	ErrorCounter.WithLabelValues(txType, errorType).Inc()
}

// RecordRequestMetrics records HTTP request metrics
func RecordRequestMetrics(method, endpoint string, duration float64, statusCode string) {
	RequestDuration.WithLabelValues(method, endpoint).Observe(duration)
	ResponseStatus.WithLabelValues(statusCode).Inc()
}

// IncrementRequestsInFlight increments the in-flight requests counter
func IncrementRequestsInFlight() {
	RequestsInFlight.Inc()
}

// DecrementRequestsInFlight decrements the in-flight requests counter
func DecrementRequestsInFlight() {
	RequestsInFlight.Dec()
}
