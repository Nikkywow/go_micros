package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	TotalRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	RequestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_errors_total",
			Help: "Total number of request errors (4xx and 5xx)",
		},
		[]string{"method", "endpoint", "status"},
	)
)

func init() {
	prometheus.MustRegister(TotalRequests)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(RequestErrors)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		endpoint := routePath(r)
		status := strconv.Itoa(sw.status)

		TotalRequests.WithLabelValues(r.Method, endpoint, status).Inc()
		RequestDuration.WithLabelValues(r.Method, endpoint).Observe(time.Since(start).Seconds())
		if sw.status >= 400 {
			RequestErrors.WithLabelValues(r.Method, endpoint, status).Inc()
		}
	})
}

func routePath(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if route == nil {
		return r.URL.Path
	}
	tpl, err := route.GetPathTemplate()
	if err != nil || tpl == "" {
		return r.URL.Path
	}
	return tpl
}
