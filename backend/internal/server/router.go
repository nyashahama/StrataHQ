package server

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/middleware"
	"github.com/stratahq/backend/internal/platform/health"
)

func NewRouter(cfg *config.Config, logger *slog.Logger, healthHandler *health.Handler, rdb *redis.Client) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware stack
	r.Use(middleware.Recover)
	r.Use(middleware.Metrics)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RateLimit(rdb, 100, 1*time.Minute))

	// Health & metrics (outside /api/v1, no auth)
	r.Get("/healthz", healthHandler.Healthz)
	r.Get("/readyz", healthHandler.Readyz)
	r.Handle("/metrics", promhttp.Handler())

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			// Auth routes will be mounted here
			// Stripe webhook routes will be mounted here
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			// Domain routes will be mounted here:
			// r.Mount("/schemes", scheme.Routes())
			// r.Mount("/levies", levy.Routes())
			// r.Mount("/maintenance", maintenance.Routes())
			// r.Mount("/billing", billing.Routes())
		})
	})

	return r
}
