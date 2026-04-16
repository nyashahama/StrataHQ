package levy

import (
	"sort"
	"time"
)

type attentionAccount struct {
	PromiseDateOverdue bool
	DaysOverdue        int
	LastActionDaysAgo  int
	OutstandingCents   int64
	LevyAccountID      string
	SchemeID           string
	SchemeName         string
	UnitID             string
	UnitIdentifier     string
	OwnerName          string
	LastActionType     string
}

type AttentionItem struct {
	DaysOverdue       int      `json:"days_overdue"`
	RiskScore         int      `json:"risk_score"`
	OutstandingCents  int64    `json:"outstanding_cents"`
	LevyAccountID     string   `json:"levy_account_id"`
	SchemeID          string   `json:"scheme_id"`
	SchemeName        string   `json:"scheme_name"`
	UnitID            string   `json:"unit_id"`
	UnitIdentifier    string   `json:"unit_identifier"`
	OwnerName         string   `json:"owner_name"`
	ScoreDrivers      []string `json:"score_drivers"`
	RecommendedAction string   `json:"recommended_action"`
}

func scoreAttentionItem(item attentionAccount, _ time.Time) AttentionItem {
	score := 0
	drivers := make([]string, 0, 4)

	switch {
	case item.DaysOverdue >= 90:
		score += 40
		drivers = append(drivers, "90+ days overdue")
	case item.DaysOverdue >= 30:
		score += 25
		drivers = append(drivers, "30+ days overdue")
	default:
		score += 10
	}

	switch {
	case item.OutstandingCents >= 700000:
		score += 25
		drivers = append(drivers, "high balance outstanding")
	case item.OutstandingCents >= 250000:
		score += 15
	default:
		score += 5
	}

	if item.LastActionDaysAgo >= 7 {
		score += 15
		drivers = append(drivers, "no recent follow-up")
	} else if item.LastActionType == "follow_up_logged" {
		score -= 10
	}

	recommended := "reminder_sent"
	if item.PromiseDateOverdue || item.DaysOverdue >= 90 {
		score += 20
		drivers = append(drivers, "broken promise or legal threshold")
		recommended = "legal_review_flagged"
	} else if item.LastActionType == "follow_up_logged" && item.LastActionDaysAgo <= 3 {
		recommended = "follow_up_logged"
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return AttentionItem{
		LevyAccountID:     item.LevyAccountID,
		SchemeID:          item.SchemeID,
		SchemeName:        item.SchemeName,
		UnitID:            item.UnitID,
		UnitIdentifier:    item.UnitIdentifier,
		OwnerName:         item.OwnerName,
		OutstandingCents:  item.OutstandingCents,
		DaysOverdue:       item.DaysOverdue,
		RiskScore:         score,
		ScoreDrivers:      drivers,
		RecommendedAction: recommended,
	}
}

type RecordCollectionEventInput struct {
	EventType          string
	PromiseAmountCents *int64
	PromiseDate        *time.Time
	Note               *string
}

type CollectionEvent struct {
	CreatedAt          string  `json:"created_at"`
	EventType          string  `json:"event_type"`
	ActorRole          string  `json:"actor_role"`
	ID                 string  `json:"id"`
	LevyAccountID      string  `json:"levy_account_id"`
	SchemeID           string  `json:"scheme_id"`
	Note               *string `json:"note"`
	PromiseAmountCents *int64  `json:"promise_amount_cents"`
	PromiseDate        *string `json:"promise_date"`
}

type AttentionQueueResponse struct {
	Items []AttentionItem `json:"items"`
	Scope string          `json:"scope"`
}

func validateCollectionEventInput(input RecordCollectionEventInput) error {
	switch input.EventType {
	case "reminder_sent", "follow_up_logged", "legal_review_flagged":
		return nil
	case "promise_to_pay":
		if input.PromiseAmountCents == nil || *input.PromiseAmountCents <= 0 || input.PromiseDate == nil {
			return ErrInvalidInput
		}
		return nil
	default:
		return ErrInvalidInput
	}
}

func sortAttentionItems(items []AttentionItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].RiskScore == items[j].RiskScore {
			if items[i].DaysOverdue == items[j].DaysOverdue {
				return items[i].OutstandingCents > items[j].OutstandingCents
			}
			return items[i].DaysOverdue > items[j].DaysOverdue
		}
		return items[i].RiskScore > items[j].RiskScore
	})
}
