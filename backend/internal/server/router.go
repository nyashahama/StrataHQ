package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/agm"
	"github.com/stratahq/backend/internal/ai"
	"github.com/stratahq/backend/internal/audit"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/billing"
	"github.com/stratahq/backend/internal/communications"
	"github.com/stratahq/backend/internal/compliance"
	"github.com/stratahq/backend/internal/contractors"
	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/documents"
	"github.com/stratahq/backend/internal/earlyaccess"
	"github.com/stratahq/backend/internal/financials"
	"github.com/stratahq/backend/internal/integrations"
	"github.com/stratahq/backend/internal/invitation"
	"github.com/stratahq/backend/internal/levy"
	"github.com/stratahq/backend/internal/maintenance"
	"github.com/stratahq/backend/internal/middleware"
	"github.com/stratahq/backend/internal/platform/health"
	"github.com/stratahq/backend/internal/scheme"
	"github.com/stratahq/backend/internal/whatsapp"
)

type Handlers struct {
	Health          *health.Handler
	Auth            *auth.Handler
	Audit           *audit.Handler
	Agm             *agm.Handler
	AI              *ai.Handler
	Scheme          *scheme.Handler
	Compliance      *compliance.Handler
	Communications  *communications.Handler
	Documents       *documents.Handler
	Financials      *financials.Handler
	Levy            *levy.Handler
	Maintenance     *maintenance.Handler
	WhatsApp        *whatsapp.Handler
	WhatsAppWebhook *whatsapp.WebhookHandler
	Billing         *billing.Handler
	Contractors     *contractors.Handler
	Invitation      *invitation.Handler
	EarlyAccess     *earlyaccess.Handler
	Integrations    *integrations.Handler
}

func NewRouter(cfg *config.Config, logger *slog.Logger, rdb *redis.Client, auditRecorder audit.Recorder, h Handlers) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware stack
	r.Use(middleware.Recover)
	r.Use(middleware.RequestID)
	r.Use(middleware.Metrics)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RateLimit(rdb, 100, 1*time.Minute))
	r.Use(middleware.MaxBodyHandler())

	// Health & metrics (outside /api/v1, no auth)
	r.Get("/healthz", h.Health.Healthz)
	r.Get("/readyz", h.Health.Readyz)
	if cfg.MetricsToken != "" {
		r.Group(func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer "+cfg.MetricsToken {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					next.ServeHTTP(w, r)
				})
			})
			r.Handle("/metrics", promhttp.Handler())
		})
	} else {
		r.Handle("/metrics", promhttp.Handler())
	}

	// Open API routes (public, rate-limited, outside /api/v1)
	r.Group(func(r chi.Router) {
		r.Use(middleware.PerEndpointRateLimit(rdb, "open-api", 300, 1*time.Minute))
		r.Mount("/api/open/v1", h.Integrations.OpenRoutes())
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			// Auth endpoints with per-endpoint rate limiting
			r.Route("/auth", func(r chi.Router) {
				r.Use(middleware.AuditEvents(auditRecorder, logger))
				r.With(middleware.PerEndpointRateLimit(rdb, "auth-login", 5, 1*time.Minute)).Post("/login", h.Auth.Login)
				r.With(middleware.PerEndpointRateLimit(rdb, "auth-refresh", 30, 1*time.Minute)).Post("/refresh", h.Auth.Refresh)
				r.Post("/register", h.Auth.Register)
				r.Post("/logout", h.Auth.Logout)
				r.Post("/forgot-password", h.Auth.ForgotPassword)
				r.Post("/reset-password", h.Auth.ResetPassword)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.PerEndpointRateLimit(rdb, "stripe-webhook", 60, 1*time.Minute))
				r.Mount("/billing/webhooks", h.Billing.WebhookRoutes())
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.PerEndpointRateLimit(rdb, "twilio-webhook", 120, 1*time.Minute))
				r.Mount("/whatsapp/webhooks", h.WhatsAppWebhook.Routes())
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.AuditEvents(auditRecorder, logger))
				r.Mount("/invitations/verify", h.Invitation.PublicRoutes())
			})
			r.Route("/early-access", func(r chi.Router) {
				r.Use(middleware.PerEndpointRateLimit(rdb, "earlyaccess", 3, 1*time.Minute))
				r.Use(middleware.AuditEvents(auditRecorder, logger))
				r.Mount("/", h.EarlyAccess.PublicRoutes())
			})
		})

		// AI routes with dedicated rate limit (rate limited before auth)
		r.Group(func(r chi.Router) {
			r.Use(middleware.PerEndpointRateLimit(rdb, "ai-copilot", 30, 1*time.Minute))
			r.Use(middleware.Auth(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience))
			r.Use(middleware.AuditEvents(auditRecorder, logger))
			r.Mount("/ai", h.AI.Routes())
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience))
			r.Use(middleware.AuditEvents(auditRecorder, logger))
			r.Get("/auth/me", h.Auth.Me)
			r.Patch("/auth/profile", h.Auth.UpdateProfile)
			r.Patch("/auth/org", h.Auth.UpdateOrg)
			r.Post("/auth/change-password", h.Auth.ChangePassword)
			r.Mount("/onboarding", h.Auth.OnboardingRoutes())
			r.Mount("/invitations", h.Invitation.ProtectedRoutes())
			r.Mount("/agm", h.Agm.Routes())
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
			r.Mount("/contractors", h.Contractors.Routes())
			r.Mount("/audit", h.Audit.Routes())
			r.Mount("/integrations/api-clients", h.Integrations.AdminRoutes())
		})
	})

	return r
}
