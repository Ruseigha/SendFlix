package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Ruseigha/SendFlix/pkg/logger"
	"github.com/gin-gonic/gin"

	emailuc "github.com/Ruseigha/SendFlix/internal/usecase/email"
	templateuc "github.com/Ruseigha/SendFlix/internal/usecase/template"
)

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// UseCases contains all use cases
type UseCases struct {
	SendEmail        *emailuc.SendEmailUseCase
	SendBulkEmail    *emailuc.SendBulkEmailUseCase
	ScheduleEmail    *emailuc.ScheduleEmailUseCase
	GetEmail         *emailuc.GetEmailUseCase
	CreateTemplate   *templateuc.CreateTemplateUseCase
	GetTemplate      *templateuc.GetTemplateUseCase
	UpdateTemplate   *templateuc.UpdateTemplateUseCase
	DeleteTemplate   *templateuc.DeleteTemplateUseCase
	ActivateTemplate *templateuc.ActivateTemplateUseCase
	PreviewTemplate  *templateuc.PreviewTemplateUseCase
}

// Server represents HTTP server
type Server struct {
	config   ServerConfig
	router   *gin.Engine
	useCases UseCases
	logger   logger.Logger
}

// NewServer creates new HTTP server
func NewServer(
	config ServerConfig,
	useCases UseCases,
	logger logger.Logger,
) *Server {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	server := &Server{
		config:   config,
		router:   router,
		useCases: useCases,
		logger:   logger,
	}

	// Setup middleware
	server.setupMiddleware()

	// Setup routes
	server.setupRoutes()

	return server
}

// setupMiddleware sets up middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logger middleware
	s.router.Use(s.loggerMiddleware())

	// CORS middleware
	s.router.Use(s.corsMiddleware())
}

// setupRoutes sets up HTTP routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthCheck)

	// API v1
	v1 := s.router.Group("/api/v1")
	{
		// Email routes
		emails := v1.Group("/emails")
		{
			emails.POST("", s.sendEmail)
			emails.POST("/bulk", s.sendBulkEmails)
			emails.POST("/scheduled", s.scheduleEmail)
			emails.GET("/:id", s.getEmail)
			emails.GET("", s.listEmails)
			emails.DELETE("/:id/cancel", s.cancelScheduledEmail)
		}

		// Template routes
		templates := v1.Group("/templates")
		{
			templates.POST("", s.createTemplate)
			templates.GET("", s.listTemplates)
			templates.GET("/:id", s.getTemplate)
			templates.PUT("/:id", s.updateTemplate)
			templates.DELETE("/:id", s.deleteTemplate)
			templates.POST("/:id/activate", s.activateTemplate)
			templates.POST("/:id/deactivate", s.deactivateTemplate)
			templates.POST("/:id/preview", s.previewTemplate)
		}
	}
}

// HTTPServer returns configured HTTP server
func (s *Server) HTTPServer() *http.Server {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	return &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "sendflix",
		"version": "1.0.0",
	})
}