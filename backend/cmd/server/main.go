package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/suma-iya/portfolio-backend/internal/config"
	"github.com/suma-iya/portfolio-backend/internal/logger"
	"github.com/suma-iya/portfolio-backend/internal/middleware"
	"github.com/suma-iya/portfolio-backend/internal/routes"
)

func main() {
	cfg := config.Load()
	log := logger.New()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	// Only trust X-Forwarded-For from Nginx on the same host. Without this, Gin
	// trusts the header from anyone, and clients could spoof their IP to dodge
	// the per-IP rate limit.
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Error("invalid TRUSTED_PROXIES", "error", err)
		os.Exit(1)
	}
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(log))
	router.Use(middleware.CORS(cfg.AllowedOrigins))

	routes.Register(router, cfg, log)

	srv := &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second, // SMTP send happens inside the request
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("starting server", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown so `systemctl restart` during a deploy doesn't cut off
	// a contact-form request mid-send.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}
}
