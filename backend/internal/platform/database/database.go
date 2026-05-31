package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/stratahq/backend/db/gen"
)

// Keep this aligned with the latest migration in backend/db/migrations.
const minimumMigrationVersion = 32

// Pool wraps pgxpool.Pool together with a ready-to-use *dbgen.Queries so
// callers only need to carry one value.
type Pool struct {
	*pgxpool.Pool
	Q *dbgen.Queries
}

// New creates and validates a pgx connection pool with production-ready
// defaults, then returns a Pool that includes the sqlc query layer.
func New(ctx context.Context, databaseURL string) (*Pool, error) {
	cfg, err := newPoolConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Pool{
		Pool: pool,
		Q:    dbgen.New(pool),
	}, nil
}

func newPoolConfig(databaseURL string) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	// Sensible defaults — override via DATABASE_URL query params if needed.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	return cfg, nil
}

// Ping implements health.Checker so *Pool can be passed directly to the
// health handler.
func (p *Pool) Ping(ctx context.Context) error {
	return p.Pool.Ping(ctx)
}

func (p *Pool) CheckMigrations(ctx context.Context) error {
	const query = "SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version"

	var version int64
	if err := p.QueryRow(ctx, query).Scan(&version); err != nil {
		return fmt.Errorf("database migration check failed: %w", err)
	}

	if version < minimumMigrationVersion {
		return fmt.Errorf("database migration check failed: expected version %d, got %d", minimumMigrationVersion, version)
	}

	return nil
}
