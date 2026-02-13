package httpapi

import (
	"time"

	"github.com/Ruseigha/SendFlix/pkg/errors"
	"github.com/gin-gonic/gin"
)

// loggerMiddleware logs HTTP requests
func (s *Server) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Log
		duration := time.Since(start)
		status := c.Writer.Status()

		s.logger.Info("http request",
			"method", method,
			"path", path,
			"status", status,
			"duration", duration,
			"ip", c.ClientIP(),
		)
	}
}

// corsMiddleware handles CORS
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// handleUseCaseError handles use case errors
func (s *Server) handleUseCaseError(c *gin.Context, err error) {
	if appErr, ok := err.(*errors.AppError); ok {
		s.respondError(c, appErr.StatusCode, appErr.Code, appErr.Message)
		return
	}

	s.respondError(c, 500, "INTERNAL_ERROR", err.Error())
}

// respondError sends error response
func (s *Server) respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}