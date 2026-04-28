package levy

import "testing"

func TestLevyPeriodCreatedAuditEvent(t *testing.T) {
	event := levyPeriodCreatedAuditEvent(levyPeriodAuditInput{
		SchemeID:     "scheme-1",
		OrgID:        "org-1",
		ActorUserID:  "user-1",
		ActorRole:    "admin",
		PeriodID:     "period-1",
		Label:        "May 2026",
		AmountCents:  250000,
		DueDate:      "2026-05-31",
		AccountCount: 12,
	})

	if event.Action != "levy_period.created" {
		t.Fatalf("action = %q, want levy_period.created", event.Action)
	}
	after := event.AfterState.(map[string]any)
	if after["account_count"] != 12 {
		t.Fatalf("account count = %v, want 12", after["account_count"])
	}
}

func TestLevyReconciledAuditEvent(t *testing.T) {
	event := levyReconciledAuditEvent(levyReconcileAuditInput{
		SchemeID:          "scheme-1",
		OrgID:             "org-1",
		ActorUserID:       "user-1",
		ActorRole:         "admin",
		AppliedCount:      2,
		SkippedCount:      1,
		UpdatedAccountIDs: []string{"account-1", "account-2"},
	})

	if event.Action != "levy.reconciled" {
		t.Fatalf("action = %q, want levy.reconciled", event.Action)
	}
	metadata := event.Metadata.(map[string]any)
	if metadata["skipped_count"] != 1 {
		t.Fatalf("skipped count = %v, want 1", metadata["skipped_count"])
	}
}
