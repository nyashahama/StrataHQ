package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
)

type Querier interface {
	EnqueueBackgroundJob(ctx context.Context, arg dbgen.EnqueueBackgroundJobParams) (dbgen.BackgroundJob, error)
	ClaimDueBackgroundJobs(ctx context.Context, arg dbgen.ClaimDueBackgroundJobsParams) ([]dbgen.BackgroundJob, error)
	MarkBackgroundJobSucceeded(ctx context.Context, id uuid.UUID) (dbgen.BackgroundJob, error)
	RetryBackgroundJob(ctx context.Context, arg dbgen.RetryBackgroundJobParams) (dbgen.BackgroundJob, error)
	MarkBackgroundJobFailed(ctx context.Context, arg dbgen.MarkBackgroundJobFailedParams) (dbgen.BackgroundJob, error)
	RecoverStaleBackgroundJobs(ctx context.Context, arg dbgen.RecoverStaleBackgroundJobsParams) ([]dbgen.BackgroundJob, error)
}

type Service struct {
	q         Querier
	clock     Clock
	registry  Registry
	logger    *slog.Logger
	workerID  string
	batchSize int32
	leaseTTL  time.Duration
}

type Config struct {
	WorkerID  string
	BatchSize int32
	LeaseTTL  time.Duration
}

func NewService(q Querier, registry Registry, logger *slog.Logger, clock Clock, cfg Config) *Service {
	if clock == nil {
		clock = RealClock{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.WorkerID == "" {
		cfg.WorkerID = "worker-" + uuid.NewString()
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}
	if cfg.LeaseTTL <= 0 {
		cfg.LeaseTTL = 5 * time.Minute
	}
	return &Service{
		q:         q,
		clock:     clock,
		registry:  registry,
		logger:    logger,
		workerID:  cfg.WorkerID,
		batchSize: cfg.BatchSize,
		leaseTTL:  cfg.LeaseTTL,
	}
}

func (s *Service) Enqueue(ctx context.Context, input EnqueueInput) (dbgen.BackgroundJob, error) {
	if input.Kind == "" {
		return dbgen.BackgroundJob{}, fmt.Errorf("job kind is required")
	}
	if input.IdempotencyKey == "" {
		return dbgen.BackgroundJob{}, fmt.Errorf("job idempotency key is required")
	}
	if input.MaxAttempts <= 0 {
		input.MaxAttempts = 5
	}
	if input.RunAfter.IsZero() {
		input.RunAfter = s.clock.Now()
	}

	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return dbgen.BackgroundJob{}, fmt.Errorf("marshal job payload: %w", err)
	}

	return s.q.EnqueueBackgroundJob(ctx, dbgen.EnqueueBackgroundJobParams{
		Kind:           input.Kind,
		Payload:        payload,
		IdempotencyKey: input.IdempotencyKey,
		MaxAttempts:    input.MaxAttempts,
		RunAfter:       input.RunAfter,
	})
}

func (s *Service) WorkOnce(ctx context.Context) (int, error) {
	recovered, recoverErr := s.q.RecoverStaleBackgroundJobs(ctx, dbgen.RecoverStaleBackgroundJobsParams{
		LockedAt: pgtype.Timestamptz{Time: s.clock.Now().Add(-s.leaseTTL), Valid: true},
		LastError: pgtype.Text{
			String: "recovered stale worker lease",
			Valid:  true,
		},
	})
	if recoverErr != nil {
		return 0, fmt.Errorf("recover stale jobs: %w", recoverErr)
	}
	for _, job := range recovered {
		s.logger.Warn("recovered stale background job", "job_id", job.ID, "kind", job.Kind)
	}

	jobs, err := s.q.ClaimDueBackgroundJobs(ctx, dbgen.ClaimDueBackgroundJobsParams{
		Limit:    s.batchSize,
		LockedBy: pgtype.Text{String: s.workerID, Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("claim due jobs: %w", err)
	}

	for _, job := range jobs {
		s.handleJob(ctx, job)
	}

	return len(jobs), nil
}

func (s *Service) handleJob(ctx context.Context, job dbgen.BackgroundJob) {
	handler, ok := s.registry[job.Kind]
	if !ok {
		_, err := s.q.MarkBackgroundJobFailed(ctx, dbgen.MarkBackgroundJobFailedParams{
			ID:        job.ID,
			LastError: pgtype.Text{String: ErrUnknownKind.Error(), Valid: true},
		})
		if err != nil {
			s.logger.Error("failed to mark unknown job kind failed", "job_id", job.ID, "kind", job.Kind, "error", err)
		}
		return
	}

	err := handler.Handle(ctx, rawPayload(job))
	if err == nil {
		if _, markErr := s.q.MarkBackgroundJobSucceeded(ctx, job.ID); markErr != nil {
			s.logger.Error("failed to mark background job succeeded", "job_id", job.ID, "kind", job.Kind, "error", markErr)
		}
		return
	}

	if isNonRetryable(err) || shouldFailPermanently(job) {
		if _, markErr := s.q.MarkBackgroundJobFailed(ctx, dbgen.MarkBackgroundJobFailedParams{
			ID:        job.ID,
			LastError: pgtype.Text{String: jobErrorString(err), Valid: true},
		}); markErr != nil {
			s.logger.Error("failed to mark background job failed", "job_id", job.ID, "kind", job.Kind, "error", markErr)
		}
		return
	}

	if _, retryErr := s.q.RetryBackgroundJob(ctx, dbgen.RetryBackgroundJobParams{
		ID:        job.ID,
		RunAfter:  nextRunAfter(s.clock.Now(), job.Attempts),
		LastError: pgtype.Text{String: jobErrorString(err), Valid: true},
	}); retryErr != nil {
		s.logger.Error("failed to retry background job", "job_id", job.ID, "kind", job.Kind, "error", retryErr)
	}
}

func nextRunAfter(now time.Time, attempts int32) time.Time {
	delay := 30 * time.Second
	for i := int32(0); i < attempts; i++ {
		delay *= 2
		if delay >= 5*time.Minute {
			delay = 5 * time.Minute
			break
		}
	}
	return now.Add(delay)
}

func shouldFailPermanently(job dbgen.BackgroundJob) bool {
	return job.Attempts+1 >= job.MaxAttempts
}

func isNonRetryable(err error) bool {
	return errors.Is(err, ErrNonRetryable)
}

func decodePayload(raw json.RawMessage, dest any) error {
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("%w: %v", ErrBadPayload, err)
	}
	switch v := dest.(type) {
	case *CollectionReminderEmailPayload:
		if v.CollectionEventID == uuid.Nil || v.To == "" || v.Subject == "" || v.HTMLBody == "" {
			return fmt.Errorf("%w: invalid collection reminder email payload", ErrBadPayload)
		}
	case *CollectionReminderWhatsAppPayload:
		if v.CollectionEventID == uuid.Nil || v.To == "" || v.Body == "" {
			return fmt.Errorf("%w: invalid collection reminder whatsapp payload", ErrBadPayload)
		}
	}
	return nil
}

func rawPayload(job dbgen.BackgroundJob) json.RawMessage {
	return json.RawMessage(job.Payload)
}

func jobErrorString(err error) string {
	if err == nil {
		return "unknown job error"
	}
	return err.Error()
}

func uuidText(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}
