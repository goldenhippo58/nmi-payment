package middleware

import (
	"context"
	"net/http"
	"time"

	"nmi-pay-int/metrics" // Make sure this matches your module name

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

// SecurityMiddleware handles rate limiting and security measures
type SecurityMiddleware struct {
	limiter *rate.Limiter
}

// NewSecurityMiddleware creates a new security middleware instance
func NewSecurityMiddleware(requestsPerMinute float64) *SecurityMiddleware {
	return &SecurityMiddleware{
		limiter: rate.NewLimiter(rate.Limit(requestsPerMinute/60), 1),
	}
}

// RateLimiter implements rate limiting middleware
func (m *SecurityMiddleware) RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.limiter.Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			metrics.RecordErrorMetrics("rate_limit", "too_many_requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MetricsMiddleware adds prometheus metrics tracking
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		metrics.IncrementRequestsInFlight()
		defer metrics.DecrementRequestsInFlight()

		// Create a custom response writer to capture the status code
		rw := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		metrics.RecordRequestMetrics(
			r.Method,
			path,
			duration,
			string(rw.statusCode),
		)
	})
}

// TimeoutMiddleware adds request timeout
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			done := make(chan bool)
			go func() {
				next.ServeHTTP(w, r)
				done <- true
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				w.WriteHeader(http.StatusGatewayTimeout)
				metrics.RecordErrorMetrics("timeout", "request_timeout")
				return
			}
		})
	}
}

// LoggingMiddleware adds request logging
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log incoming request
		metrics.LogInfo("Incoming request: " + r.Method + " " + r.URL.Path)

		// Create a custom response writer to capture the status code
		rw := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(rw, r)

		// Log response
		duration := time.Since(start)
		metrics.LogDebug("Request completed: " + r.Method + " " + r.URL.Path +
			" Status: " + string(rw.statusCode) +
			" Duration: " + duration.String())
	})
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		ctx := context.WithValue(r.Context(), "requestID", requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" +
		string(time.Now().Nanosecond())
}
