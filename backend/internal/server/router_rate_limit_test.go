package server

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/audit"
	"github.com/stratahq/backend/internal/config"
)

func TestRouterAppliesDistinctRateLimitPrefixesForLoginAndRefresh(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.New(slog.NewJSONHandler(&nopWriter{}, &slog.HandlerOptions{}))

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	ctx := context.Background()

	existingKeys, _ := rdb.Keys(ctx, "ratelimit:*").Result()
	if len(existingKeys) > 0 {
		rdb.Del(ctx, existingKeys...)
	}

	router := NewRouter(cfg, logger, rdb, &silentRecorder{}, Handlers{})

	testIP := "192.0.2.1"

	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	reqLogin.RemoteAddr = testIP + ":12345"
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)

	reqRefresh := httptest.NewRequest("POST", "/api/v1/auth/refresh", nil)
	reqRefresh.RemoteAddr = testIP + ":12345"
	wRefresh := httptest.NewRecorder()
	router.ServeHTTP(wRefresh, reqRefresh)

	keys, _ := rdb.Keys(ctx, "ratelimit:*").Result()

	var loginPrefixUsed, refreshPrefixUsed bool
	for _, k := range keys {
		if strings.HasPrefix(k, "ratelimit:auth-login:") {
			loginPrefixUsed = true
		}
		if strings.HasPrefix(k, "ratelimit:auth-refresh:") {
			refreshPrefixUsed = true
		}
	}

	if !loginPrefixUsed {
		t.Error("login route should use 'ratelimit:auth-login:*' key, but found none")
	}
	if !refreshPrefixUsed {
		t.Error("refresh route should use 'ratelimit:auth-refresh:*' key, but found none")
	}
}

func TestDistinctRateLimitPrefixesRequiredForLoginVsRefresh(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.New(slog.NewJSONHandler(&nopWriter{}, &slog.HandlerOptions{}))

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	ctx := context.Background()

	existingKeys, _ := rdb.Keys(ctx, "ratelimit:*").Result()
	if len(existingKeys) > 0 {
		rdb.Del(ctx, existingKeys...)
	}

	router := NewRouter(cfg, logger, rdb, &silentRecorder{}, Handlers{})

	testIP := "192.0.2.1"

	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	reqLogin.RemoteAddr = testIP + ":12345"
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)

	reqRefresh := httptest.NewRequest("POST", "/api/v1/auth/refresh", nil)
	reqRefresh.RemoteAddr = testIP + ":12345"
	wRefresh := httptest.NewRecorder()
	router.ServeHTTP(wRefresh, reqRefresh)

	keys, _ := rdb.Keys(ctx, "ratelimit:*").Result()

	var prefixes []string
	for _, k := range keys {
		parts := strings.Split(k, ":")
		if len(parts) >= 2 && parts[0] == "ratelimit" {
			prefixes = append(prefixes, parts[1])
		}
	}

	hasLogin := false
	hasRefresh := false
	for _, p := range prefixes {
		if p == "auth-login" {
			hasLogin = true
		}
		if p == "auth-refresh" {
			hasRefresh = true
		}
	}

	if !hasLogin {
		t.Error("login endpoint must use 'auth-login' rate limit prefix")
	}
	if !hasRefresh {
		t.Error("refresh endpoint must use 'auth-refresh' rate limit prefix")
	}

	if !hasLogin || !hasRefresh {
		t.Fatalf("FAIL: Both 'auth-login' and 'auth-refresh' prefixes required. Found: %v", prefixes)
	}
}

type nopWriter struct{}

func (n *nopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

type silentRecorder struct{}

func (s *silentRecorder) Record(ctx context.Context, event audit.Event) error {
	return nil
}