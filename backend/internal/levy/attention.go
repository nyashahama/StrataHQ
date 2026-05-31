package levy

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type attentionAccount struct {
	PromiseDateOverdue bool
	HasActivePromise   bool
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

const legalReviewCooldownDays = 7

func scoreAttentionItem(item attentionAccount, _ time.Time) AttentionItem {
	score := 0
	drivers := make([]string, 0, 4)
	recentLegalReview := item.LastActionType == "legal_review_flagged" && item.LastActionDaysAgo <= legalReviewCooldownDays

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

	if item.LastActionType == "reminder_sent" && item.LastActionDaysAgo <= 2 {
		score -= 20
		drivers = append(drivers, "recent reminder already sent")
	}

	if recentLegalReview {
		score -= 25
		drivers = append(drivers, "recent legal review already flagged")
	}

	recommended := "reminder_sent"
	if item.LastActionType == "reminder_sent" && item.LastActionDaysAgo <= 2 {
		recommended = "follow_up_logged"
	} else if recentLegalReview {
		recommended = "follow_up_logged"
	} else if item.HasActivePromise {
		score -= 20
		drivers = append(drivers, "active promise to pay")
		recommended = "active_promise"
	} else if item.PromiseDateOverdue || item.DaysOverdue >= 90 {
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

type ReminderDelivery struct {
	To      string
	Subject string
	Body    string
	Status  string
	Error   string
}

type RecordCollectionEventInput struct {
	EventType          string
	PromiseAmountCents *int64
	PromiseDate        *time.Time
	Note               *string
	Email              ReminderDelivery
	WhatsApp           ReminderDelivery
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

func normalizeSendReminderInput(input SendReminderInput) SendReminderInput {
	input.Email.Subject = strings.TrimSpace(input.Email.Subject)
	input.Email.Body = strings.TrimSpace(input.Email.Body)
	input.WhatsApp.Body = strings.TrimSpace(input.WhatsApp.Body)
	return input
}

func validateSendReminderInput(input SendReminderInput) error {
	input = normalizeSendReminderInput(input)
	if !input.Email.Enabled && !input.WhatsApp.Enabled {
		return ErrInvalidInput
	}
	if input.Email.Enabled && (input.Email.Subject == "" || input.Email.Body == "") {
		return ErrInvalidInput
	}
	if input.WhatsApp.Enabled && input.WhatsApp.Body == "" {
		return ErrInvalidInput
	}
	return nil
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

type ReminderChannelDraft struct {
	To             string `json:"to"`
	Subject        string `json:"subject,omitempty"`
	Body           string `json:"body"`
	DisabledReason string `json:"disabled_reason,omitempty"`
	Enabled        bool   `json:"enabled"`
}

type ReminderDraftResponse struct {
	AccountID  string               `json:"account_id"`
	SchemeID   string               `json:"scheme_id"`
	SchemeName string               `json:"scheme_name"`
	UnitLabel  string               `json:"unit_label"`
	OwnerName  string               `json:"owner_name"`
	Email      ReminderChannelDraft `json:"email"`
	WhatsApp   ReminderChannelDraft `json:"whatsapp"`
}

func formatCurrency(cents int64) string {
	rands := cents / 100
	return fmt.Sprintf("R %d %02d.00", rands/1000, rands%1000)
}

func buildReminderDraft(item attentionAccount, email, whatsappPhone string, whatsappConnected bool) ReminderDraftResponse {
	emailDraft := ReminderChannelDraft{
		To:      email,
		Subject: fmt.Sprintf("Levy arrears reminder for %s", item.SchemeName),
		Body: fmt.Sprintf(
			"Hi %s,\n\nOur records show that Unit %s at %s has an outstanding levy balance of %s that is now %d days overdue.\n\nPlease arrange payment or contact the scheme team to discuss the next step.\n",
			item.OwnerName,
			item.UnitIdentifier,
			item.SchemeName,
			formatCurrency(item.OutstandingCents),
			item.DaysOverdue,
		),
		Enabled: email != "",
	}
	if !emailDraft.Enabled {
		emailDraft.DisabledReason = "No email on file"
	}

	whatsAppDraft := ReminderChannelDraft{
		To:      whatsappPhone,
		Body:    fmt.Sprintf("Hi %s, this is a reminder that Unit %s at %s has an overdue levy balance of %s. Please reply if you need to discuss the next step.", item.OwnerName, item.UnitIdentifier, item.SchemeName, formatCurrency(item.OutstandingCents)),
		Enabled: whatsappConnected && whatsappPhone != "",
	}
	if !whatsAppDraft.Enabled {
		whatsAppDraft.DisabledReason = "No WhatsApp number or active thread"
	}

	return ReminderDraftResponse{
		AccountID:  item.LevyAccountID,
		SchemeID:   item.SchemeID,
		SchemeName: item.SchemeName,
		UnitLabel:  "Unit " + item.UnitIdentifier,
		OwnerName:  item.OwnerName,
		Email:      emailDraft,
		WhatsApp:   whatsAppDraft,
	}
}
