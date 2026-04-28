//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/jobs"
)

func TestBackgroundJobsClaimDueJobsOnceAcrossWorkers(t *testing.T) {
	ctx := context.Background()

	key := fmt.Sprintf("integration-email-%d", time.Now().UnixNano())
	payload := []byte(`{"collectionEventId":"11111111-1111-1111-1111-111111111111","to":"owner@example.com","subject":"Reminder","htmlBody":"<p>Pay</p>"}`)
	_, err := testQ.EnqueueBackgroundJob(ctx, dbgen.EnqueueBackgroundJobParams{
		Kind:           jobs.KindCollectionReminderEmail,
		Payload:        payload,
		IdempotencyKey: key,
		MaxAttempts:    5,
		RunAfter:       time.Now().Add(-time.Minute),
	})
	require.NoError(t, err)

	first, err := testQ.ClaimDueBackgroundJobs(ctx, dbgen.ClaimDueBackgroundJobsParams{
		Limit:    1,
		LockedBy: pgtype.Text{String: "worker-a", Valid: true},
	})
	require.NoError(t, err)
	require.Len(t, first, 1)

	second, err := testQ.ClaimDueBackgroundJobs(ctx, dbgen.ClaimDueBackgroundJobsParams{
		Limit:    1,
		LockedBy: pgtype.Text{String: "worker-b", Valid: true},
	})
	require.NoError(t, err)
	require.Empty(t, second)
}
