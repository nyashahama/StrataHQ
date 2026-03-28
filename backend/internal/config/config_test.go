package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_AllFieldsSet(t *testing.T) {
	envs := map[string]string{
		"PORT":                   "9090",
		"ENV":                    "production",
		"DATABASE_URL":           "postgres://user:pass@localhost:5432/db",
		"REDIS_URL":              "redis://localhost:6379",
		"JWT_SECRET":             "test-secret-that-is-long-enough-32ch",
		"JWT_EXPIRY":             "30m",
		"REFRESH_EXPIRY":         "48h",
		"STRIPE_SECRET_KEY":      "sk_test_123",
		"STRIPE_WEBHOOK_SECRET":  "whsec_123",
		"RESEND_API_KEY":         "re_123",
		"AI_BASE_URL":            "https://api.deepseek.com/v1",
		"AI_API_KEY":             "sk-ai-123",
		"AI_MODEL":               "deepseek-chat",
		"ALLOWED_ORIGINS":        "http://localhost:3000,https://app.stratahq.com",
	}

	for k, v := range envs {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.Env != "production" {
		t.Errorf("Env = %q, want %q", cfg.Env, "production")
	}
	if cfg.DatabaseURL != envs["DATABASE_URL"] {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, envs["DATABASE_URL"])
	}
	if cfg.JWTExpiry != 30*time.Minute {
		t.Errorf("JWTExpiry = %v, want %v", cfg.JWTExpiry, 30*time.Minute)
	}
	if cfg.RefreshExpiry != 48*time.Hour {
		t.Errorf("RefreshExpiry = %v, want %v", cfg.RefreshExpiry, 48*time.Hour)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("AllowedOrigins length = %d, want 2", len(cfg.AllowedOrigins))
	}
}

func TestLoad_Defaults(t *testing.T) {
	required := map[string]string{
		"DATABASE_URL":           "postgres://user:pass@localhost:5432/db",
		"REDIS_URL":              "redis://localhost:6379",
		"JWT_SECRET":             "test-secret-that-is-long-enough-32ch",
		"STRIPE_SECRET_KEY":      "sk_test_123",
		"STRIPE_WEBHOOK_SECRET":  "whsec_123",
		"RESEND_API_KEY":         "re_123",
		"AI_BASE_URL":            "https://api.deepseek.com/v1",
		"AI_API_KEY":             "sk-ai-123",
		"AI_MODEL":               "deepseek-chat",
	}

	for k, v := range required {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	os.Unsetenv("PORT")
	os.Unsetenv("ENV")
	os.Unsetenv("JWT_EXPIRY")
	os.Unsetenv("REFRESH_EXPIRY")
	os.Unsetenv("ALLOWED_ORIGINS")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want default %q", cfg.Port, "8080")
	}
	if cfg.Env != "development" {
		t.Errorf("Env = %q, want default %q", cfg.Env, "development")
	}
	if cfg.JWTExpiry != 15*time.Minute {
		t.Errorf("JWTExpiry = %v, want default %v", cfg.JWTExpiry, 15*time.Minute)
	}
	if cfg.RefreshExpiry != 168*time.Hour {
		t.Errorf("RefreshExpiry = %v, want default %v", cfg.RefreshExpiry, 168*time.Hour)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	for _, key := range []string{"DATABASE_URL", "REDIS_URL", "JWT_SECRET", "STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET", "RESEND_API_KEY", "AI_BASE_URL", "AI_API_KEY", "AI_MODEL"} {
		os.Unsetenv(key)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing required fields, got nil")
	}
}
