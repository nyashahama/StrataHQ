package levy

import (
	"strings"
	"testing"
	"time"
)

func TestScoreAttentionItemPrioritizesOlderHigherBalanceDebt(t *testing.T) {
	now := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	oldHigh := attentionAccount{
		LevyAccountID:     "acc-old",
		SchemeID:          "scheme-1",
		SchemeName:        "Rosewood Estate",
		UnitID:            "unit-1",
		UnitIdentifier:    "5C",
		OwnerName:         "Rose Example",
		OutstandingCents:  930000,
		DaysOverdue:       97,
		LastActionType:    "",
		LastActionDaysAgo: 999,
	}
	newLow := attentionAccount{
		LevyAccountID:     "acc-new",
		SchemeID:          "scheme-1",
		SchemeName:        "Rosewood Estate",
		UnitID:            "unit-2",
		UnitIdentifier:    "1A",
		OwnerName:         "New Example",
		OutstandingCents:  145000,
		DaysOverdue:       18,
		LastActionType:    "follow_up_logged",
		LastActionDaysAgo: 2,
	}

	oldScored := scoreAttentionItem(oldHigh, now)
	newScored := scoreAttentionItem(newLow, now)

	if oldScored.RiskScore <= newScored.RiskScore {
		t.Fatalf("older larger arrears scored %d, want > %d", oldScored.RiskScore, newScored.RiskScore)
	}
	if oldScored.RecommendedAction != "legal_review_flagged" {
		t.Fatalf("recommended_action = %q, want legal_review_flagged", oldScored.RecommendedAction)
	}
}

func TestScoreAttentionItemUsesPromiseBreakageAsEscalationSignal(t *testing.T) {
	now := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	item := attentionAccount{
		LevyAccountID:      "acc-1",
		SchemeID:           "scheme-1",
		SchemeName:         "Rosewood Estate",
		UnitID:             "unit-1",
		UnitIdentifier:     "3B",
		OwnerName:          "Owner Example",
		OutstandingCents:   380000,
		DaysOverdue:        44,
		LastActionType:     "promise_to_pay",
		LastActionDaysAgo:  10,
		PromiseDateOverdue: true,
	}

	scored := scoreAttentionItem(item, now)

	if scored.RecommendedAction != "legal_review_flagged" {
		t.Fatalf("recommended_action = %q, want legal_review_flagged", scored.RecommendedAction)
	}
	if len(scored.ScoreDrivers) == 0 {
		t.Fatalf("score drivers should not be empty")
	}
}

func TestScoreAttentionItemSuppressesUrgencyWhenFreshFollowUpExists(t *testing.T) {
	now := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	item := attentionAccount{
		LevyAccountID:     "acc-1",
		SchemeID:          "scheme-1",
		SchemeName:        "Rosewood Estate",
		UnitID:            "unit-1",
		UnitIdentifier:    "8A",
		OwnerName:         "Owner Example",
		OutstandingCents:  245000,
		DaysOverdue:       12,
		LastActionType:    "follow_up_logged",
		LastActionDaysAgo: 1,
	}

	scored := scoreAttentionItem(item, now)

	if scored.RecommendedAction != "follow_up_logged" {
		t.Fatalf("recommended_action = %q, want follow_up_logged", scored.RecommendedAction)
	}
	if scored.RiskScore >= 60 {
		t.Fatalf("risk score = %d, want less than 60 after recent follow-up", scored.RiskScore)
	}
}

func TestScoreAttentionItemSuppressesRepeatedLegalReviewRecommendation(t *testing.T) {
	now := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	item := attentionAccount{
		LevyAccountID:     "acc-1",
		SchemeID:          "scheme-1",
		SchemeName:        "Rosewood Estate",
		UnitID:            "unit-1",
		UnitIdentifier:    "12C",
		OwnerName:         "Owner Example",
		OutstandingCents:  930000,
		DaysOverdue:       97,
		LastActionType:    "legal_review_flagged",
		LastActionDaysAgo: 1,
	}

	scored := scoreAttentionItem(item, now)

	if scored.RecommendedAction != "follow_up_logged" {
		t.Fatalf("recommended_action = %q, want follow_up_logged after recent legal review", scored.RecommendedAction)
	}
	if scored.RiskScore >= 65 {
		t.Fatalf("risk score = %d, want reduced score below 65 after recent legal review", scored.RiskScore)
	}
}

