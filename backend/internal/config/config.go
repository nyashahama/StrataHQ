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
	WorkerBatchSize             int32
	WorkerMaxAttempts           int32
	AuthLoginRateLimit          int
	AuthRefreshRateLimit        int
	AuthForgotPasswordRateLimit int
	EarlyAccessRateLimit        int
	InvitationAcceptRateLimit   int
}

func Load() (*Config, error) {
	cfg, err := load()
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(requiredConfigForAPI(cfg)); err != nil {
		return nil, err
	}

	return cfg, nil
}

func LoadWorker() (*Config, error) {
	cfg, err := load()
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(requiredConfigForWorker(cfg)); err != nil {
		return nil, err
	}

	return cfg, nil
}

func load() (*Config, error) {
	appBaseURL := strings.TrimSpace(os.Getenv("APP_BASE_URL"))
	cfg := &Config{
		ConfigStrings: ConfigStrings{
			Port:                 getEnv("PORT", "8080"),
			Env:                  getEnv("ENV", "development"),
			DatabaseURL:          strings.TrimSpace(os.Getenv("DATABASE_URL")),
			RedisURL:             strings.TrimSpace(os.Getenv("REDIS_URL")),
			JWTSecret:            strings.TrimSpace(os.Getenv("JWT_SECRET")),
			JWTIssuer:            getEnv("JWT_ISSUER", appBaseURL),
			JWTAudience:          getEnv("JWT_AUDIENCE", "stratahq-api"),
			StripeSecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
			StripeWebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),
			StripePriceID:        os.Getenv("STRIPE_PRICE_ID"),
			ResendAPIKey:         strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
			AIBaseURL:            strings.TrimSpace(os.Getenv("AI_BASE_URL")),
			AIAPIKey:             strings.TrimSpace(os.Getenv("AI_API_KEY")),
			AIModel:              strings.TrimSpace(os.Getenv("AI_MODEL")),
			AppBaseURL:           appBaseURL,
			BackendBaseURL:       getEnv("BACKEND_BASE_URL", appBaseURL),
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
	cfg.ConfigInts.AuthForgotPasswordRateLimit, err = parseInt("AUTH_FORGOT_PASSWORD_RATE_LIMIT", 3)
	if err != nil {
		return nil, err
	}
	cfg.ConfigInts.EarlyAccessRateLimit, err = parseInt("EARLY_ACCESS_RATE_LIMIT", 3)
	if err != nil {
		return nil, err
	}
	cfg.ConfigInts.InvitationAcceptRateLimit, err = parseInt("INVITATION_ACCEPT_RATE_LIMIT", 10)
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

	return cfg, nil
}

func requiredConfigForAPI(c *Config) map[string]string {
	return map[string]string{
		"DATABASE_URL":   c.DatabaseURL,
		"REDIS_URL":      c.RedisURL,
		"JWT_SECRET":     c.JWTSecret,
		"RESEND_API_KEY": c.ResendAPIKey,
		"AI_BASE_URL":    c.AIBaseURL,
		"AI_API_KEY":     c.AIAPIKey,
		"AI_MODEL":       c.AIModel,
		"APP_BASE_URL":   c.AppBaseURL,
	}
}

func requiredConfigForWorker(c *Config) map[string]string {
	return map[string]string{
		"DATABASE_URL":   c.DatabaseURL,
		"RESEND_API_KEY": c.ResendAPIKey,
	}
}

func (c *Config) validate(required map[string]string) error {
	var missing []string
	for name, val := range required {
		if strings.TrimSpace(val) == "" {
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
