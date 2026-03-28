//go:build integration

package integration

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	testDB    *pgxpool.Pool
	testRedis *redis.Client
	testLog   *slog.Logger
)

// redisChecker adapts *redis.Client to health.Checker.
type redisChecker struct{ c *redis.Client }

func (r *redisChecker) Ping(ctx context.Context) error {
	return r.c.Ping(ctx).Err()
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	testLog = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://stratahq:stratahq@localhost:5432/stratahq?sslmode=disable"
	}

	var err error
	testDB, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		testLog.Error("failed to connect to test database", "error", err)
		os.Exit(1)
	}
	defer testDB.Close()

	// Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		testLog.Error("failed to parse redis URL", "error", err)
		os.Exit(1)
	}
	testRedis = redis.NewClient(opts)
	defer testRedis.Close()

	os.Exit(m.Run())
}
