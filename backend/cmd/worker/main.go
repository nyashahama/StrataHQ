package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/jobs"
	"github.com/stratahq/backend/internal/levy"
	"github.com/stratahq/backend/internal/notification"
	"github.com/stratahq/backend/internal/platform/database"
	"github.com/stratahq/backend/internal/twilio"
	"github.com/stratahq/backend/internal/whatsapp"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	emailClient := notification.NewEmailClient(cfg.ResendAPIKey, cfg.EmailFrom)
	var sender jobs.CollectionReminderWhatsAppSender
	if cfg.TwilioAccountSID != "" && cfg.TwilioAuthToken != "" && cfg.TwilioWhatsAppNumber != "" {
		sender = twilio.NewClient(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioWhatsAppNumber)
	} else {
		sender = whatsapp.NewDisabledSender()
		logger.Info("twilio whatsapp disabled, using disabled sender")
	}

	levyService := levy.NewService(db, nil, nil)

	registry := jobs.Registry{
		jobs.KindCollectionReminderEmail:    jobs.NewCollectionReminderEmailHandler(db.Q, emailClient),
		jobs.KindCollectionReminderWhatsApp: jobs.NewCollectionReminderWhatsAppHandler(db.Q, sender),
		jobs.KindBankStatementImport:        jobs.NewBankStatementImportHandler(levyService),
	}

	workerID, err := os.Hostname()
	if err != nil || workerID == "" {
		workerID = "worker-" + time.Now().UTC().Format("20060102150405")
	}

	service := jobs.NewService(db.Q, registry, logger, jobs.RealClock{}, jobs.Config{
		WorkerID:  workerID,
		BatchSize: cfg.WorkerBatchSize,
		LeaseTTL:  cfg.WorkerLeaseTTL,
	})

	ticker := time.NewTicker(cfg.WorkerPollInterval)
	defer ticker.Stop()

	logger.Info("background worker started", "worker_id", workerID, "poll_interval", cfg.WorkerPollInterval.String(), "batch_size", cfg.WorkerBatchSize)
	for {
		count, workErr := service.WorkOnce(ctx)
		if workErr != nil && !errors.Is(workErr, context.Canceled) {
			logger.Error("background worker iteration failed", "error", workErr)
		}
		if count > 0 {
			logger.Info("background worker processed jobs", "count", count)
			continue
		}

		select {
		case <-ctx.Done():
			logger.Info("background worker stopped")
			return
		case <-ticker.C:
		}
	}
}
