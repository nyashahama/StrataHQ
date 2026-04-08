package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/platform/database"
)

type Bot struct {
	db *database.Pool
}

func NewBot(db *database.Pool) *Bot {
	return &Bot{db: db}
}

func (b *Bot) Respond(ctx context.Context, schemeID, unitID uuid.UUID, incomingText string) (string, error) {
	text := strings.TrimSpace(strings.ToLower(incomingText))

	switch {
	case text == "":
		return helpMenu(), nil
	case text == "menu", text == "hi", text == "hello", text == "hey", text == "start":
		return helpMenu(), nil
	case text == "1", text == "balance", strings.HasPrefix(text, "balance"):
		return b.levyBalance(ctx, schemeID, unitID)
	case text == "2", text == "request", strings.HasPrefix(text, "request"):
		return b.logMaintenanceRequest(ctx, schemeID, unitID, incomingText)
	case text == "3", text == "notices", strings.HasPrefix(text, "notices"):
		return b.recentNotices(ctx, schemeID)
	default:
		return helpMenu(), nil
	}
}

func helpMenu() string {
	return "Welcome to your scheme on WhatsApp.\n\nReply with:\n1 Balance - check your levy account\n2 Request - log a maintenance request\n3 Notices - see recent scheme notices"
}

func (b *Bot) levyBalance(ctx context.Context, schemeID, unitID uuid.UUID) (string, error) {
	period, err := b.db.Q.GetLatestLevyPeriodByScheme(ctx, schemeID)
	if err != nil {
		return "Could not retrieve levy information at this time. Please try again later.", nil
	}

	account, err := b.db.Q.GetLevyAccountByUnitAndPeriod(ctx, dbgen.GetLevyAccountByUnitAndPeriodParams{
		UnitID:   unitID,
		PeriodID: period.ID,
	})
	if err != nil {
		return "Could not retrieve your levy account. Please contact your managing agent.", nil
	}

	outstanding := account.AmountCents - account.PaidCents
	dueDate := "N/A"
	if account.DueDate.Valid {
		dueDate = account.DueDate.Time.Format("2 Jan 2006")
	}

	return fmt.Sprintf(
		"Unit levy account for %s:\nMonthly levy: R %s\nAmount paid: R %s\nOutstanding: R %s\nDue date: %s",
		period.Label,
		centsToRands(account.AmountCents),
		centsToRands(account.PaidCents),
		centsToRands(outstanding),
		dueDate,
	), nil
}

func (b *Bot) logMaintenanceRequest(ctx context.Context, schemeID, unitID uuid.UUID, incomingText string) (string, error) {
	description := incomingText
	title := "WhatsApp maintenance request"

	mr, err := b.db.Q.CreateMaintenanceRequest(ctx, dbgen.CreateMaintenanceRequestParams{
		SchemeID:        schemeID,
		UnitID:          pgtype.UUID{Bytes: unitID, Valid: true},
		Title:           title,
		Description:     description,
		Category:        dbgen.MaintenanceCategoryOther,
		SlaHours:        72,
		SubmittedByUnit: pgtype.Text{},
	})
	if err != nil {
		return "Could not log your maintenance request. Please try again later.", nil
	}

	return fmt.Sprintf("Thanks. I've logged a maintenance request on your behalf.\n\nRef: %s\nStatus: Open\n\nWe'll keep you posted.", mr.ID.String()[:8]), nil
}

func (b *Bot) recentNotices(ctx context.Context, schemeID uuid.UUID) (string, error) {
	notices, err := b.db.Q.ListNoticesByScheme(ctx, schemeID)
	if err != nil {
		return "Could not retrieve notices at this time.", nil
	}

	if len(notices) == 0 {
		return "No recent notices for your scheme.", nil
	}

	var sb strings.Builder
	sb.WriteString("Recent notices:\n\n")
	for i, n := range notices {
		if i >= 3 {
			break
		}
		date := n.SentAt.Format("2 Jan 2006")
		sb.WriteString(fmt.Sprintf("• %s (%s) - %s\n", n.Title, date, n.Type))
	}

	if len(notices) > 3 {
		sb.WriteString(fmt.Sprintf("\n...and %d more. Check your app for details.", len(notices)-3))
	}

	return sb.String(), nil
}

func (b *Bot) WelcomeMessage(schemeName string) string {
	return fmt.Sprintf("Hi! Welcome to %s on WhatsApp.\n\nReply with:\n1 Balance - check your levy account\n2 Request - log a maintenance request\n3 Notices - see recent scheme notices", schemeName)
}

func (b *Bot) AcknowledgePayment(unitIdentifier string) string {
	return fmt.Sprintf("Payment confirmed for Unit %s. Thank you!", unitIdentifier)
}

func centsToRands(cents int64) string {
	rands := float64(cents) / 100.0
	return fmt.Sprintf("%.2f", rands)
}

func FormatReference(unitIdentifier, periodLabel string) string {
	cleanUnit := strings.ReplaceAll(strings.ToUpper(unitIdentifier), " ", "")
	cleanPeriod := strings.ReplaceAll(strings.ToUpper(periodLabel), " ", "")
	return fmt.Sprintf("SH-%s-%s", cleanUnit, cleanPeriod)
}

func TimeNowFunc() time.Time {
	return time.Now().UTC()
}
