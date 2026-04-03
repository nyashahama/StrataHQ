package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	JWTExpiry           time.Duration
	RefreshExpiry       time.Duration
	Port                string
	Env                 string
	DatabaseURL         string
	RedisURL            string
	JWTSecret           string
	StripeSecretKey     string
	StripeWebhookSecret string
	StripePriceID       string
	ResendAPIKey        string
	AIBaseURL           string
	AIAPIKey            string
	AIModel             string
	AppBaseURL          string
	EmailFrom           string
	AllowedOrigins      []string
	AdminEmail          string
	AdminSecret         string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		Env:                 getEnv("ENV", "development"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		RedisURL:            os.Getenv("REDIS_URL"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripePriceID:       os.Getenv("STRIPE_PRICE_ID"),
		ResendAPIKey:        os.Getenv("RESEND_API_KEY"),
		AIBaseURL:           os.Getenv("AI_BASE_URL"),
		AIAPIKey:            os.Getenv("AI_API_KEY"),
		AIModel:             os.Getenv("AI_MODEL"),
		AppBaseURL:          os.Getenv("APP_BASE_URL"),
		EmailFrom:           getEnv("EMAIL_FROM", "noreply@stratahq.co.za"),
		AdminEmail:          os.Getenv("ADMIN_EMAIL"),
		AdminSecret:         os.Getenv("ADMIN_SECRET"),
	}

	var err error
	cfg.JWTExpiry, err = parseDuration("JWT_EXPIRY", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	cfg.RefreshExpiry, err = parseDuration("REFRESH_EXPIRY", 168*time.Hour)
	if err != nil {
		return nil, err
	}

	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins != "" {
		cfg.AllowedOrigins = strings.Split(origins, ",")
		for i := range cfg.AllowedOrigins {
			cfg.AllowedOrigins[i] = strings.TrimSpace(cfg.AllowedOrigins[i])
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL":   c.DatabaseURL,
		"REDIS_URL":      c.RedisURL,
		"JWT_SECRET":     c.JWTSecret,
		"RESEND_API_KEY": c.ResendAPIKey,
		"AI_BASE_URL":    c.AIBaseURL,
		"AI_API_KEY":     c.AIAPIKey,
		"AI_MODEL":       c.AIModel,
		"APP_BASE_URL":   c.AppBaseURL,
	}

	var missing []string
	for name, val := range required {
		if val == "" {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func parseDuration(key string, fallback time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}
	return d, nil
}
