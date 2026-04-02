package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/agm"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/billing"
	"github.com/stratahq/backend/internal/communications"
	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/documents"
	"github.com/stratahq/backend/internal/financials"
	"github.com/stratahq/backend/internal/invitation"
	"github.com/stratahq/backend/internal/levy"
	"github.com/stratahq/backend/internal/maintenance"
	"github.com/stratahq/backend/internal/notification"
	"github.com/stratahq/backend/internal/platform/cache"
	"github.com/stratahq/backend/internal/platform/database"
	"github.com/stratahq/backend/internal/platform/health"
	"github.com/stratahq/backend/internal/scheme"
	"github.com/stratahq/backend/internal/server"
)

// redisChecker adapts *redis.Client to health.Checker.
type redisChecker struct{ c *redis.Client }

func (r *redisChecker) Ping(ctx context.Context) error {
	return r.c.Ping(ctx).Err()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load .env in development (no-op if file doesn't exist or vars already set)
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Database
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connected")

	// Redis
	rdb, err := cache.New(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("redis connected")

	// Notification
	emailClient := notification.NewEmailClient(cfg.ResendAPIKey, cfg.EmailFrom)

	// Services
	authService := auth.NewService(db, rdb, emailClient, cfg.JWTSecret, cfg.AppBaseURL, cfg.JWTExpiry, cfg.RefreshExpiry)
	agmService := agm.NewService(db)
	schemeService := scheme.NewService(db)
	communicationsService := communications.NewService(db)
	documentsService := documents.NewService(db)
	financialsService := financials.NewService(db)
	levyService := levy.NewService(db)
	maintenanceService := maintenance.NewService(db)
	billingService := billing.NewService(db)
	invitationService := invitation.NewService(db, emailClient, cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry)

	// Handlers
	handlers := server.Handlers{
		Health:         health.New(db, &redisChecker{rdb}),
		Auth:           auth.NewHandler(authService),
		Agm:            agm.NewHandler(agmService),
		Scheme:         scheme.NewHandler(schemeService),
		Communications: communications.NewHandler(communicationsService),
		Documents:      documents.NewHandler(documentsService),
		Financials:     financials.NewHandler(financialsService),
		Levy:           levy.NewHandler(levyService),
		Maintenance:    maintenance.NewHandler(maintenanceService),
		Billing:        billing.NewHandler(billingService),
		Invitation:     invitation.NewHandler(invitationService, cfg.AppBaseURL),
	}

	// Router & Server
	router := server.NewRouter(cfg, logger, rdb, handlers)
	srv := server.New(router, cfg.Port, logger)

	logger.Info("starting server", "port", cfg.Port, "env", cfg.Env)
	if err := srv.Start(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
