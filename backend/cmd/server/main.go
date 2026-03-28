package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/platform/cache"
	"github.com/stratahq/backend/internal/platform/database"
	"github.com/stratahq/backend/internal/platform/health"
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

	// Health
	healthHandler := health.New(db, &redisChecker{rdb})

	// Router & Server
	router := server.NewRouter(cfg, logger, healthHandler, rdb)
	srv := server.New(router, cfg.Port, logger)

	logger.Info("starting server", "port", cfg.Port, "env", cfg.Env)
	if err := srv.Start(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
