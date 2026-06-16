package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	dbgen "github.com/stratahq/backend/db/gen"
)

func TestBackoffCapsAtFiveMinutes(t *testing.T) {
	now := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	got := nextRunAfter(now, 5)
	require.Equal(t, now.Add(5*time.Minute), got)
}

func TestBackoffStartsAtThirtySeconds(t *testing.T) {
	now := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	got := nextRunAfter(now, 0)
	require.Equal(t, now.Add(30*time.Second), got)
}

func TestShouldFailPermanentlyWhenNextAttemptReachesMax(t *testing.T) {
	job := dbgen.BackgroundJob{
		Attempts:    4,
		MaxAttempts: 5,
	}
	require.True(t, shouldFailPermanently(job))
}

func TestShouldRetryWhenAttemptsRemain(t *testing.T) {
	job := dbgen.BackgroundJob{
		Attempts:    2,
		MaxAttempts: 5,
	}
	require.False(t, shouldFailPermanently(job))
}

func TestDecodePayloadWrapsInvalidJSON(t *testing.T) {
	var payload CollectionReminderEmailPayload
	err := decodePayload(json.RawMessage(`{"collectionEventId":`), &payload)
	require.ErrorIs(t, err, ErrBadPayload)
}

func TestDecodePayloadRejectsEmptyRequiredFields(t *testing.T) {
	var payload CollectionReminderEmailPayload
	err := decodePayload(json.RawMessage(`{"collectionEventId":"00000000-0000-0000-0000-000000000000","to":"","subject":"s","htmlBody":"h"}`), &payload)
	require.ErrorIs(t, err, ErrBadPayload)
}

func TestJobErrorStringIsStable(t *testing.T) {
	err := errors.New("provider failed")
	require.Equal(t, "provider failed", jobErrorString(err))
	require.Equal(t, "unknown job error", jobErrorString(nil))
}

func TestRawPayloadFromModel(t *testing.T) {
	raw := rawPayload(dbgen.BackgroundJob{
		Payload: []byte(`{"ok":true}`),
	})
	require.JSONEq(t, `{"ok":true}`, string(raw))
}

func TestUUIDTextHelper(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	got := uuidText(id)
	require.Equal(t, pgtype.UUID{Bytes: id, Valid: true}, got)
}

func TestIsNonRetryableMatchesBadPayload(t *testing.T) {
	wrapped := errors.Join(errors.New("decode failed"), ErrBadPayload)
	require.True(t, isNonRetryable(wrapped))
	require.True(t, isNonRetryable(ErrBadPayload))
	require.False(t, isNonRetryable(errors.New("transient network error")))
	require.True(t, isNonRetryable(ErrNonRetryable))
}

type fakeQuerier struct {
	recoveredParams []dbgen.RecoverStaleBackgroundJobsParams
	recovered       []dbgen.BackgroundJob
	claimed         []dbgen.ClaimDueBackgroundJobsParams
	dueJobs         []dbgen.BackgroundJob
	retryCalls      []dbgen.RetryBackgroundJobParams
	failedCalls     []dbgen.MarkBackgroundJobFailedParams
	succeeded       []uuid.UUID
}

func (f *fakeQuerier) RecoverStaleBackgroundJobs(_ context.Context, arg dbgen.RecoverStaleBackgroundJobsParams) ([]dbgen.BackgroundJob, error) {
	f.recoveredParams = append(f.recoveredParams, arg)
	return f.recovered, nil
}

func (f *fakeQuerier) ClaimDueBackgroundJobs(_ context.Context, arg dbgen.ClaimDueBackgroundJobsParams) ([]dbgen.BackgroundJob, error) {
	f.claimed = append(f.claimed, arg)
	return f.dueJobs, nil
}

func (f *fakeQuerier) MarkBackgroundJobSucceeded(_ context.Context, id uuid.UUID) (dbgen.BackgroundJob, error) {
	f.succeeded = append(f.succeeded, id)
	return dbgen.BackgroundJob{ID: id, Status: "succeeded"}, nil
}

func (f *fakeQuerier) RetryBackgroundJob(_ context.Context, arg dbgen.RetryBackgroundJobParams) (dbgen.BackgroundJob, error) {
	f.retryCalls = append(f.retryCalls, arg)
	return dbgen.BackgroundJob{ID: arg.ID, Status: "queued", Attempts: 1}, nil
}

func (f *fakeQuerier) MarkBackgroundJobFailed(_ context.Context, arg dbgen.MarkBackgroundJobFailedParams) (dbgen.BackgroundJob, error) {
	f.failedCalls = append(f.failedCalls, arg)
	return dbgen.BackgroundJob{ID: arg.ID, Status: "failed"}, nil
}

func (f *fakeQuerier) EnqueueBackgroundJob(_ context.Context, _ dbgen.EnqueueBackgroundJobParams) (dbgen.BackgroundJob, error) {
	return dbgen.BackgroundJob{}, nil
}

func TestWorkOnceRecoversStaleJobsBeforeClaim(t *testing.T) {
	now := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	q := &fakeQuerier{}
	svc := NewService(q, Registry{}, slog.Default(), &fakeClock{t: now}, Config{
		WorkerID:  "worker-test",
		BatchSize: 5,
		LeaseTTL:  time.Minute,
	})

	processed, err := svc.WorkOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, processed)

	require.Len(t, q.recoveredParams, 1)
	require.True(t, q.recoveredParams[0].LastError.Valid)
	require.Equal(t, "recovered stale worker lease", q.recoveredParams[0].LastError.String)

	require.Len(t, q.claimed, 1)
	require.Equal(t, "worker-test", q.claimed[0].LockedBy.String)
}

func TestHandleJobFailsPermanentlyOnBadPayload(t *testing.T) {
	now := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	q := &fakeQuerier{}

	handler := HandlerFunc(func(_ context.Context, _ json.RawMessage) error {
		return errors.Join(errors.New("invalid collection reminder email payload"), ErrBadPayload)
	})
	registry := Registry{KindCollectionReminderEmail: handler}

	jobID := uuid.New()
	job := dbgen.BackgroundJob{
		ID:          jobID,
		Kind:        KindCollectionReminderEmail,
		Attempts:    1,
		MaxAttempts: 5,
		Status:      "running",
	}

	svc := NewService(q, registry, slog.Default(), &fakeClock{t: now}, Config{WorkerID: "worker-test"})
	svc.handleJob(context.Background(), job)

	require.Len(t, q.failedCalls, 1)
	require.Equal(t, jobID, q.failedCalls[0].ID)
	require.Empty(t, q.retryCalls, "bad payload should not be retried")
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time { return c.t }
