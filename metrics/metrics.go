package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Transaction metrics
	TransactionCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nmi_transactions_total",
			Help: "Total number of transactions processed",
		},
		[]string{"type", "status"},
	)

	TransactionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nmi_transaction_duration_seconds",
			Help:    "Transaction processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	// Error metrics
	ErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nmi_errors_total",
			Help: "Total number of errors encountered",
		},
		[]string{"type", "error_type"},
	)

	// API Request metrics
	RequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nmi_requests_in_flight",
			Help: "Current number of API requests being processed",
		},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nmi_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	ResponseStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nmi_http_responses_total",
			Help: "Total number of HTTP responses sent, by status code",
		},
		[]string{"code"},
	)

	// Vault metrics
	VaultOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nmi_vault_operations_total",
			Help: "Total number of customer vault operations",
		},
		[]string{"operation", "status"},
	)

	// Recurring payment metrics
	RecurringPayments = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nmi_recurring_payments_total",
			Help: "Total number of recurring payment operations",
		},
		[]string{"operation", "status"},
	)
)

func init() {
	// Register all metrics with Prometheus
	prometheus.MustRegister(
		TransactionCounter,
		TransactionDuration,
		ErrorCounter,
		RequestsInFlight,
		RequestDuration,
		ResponseStatus,
		VaultOperations,
		RecurringPayments,
	)
}

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

// RecordVaultOperation records customer vault operations
func RecordVaultOperation(operation, status string) {
	VaultOperations.WithLabelValues(operation, status).Inc()
}

// RecordRecurringPayment records recurring payment operations
func RecordRecurringPayment(operation, status string) {
	RecurringPayments.WithLabelValues(operation, status).Inc()
}
