package levy

import (
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