func TestBuildReminderDraftMarksMissingChannels(t *testing.T) {
	item := attentionAccount{
		LevyAccountID:    "acc-1",
		SchemeID:         "scheme-1",
		SchemeName:       "Rosewood Estate",
		UnitID:           "unit-1",
		UnitIdentifier:   "5C",
		OwnerName:        "Rose Example",
		OutstandingCents: 930000,
		DaysOverdue:      97,
	}

	draft := buildReminderDraft(item, "", "", false)

	if draft.Email.Enabled {
		t.Fatalf("email should be disabled when no email exists")
	}
	if draft.Email.DisabledReason != "No email on file" {
		t.Fatalf("email disabled reason = %q", draft.Email.DisabledReason)
	}
	if draft.WhatsApp.Enabled {
		t.Fatalf("whatsapp should be disabled when no connected number exists")
	}
	if draft.WhatsApp.DisabledReason != "No WhatsApp number or active thread" {
		t.Fatalf("whatsapp disabled reason = %q", draft.WhatsApp.DisabledReason)
	}
}

func TestBuildReminderDraftGeneratesDeterministicBodies(t *testing.T) {
	item := attentionAccount{
		LevyAccountID:    "acc-1",
		SchemeID:         "scheme-1",
		SchemeName:       "Rosewood Estate",
		UnitID:           "unit-1",
		UnitIdentifier:   "5C",
		OwnerName:        "Rose Example",
		OutstandingCents: 930000,
		DaysOverdue:      97,
	}

	draft := buildReminderDraft(item, "rose@example.com", "+27715550101", true)

	if !strings.Contains(draft.Email.Subject, "Rosewood Estate") {
		t.Fatalf("email subject = %q, want scheme name included", draft.Email.Subject)
	}
	if !strings.Contains(draft.Email.Body, "R 9 300.00") {
		t.Fatalf("email body = %q, want outstanding amount", draft.Email.Body)
	}
	if !strings.Contains(draft.WhatsApp.Body, "Unit 5C") {
		t.Fatalf("whatsapp body = %q, want unit identifier", draft.WhatsApp.Body)
	}
}

func TestValidateSendReminderInputRejectsBlankEnabledChannelContent(t *testing.T) {
	tests := []struct {
		name  string
		input SendReminderInput
	}{
		{
			name: "email subject",
			input: SendReminderInput{
				Email: ReminderChannelInput{Enabled: true, Subject: " ", Body: "Please pay"},
			},
		},
		{
			name: "email body",
			input: SendReminderInput{
				Email: ReminderChannelInput{Enabled: true, Subject: "Levy reminder", Body: " "},
			},
		},
		{
			name: "whatsapp body",
			input: SendReminderInput{
				WhatsApp: ReminderChannelInput{Enabled: true, Body: " "},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateSendReminderInput(tt.input); err != ErrInvalidInput {
				t.Fatalf("validateSendReminderInput error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestScoreAttentionItemSuppressesReminderForActivePromise(t *testing.T) {
	now := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	item := attentionAccount{
		LevyAccountID:     "acc-1",
		SchemeID:          "scheme-1",
		SchemeName:        "Rosewood Estate",
		UnitID:            "unit-1",
		UnitIdentifier:    "5C",
		OwnerName:         "Rose Example",
		OutstandingCents:  930000,
		DaysOverdue:       97,
		LastActionType:    "promise_to_pay",
		LastActionDaysAgo: 2,
		HasActivePromise:  true,
	}

	scored := scoreAttentionItem(item, now)

	if scored.RecommendedAction != "active_promise" {
		t.Fatalf("recommended action = %q, want active_promise", scored.RecommendedAction)
	}
	if scored.RiskScore >= 70 {
		t.Fatalf("risk score = %d, want reduced score below 70 for active promise", scored.RiskScore)
	}
}

func TestScoreAttentionItemReducesUrgencyAfterFreshReminder(t *testing.T) {
	now := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	item := attentionAccount{
		LevyAccountID:     "acc-1",
		SchemeID:          "scheme-1",
		SchemeName:        "Rosewood Estate",
		UnitID:            "unit-1",
		UnitIdentifier:    "5C",
		OwnerName:         "Rose Example",
		OutstandingCents:  930000,
		DaysOverdue:       97,
		LastActionType:    "reminder_sent",
		LastActionDaysAgo: 1,
	}

	scored := scoreAttentionItem(item, now)

	if scored.RiskScore >= 80 {
		t.Fatalf("risk score = %d, want cooldown below 80 after fresh reminder", scored.RiskScore)
	}
	if scored.RecommendedAction != "follow_up_logged" {
		t.Fatalf("recommended action = %q, want follow_up_logged", scored.RecommendedAction)
	}
}
