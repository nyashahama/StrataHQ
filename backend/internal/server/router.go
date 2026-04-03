package server

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/agm"
	"github.com/stratahq/backend/internal/ai"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/billing"
	"github.com/stratahq/backend/internal/communications"
	"github.com/stratahq/backend/internal/compliance"
	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/documents"
	"github.com/stratahq/backend/internal/earlyaccess"
	"github.com/stratahq/backend/internal/financials"
	"github.com/stratahq/backend/internal/invitation"
	"github.com/stratahq/backend/internal/levy"
	"github.com/stratahq/backend/internal/maintenance"
	"github.com/stratahq/backend/internal/middleware"
	"github.com/stratahq/backend/internal/platform/health"
	"github.com/stratahq/backend/internal/scheme"
	"github.com/stratahq/backend/internal/whatsapp"
)

type Handlers struct {
	Health         *health.Handler
	Auth           *auth.Handler
	Agm            *agm.Handler
	AI             *ai.Handler
	Scheme         *scheme.Handler
	Compliance     *compliance.Handler
	Communications *communications.Handler
	Documents      *documents.Handler
	Financials     *financials.Handler
	Levy           *levy.Handler
	Maintenance    *maintenance.Handler
	WhatsApp       *whatsapp.Handler
	Billing        *billing.Handler
	Invitation     *invitation.Handler
	EarlyAccess    *earlyaccess.Handler
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
			r.Mount("/early-access", h.EarlyAccess.PublicRoutes())
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
			r.Mount("/agm", h.Agm.Routes())
			r.Mount("/ai", h.AI.Routes())
			r.Mount("/schemes", h.Scheme.Routes())
			r.Mount("/compliance", h.Compliance.Routes())
			r.Mount("/communications", h.Communications.Routes())
			r.Mount("/documents", h.Documents.Routes())
			r.Mount("/financials", h.Financials.Routes())
			r.Mount("/levies", h.Levy.Routes())
			r.Mount("/maintenance", h.Maintenance.Routes())
			r.Mount("/whatsapp", h.WhatsApp.Routes())
			r.Mount("/billing", h.Billing.Routes())
			r.Mount("/admin/early-access", h.EarlyAccess.ProtectedRoutes())
		})
	})

	return r
}
