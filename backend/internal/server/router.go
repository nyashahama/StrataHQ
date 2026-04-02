package server

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/billing"
	"github.com/stratahq/backend/internal/communications"
	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/invitation"
	"github.com/stratahq/backend/internal/levy"
	"github.com/stratahq/backend/internal/maintenance"
	"github.com/stratahq/backend/internal/middleware"
	"github.com/stratahq/backend/internal/platform/health"
	"github.com/stratahq/backend/internal/scheme"
)

type Handlers struct {
	Health         *health.Handler
	Auth           *auth.Handler
	Scheme         *scheme.Handler
	Communications *communications.Handler
	Levy           *levy.Handler
	Maintenance    *maintenance.Handler
	Billing        *billing.Handler
	Invitation     *invitation.Handler
}

func NewRouter(cfg *config.Config, logger *slog.Logger, rdb *redis.Client, h Handlers) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware stack
	r.Use(middleware.Recover)
	r.Use(middleware.Metrics)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RateLimit(rdb, 100, 1*time.Minute))

	// Health & metrics (outside /api/v1, no auth)
	r.Get("/healthz", h.Health.Healthz)
	r.Get("/readyz", h.Health.Readyz)
	r.Handle("/metrics", promhttp.Handler())

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			r.Mount("/auth", h.Auth.Routes())
			r.Mount("/billing/webhooks", h.Billing.WebhookRoutes())
			r.Mount("/invitations/verify", h.Invitation.PublicRoutes())
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Get("/auth/me", h.Auth.Me)
			r.Patch("/auth/profile", h.Auth.UpdateProfile)
			r.Patch("/auth/org", h.Auth.UpdateOrg)
			r.Post("/auth/change-password", h.Auth.ChangePassword)
			r.Mount("/onboarding", h.Auth.OnboardingRoutes())
			r.Mount("/invitations", h.Invitation.ProtectedRoutes())
			r.Mount("/schemes", h.Scheme.Routes())
			r.Mount("/communications", h.Communications.Routes())
			r.Mount("/levies", h.Levy.Routes())
			r.Mount("/maintenance", h.Maintenance.Routes())
			r.Mount("/billing", h.Billing.Routes())
		})
	})

	return r
}
