package middleware

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	metricsOnce     sync.Once
)

// PrometheusMiddleware holds prometheus metrics collectors for HTTP traffic.
type PrometheusMiddleware struct {
	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// NewPrometheusMiddleware creates a new PrometheusMiddleware registered to reg.
// If the metrics are already registered (e.g. in test suites), it reuses existing collectors safely.
func NewPrometheusMiddleware(reg prometheus.Registerer) (*PrometheusMiddleware, error) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	var errCount, errDuration error

	metricsOnce.Do(func() {
		requestCount = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests processed.",
			},
			[]string{"method", "path", "status"},
		)

		requestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		)

		errCount = reg.Register(requestCount)
		errDuration = reg.Register(requestDuration)
	})

	if errCount != nil {
		var are prometheus.AlreadyRegisteredError
		if !errors.As(errCount, &are) {
			return nil, errCount
		}
		requestCount = are.ExistingCollector.(*prometheus.CounterVec)
	}

	if errDuration != nil {
		var are prometheus.AlreadyRegisteredError
		if !errors.As(errDuration, &are) {
			return nil, errDuration
		}
		requestDuration = are.ExistingCollector.(*prometheus.HistogramVec)
	}

	return &PrometheusMiddleware{
		requestCount:    requestCount,
		requestDuration: requestDuration,
	}, nil
}

// Handler returns the Gin middleware handler.
func (m *PrometheusMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Exclude /metrics from being counted
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()

		path := c.Request.URL.Path
		route := c.FullPath()
		if route == "" {
			route = "UNMATCHED"
		}

		statusStr := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		m.requestCount.WithLabelValues(
			method,
			path,
			statusStr,
		).Inc()

		m.requestDuration.WithLabelValues(
			method,
			route,
			statusStr,
		).Observe(duration)
	}
}

