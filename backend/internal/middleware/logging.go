package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/suma-iya/portfolio-backend/internal/logger"
)

// RequestLogger emits one structured log line per request.
func RequestLogger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		log.Info("request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}
