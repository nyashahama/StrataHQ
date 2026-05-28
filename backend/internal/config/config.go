package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ConfigStrings
	ConfigDurations
	ConfigInts
}

type ConfigStrings struct {
	Port                 string
	Env                  string
	DatabaseURL          string
	RedisURL             string
	JWTSecret            string
	JWTIssuer            string
	JWTAudience          string
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePriceID        string
	ResendAPIKey         string
	AIBaseURL            string
	AIAPIKey             string
	AIModel              string
	AppBaseURL           string
	BackendBaseURL       string
	EmailFrom            string
	AdminEmail           string
	AdminSecret          string
	TwilioAccountSID     string
	TwilioAuthToken      string
	TwilioWhatsAppNumber string
	AllowedOrigins       []string
	TrustedProxyCIDRs    []string
	MetricsToken         string
}

type ConfigDurations struct {
	JWTExpiry          time.Duration
	RefreshExpiry      time.Duration
	WorkerPollInterval time.Duration
	WorkerLeaseTTL     time.Duration
}

type ConfigInts struct {
	WorkerBatchSize      int32
	WorkerMaxAttempts    int32
	AuthLoginRateLimit   int
	AuthRefreshRateLimit int
}

func Load() (*Config, error) {
	cfg := &Config{
		ConfigStrings: ConfigStrings{
			Port:                 getEnv("PORT", "8080"),
			Env:                  getEnv("ENV", "development"),
			DatabaseURL:          os.Getenv("DATABASE_URL"),
			RedisURL:             os.Getenv("REDIS_URL"),
			JWTSecret:            os.Getenv("JWT_SECRET"),
			JWTIssuer:            getEnv("JWT_ISSUER", os.Getenv("APP_BASE_URL")),
			JWTAudience:          getEnv("JWT_AUDIENCE", "stratahq-api"),
			StripeSecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
			StripeWebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),
			StripePriceID:        os.Getenv("STRIPE_PRICE_ID"),
			ResendAPIKey:         os.Getenv("RESEND_API_KEY"),
			AIBaseURL:            os.Getenv("AI_BASE_URL"),
			AIAPIKey:             os.Getenv("AI_API_KEY"),
			AIModel:              os.Getenv("AI_MODEL"),
			AppBaseURL:           os.Getenv("APP_BASE_URL"),
			BackendBaseURL:       getEnv("BACKEND_BASE_URL", os.Getenv("APP_BASE_URL")),
			EmailFrom:            getEnv("EMAIL_FROM", "noreply@stratahq.co.za"),
			AdminEmail:           os.Getenv("ADMIN_EMAIL"),
			AdminSecret:          os.Getenv("ADMIN_SECRET"),
			TwilioAccountSID:     os.Getenv("TWILIO_ACCOUNT_SID"),
			TwilioAuthToken:      os.Getenv("TWILIO_AUTH_TOKEN"),
			TwilioWhatsAppNumber: os.Getenv("TWILIO_WHATSAPP_NUMBER"),
			MetricsToken:         os.Getenv("METRICS_TOKEN"),
		},
	}

	var err error
	cfg.ConfigDurations.JWTExpiry, err = parseDuration("JWT_EXPIRY", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	cfg.ConfigDurations.RefreshExpiry, err = parseDuration("REFRESH_EXPIRY", 168*time.Hour)
	if err != nil {
		return nil, err
	}
	cfg.ConfigDurations.WorkerPollInterval, err = parseDuration("WORKER_POLL_INTERVAL", 2*time.Second)
	if err != nil {
		return nil, err
	}
	cfg.ConfigDurations.WorkerLeaseTTL, err = parseDuration("WORKER_LEASE_TTL", 5*time.Minute)
	if err != nil {
		return nil, err
	}
	cfg.ConfigInts.WorkerBatchSize, err = parseInt32("WORKER_BATCH_SIZE", 10)
	if err != nil {
		return nil, err
	}
	cfg.ConfigInts.WorkerMaxAttempts, err = parseInt32("WORKER_MAX_ATTEMPTS", 5)
	if err != nil {
		return nil, err
	}
	cfg.ConfigInts.AuthLoginRateLimit, err = parseInt("AUTH_LOGIN_RATE_LIMIT", 5)
	if err != nil {
		return nil, err
	}
	cfg.ConfigInts.AuthRefreshRateLimit, err = parseInt("AUTH_REFRESH_RATE_LIMIT", 30)
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

	proxyCIDRs := os.Getenv("TRUSTED_PROXY_CIDRS")
	if proxyCIDRs != "" {
		cfg.TrustedProxyCIDRs = strings.Split(proxyCIDRs, ",")
		for i := range cfg.TrustedProxyCIDRs {
			cfg.TrustedProxyCIDRs[i] = strings.TrimSpace(cfg.TrustedProxyCIDRs[i])
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

	if c.Env == "production" && len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}
	if strings.Contains(strings.ToLower(c.JWTSecret), "change-me") {
		return fmt.Errorf("JWT_SECRET must not be a placeholder value")
	}

	if c.TwilioAuthToken != "" || c.TwilioAccountSID != "" || c.TwilioWhatsAppNumber != "" {
		if strings.TrimSpace(c.TwilioAuthToken) == "" {
			return fmt.Errorf("TWILIO_AUTH_TOKEN must be set when Twilio integration is configured")
		}
		if strings.TrimSpace(c.TwilioAccountSID) == "" {
			return fmt.Errorf("TWILIO_ACCOUNT_SID must be set when TWILIO_AUTH_TOKEN is configured")
		}
		if strings.TrimSpace(c.TwilioWhatsAppNumber) == "" {
			return fmt.Errorf("TWILIO_WHATSAPP_NUMBER must be set when TWILIO_AUTH_TOKEN is configured")
		}
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

func parseInt(key string, fallback int) (int, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for %s: %w", key, err)
	}
	if i <= 0 {
		return 0, fmt.Errorf("invalid integer for %s: must be greater than zero", key)
	}
	return i, nil
}

func parseInt32(key string, fallback int32) (int32, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for %s: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("invalid integer for %s: must be positive", key)
	}
	return int32(parsed), nil
}
