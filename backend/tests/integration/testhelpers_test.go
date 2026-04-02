//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	dbgen "github.com/stratahq/backend/db/gen"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/database"
)

var (
	testDB    *pgxpool.Pool
	testPool  *database.Pool
	testQ     *dbgen.Queries
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
	testPool = &database.Pool{Pool: testDB, Q: dbgen.New(testDB)}
	testQ = testPool.Q
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

type successEnvelope[T any] struct {
	Data T `json:"data"`
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeSuccess[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var envelope successEnvelope[T]
	if err := json.NewDecoder(recorder.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode success envelope: %v", err)
	}
	return envelope.Data
}

func decodeError(t *testing.T, recorder *httptest.ResponseRecorder) errorEnvelope {
	t.Helper()
	var envelope errorEnvelope
	if err := json.NewDecoder(recorder.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	return envelope
}

func withAuthContext(r *http.Request, accessToken, jwtSecret string) *http.Request {
	tClaims, err := auth.ValidateAccessToken(accessToken, jwtSecret)
	if err != nil {
		panic("withAuthContext: invalid token: " + err.Error())
	}
	return r.WithContext(auth.ContextWithClaims(r.Context(), tClaims))
}

func withNonAdminContext(r *http.Request) *http.Request {
	return r.WithContext(auth.ContextWithIdentity(
		r.Context(),
		"00000000-0000-0000-0000-000000000001",
		"00000000-0000-0000-0000-000000000002",
		string(auth.RoleTrustee),
	))
}

func withOrgRoleContext(r *http.Request, orgID, role string) *http.Request {
	return r.WithContext(auth.ContextWithIdentity(
		r.Context(),
		"00000000-0000-0000-0000-000000000003",
		orgID,
		role,
	))
}
