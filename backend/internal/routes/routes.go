package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/suma-iya/portfolio-backend/internal/config"
	"github.com/suma-iya/portfolio-backend/internal/handlers"
	"github.com/suma-iya/portfolio-backend/internal/logger"
	"github.com/suma-iya/portfolio-backend/internal/middleware"
	"github.com/suma-iya/portfolio-backend/internal/services"
)

func Register(router *gin.Engine, cfg *config.Config, log *logger.Logger) {
	mailer := services.NewMailer(cfg)
	contactHandler := handlers.NewContactHandler(mailer, log)
	contactLimiter := middleware.NewIPRateLimiter(cfg.RateLimitPerMinute)

	router.GET("/health", handlers.HealthCheck)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/contact", contactLimiter.Middleware(), contactHandler.Submit)
	}
}
