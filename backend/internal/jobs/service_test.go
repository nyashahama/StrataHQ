package jobs

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	dbgen "github.com/stratahq/backend/db/gen"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

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
