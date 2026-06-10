package telemetry

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware returns a Gin middleware that records HTTP request counts and durations.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		
		// Use full route path schema (e.g. /payments/:id) to prevent high-cardinality label explosion
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		method := c.Request.Method

		// Update Prometheus metrics
		HttpRequestsTotal.WithLabelValues(path, method, status).Inc()
		HttpRequestDuration.WithLabelValues(path, method).Observe(duration)
	}
}
