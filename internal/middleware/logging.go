package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

func AccessLog(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		attrs := []any{
			"request_id", c.GetString("request_id"),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		}

		spanContext := trace.SpanContextFromContext(c.Request.Context())
		if spanContext.IsValid() {
			traceID := spanContext.TraceID().String()
			spanID := spanContext.SpanID().String()
			c.Set("trace_id", traceID)
			c.Set("span_id", spanID)
			if c.Writer.Header().Get("X-Trace-ID") == "" {
				c.Writer.Header().Set("X-Trace-ID", traceID)
			}
			attrs = append(attrs, "trace_id", traceID, "span_id", spanID)
		}

		logger.Info("http request", attrs...)
	}
}
