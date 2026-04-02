package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/database"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Service struct {
	db        *database.Pool
	completer Completer
}

func NewService(db *database.Pool, completer Completer) *Service {
	return &Service{db: db, completer: completer}
}

func (s *Service) Ask(ctx context.Context, identity auth.Identity, schemeID string, history []Message, message string) (string, error) {
	if s.completer == nil {
		return "", ErrInvalidInput
	}
	if strings.TrimSpace(message) == "" {
		return "", ErrInvalidInput
	}
	if identity.Role == string(auth.RoleResident) {
		return "", ErrForbidden
	}

	var (
		systemPrompt string
		err          error
	)
	if strings.TrimSpace(schemeID) != "" {
		systemPrompt, err = s.buildSchemePrompt(ctx, identity, strings.TrimSpace(schemeID))
	} else {
		systemPrompt, err = s.buildPortfolioPrompt(ctx, identity)
	}
	if err != nil {
		return "", err
	}

	answer, err := s.completer.Complete(ctx, systemPrompt, history, message)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(answer), nil
}

func (s *Service) buildPortfolioPrompt(ctx context.Context, identity auth.Identity) (string, error) {
	if !auth.IsAdminRole(identity.Role) {
		return "", ErrForbidden
	}
	orgID, err := uuid.Parse(identity.OrgID)
	if err != nil {
		return "", ErrInvalidInput
	}

	schemes, err := s.db.Q.ListSchemesByOrg(ctx, orgID)
	if err != nil {
		return "", err
	}

	//nolint:govet // Keep this prompt payload grouped by semantic fields for readability.
	type schemeSummary struct {
		Name                 string `json:"name"`
		Address              string `json:"address"`
		UnitCount            int32  `json:"unit_count"`
		MemberCount          int    `json:"member_count"`
		TrusteeCount         int    `json:"trustee_count"`
		ResidentCount        int    `json:"resident_count"`
		OpenMaintenanceCount int    `json:"open_maintenance_count"`
		LevyCollectionPct    int    `json:"levy_collection_pct"`
		CurrentLevyPeriod    string `json:"current_levy_period,omitempty"`
	}

	summaries := make([]schemeSummary, 0, len(schemes))
	for _, scheme := range schemes {
		var members []dbgen.ListSchemeMembersBySchemeRow
		members, err = s.db.Q.ListSchemeMembersByScheme(ctx, scheme.ID)
		if err != nil {
			return "", err
		}
		maintenance, err := s.db.Q.ListMaintenanceRequestsByScheme(ctx, scheme.ID)
		if err != nil {
			return "", err
		}
		periods, err := s.db.Q.ListLevyPeriodsByScheme(ctx, scheme.ID)
		if err != nil {
			return "", err
		}

		var trusteeCount int
		var residentCount int
		for _, member := range members {
			switch member.Role {
			case string(auth.RoleTrustee):
				trusteeCount++
			case string(auth.RoleResident):
				residentCount++
			}
		}

		summary := schemeSummary{
			Name:                 scheme.Name,
			Address:              scheme.Address,
			UnitCount:            scheme.UnitCount,
			MemberCount:          len(members),
			TrusteeCount:         trusteeCount,
			ResidentCount:        residentCount,
			OpenMaintenanceCount: countOpenMaintenance(maintenance),
		}
		if len(periods) > 0 {
			var accounts []dbgen.ListLevyAccountsByPeriodRow
			accounts, err = s.db.Q.ListLevyAccountsByPeriod(ctx, periods[0].ID)
			if err != nil {
				return "", err
			}
			summary.CurrentLevyPeriod = periods[0].Label
			summary.LevyCollectionPct = collectionPctFromRows(accounts)
		}
		summaries = append(summaries, summary)
	}

	payload, err := json.MarshalIndent(map[string]any{
		"scope":   "portfolio",
		"role":    identity.Role,
		"schemes": summaries,
	}, "", "  ")
	if err != nil {
		return "", err
	}

	return buildSystemPrompt(string(payload), "portfolio"), nil
}

func (s *Service) buildSchemePrompt(ctx context.Context, identity auth.Identity, schemeID string) (string, error) {
	scheme, role, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return "", err
	}

	units, err := s.db.Q.ListUnitsByScheme(ctx, scheme.ID)
	if err != nil {
		return "", err
	}
	members, err := s.db.Q.ListSchemeMembersByScheme(ctx, scheme.ID)
	if err != nil {
		return "", err
	}
	maintenance, err := s.db.Q.ListMaintenanceRequestsByScheme(ctx, scheme.ID)
	if err != nil {
		return "", err
	}
	notices, err := s.db.Q.ListNoticesByScheme(ctx, scheme.ID)
	if err != nil {
		return "", err
	}
	budgetLines, err := s.db.Q.ListBudgetLinesByScheme(ctx, scheme.ID)
	if err != nil {
		return "", err
	}
	periods, err := s.db.Q.ListLevyPeriodsByScheme(ctx, scheme.ID)
	if err != nil {
		return "", err
	}
	meetings, err := s.db.Q.ListAgmMeetingsByScheme(ctx, scheme.ID)
	if err != nil {
		return "", err
	}

	reserveFund, reserveErr := s.db.Q.GetReserveFund(ctx, scheme.ID)
	if reserveErr != nil && !errors.Is(reserveErr, pgx.ErrNoRows) {
		return "", reserveErr
	}

	levySummary := map[string]any{}
	if len(periods) > 0 {
		var accounts []dbgen.ListLevyAccountsByPeriodRow
		accounts, err = s.db.Q.ListLevyAccountsByPeriod(ctx, periods[0].ID)
		if err != nil {
			return "", err
		}
		levySummary = map[string]any{
			"current_period":      periods[0].Label,
			"due_date":            formatPgDate(periods[0].DueDate),
			"amount_cents":        periods[0].AmountCents,
			"collection_rate_pct": collectionPctFromRows(accounts),
			"overdue_accounts":    topOverdueAccounts(accounts, 5),
		}
	}

	agmSummary, err := s.buildAgmSummary(ctx, meetings)
	if err != nil {
		return "", err
	}

	payload := map[string]any{
		"scope": "scheme",
		"role":  role,
		"scheme": map[string]any{
			"id":         scheme.ID.String(),
			"name":       scheme.Name,
			"address":    scheme.Address,
			"unit_count": scheme.UnitCount,
		},
		"units":             mapUnits(units, 8),
		"members":           mapMembers(members, 12),
		"maintenance":       mapMaintenance(maintenance, 10),
		"recent_notices":    mapNotices(notices, 6),
		"financials":        mapFinancials(budgetLines, reserveErr == nil, reserveFund),
		"levy":              levySummary,
		"agm":               agmSummary,
		"today":             time.Now().Format("2006-01-02"),
		"available_actions": []string{"scheme_qna", "levy_reconciliation"},
		"action_guidance":   levyActionGuidance(),
	}

	contextJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return buildSystemPrompt(string(contextJSON), "scheme"), nil
}

func (s *Service) resolveSchemeAccess(ctx context.Context, identity auth.Identity, schemeID string) (dbgen.Scheme, string, error) {
	id, err := uuid.Parse(schemeID)
	if err != nil {
		return dbgen.Scheme{}, "", ErrInvalidInput
	}

	scheme, err := s.db.Q.GetScheme(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dbgen.Scheme{}, "", ErrNotFound
		}
		return dbgen.Scheme{}, "", err
	}

	if auth.IsAdminRole(identity.Role) {
		orgID, parseErr := uuid.Parse(identity.OrgID)
		if parseErr != nil {
			return dbgen.Scheme{}, "", ErrInvalidInput
		}
		if scheme.OrgID != orgID {
			return dbgen.Scheme{}, "", ErrForbidden
		}
		return scheme, string(auth.RoleAdmin), nil
	}

	userID, parseErr := uuid.Parse(identity.UserID)
	if parseErr != nil {
		return dbgen.Scheme{}, "", ErrInvalidInput
	}
	membership, err := s.db.Q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{
		UserID:   userID,
		SchemeID: id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dbgen.Scheme{}, "", ErrForbidden
		}
		return dbgen.Scheme{}, "", err
	}

	if membership.Role == string(auth.RoleResident) {
		return dbgen.Scheme{}, "", ErrForbidden
	}
	return scheme, membership.Role, nil
}

func (s *Service) buildAgmSummary(ctx context.Context, meetings []dbgen.AgmMeeting) (map[string]any, error) {
	if len(meetings) == 0 {
		return map[string]any{}, nil
	}
	sort.Slice(meetings, func(i, j int) bool {
		return meetings[i].MeetingDate.Time.After(meetings[j].MeetingDate.Time)
	})

	var latest map[string]any
	var upcoming map[string]any
	now := startOfDay(time.Now())
	for _, meeting := range meetings {
		resolutions, err := s.db.Q.ListAgmResolutionsByMeeting(ctx, meeting.ID)
		if err != nil {
			return nil, err
		}
		item := map[string]any{
			"date":            meeting.MeetingDate.Time.Format("2006-01-02"),
			"status":          string(meeting.Status),
			"quorum_required": meeting.QuorumRequired,
			"quorum_present":  meeting.QuorumPresent,
			"resolutions":     mapResolutions(resolutions, 6),
		}
		if (meeting.MeetingDate.Time.Before(now) || meeting.Status == dbgen.AgmStatusClosed) && latest == nil {
			latest = item
			continue
		}
		if upcoming == nil {
			upcoming = item
		}
	}

	return map[string]any{
		"latest":   latest,
		"upcoming": upcoming,
	}, nil
}

func buildSystemPrompt(contextJSON string, scope string) string {
	return fmt.Sprintf(`You are StrataHQ Copilot, an assistant for South African sectional title scheme operations.

You must answer using only the live data provided below. Do not invent facts, amounts, dates, member names, balances, statuses, or scheme metrics.

Rules:
- Be concise and specific.
- Use South African Rand formatting like R 2 450.
- If the answer is not in the provided data, say that it is not available in the current workspace context.
- If the question is about levy reconciliation, explain which units/accounts appear overdue or partially paid and give the next operator action.
- If the user asks for a draft notice, letter, or report, draft it professionally using the scheme context below.
- Never claim to have executed a financial or operational action. You may recommend actions, but you do not perform them.
- Today is %s.
- Current scope is %s.

LIVE DATA:
%s`, time.Now().Format("2 January 2006"), scope, contextJSON)
}

func levyActionGuidance() map[string]string {
	return map[string]string{
		"scheme_qna":          "Answer factual questions about the loaded scheme or portfolio using the live data only.",
		"levy_reconciliation": "When asked about reconciliation, focus on overdue or partially paid levy accounts and recommend the next operator step.",
	}
}

func countOpenMaintenance(requests []dbgen.MaintenanceRequest) int {
	var count int
	for _, request := range requests {
		if request.Status != dbgen.MaintenanceStatusResolved {
			count++
		}
	}
	return count
}

func collectionPctFromRows(accounts []dbgen.ListLevyAccountsByPeriodRow) int {
	if len(accounts) == 0 {
		return 0
	}
	var totalDue int64
	var totalPaid int64
	for _, account := range accounts {
		totalDue += account.AmountCents
		totalPaid += minInt64(account.PaidCents, account.AmountCents)
	}
	if totalDue == 0 {
		return 0
	}
	return int(math.Round(float64(totalPaid) * 100 / float64(totalDue)))
}

func topOverdueAccounts(accounts []dbgen.ListLevyAccountsByPeriodRow, limit int) []map[string]any {
	items := make([]map[string]any, 0, limit)
	for _, account := range accounts {
		status := levyStatusFor(account.PaidCents, account.AmountCents, account.DueDate)
		if status == "paid" || status == "pending" {
			continue
		}
		items = append(items, map[string]any{
			"unit_identifier":   account.UnitIdentifier,
			"owner_name":        account.OwnerName,
			"status":            status,
			"amount_cents":      account.AmountCents,
			"paid_cents":        account.PaidCents,
			"outstanding_cents": maxInt64(account.AmountCents-account.PaidCents, 0),
			"due_date":          formatPgDate(account.DueDate),
		})
		if len(items) == limit {
			break
		}
	}
	return items
}

func mapUnits(units []dbgen.Unit, limit int) []map[string]any {
	items := make([]map[string]any, 0, min(limit, len(units)))
	for i, unit := range units {
		if i == limit {
			break
		}
		items = append(items, map[string]any{
			"identifier":        unit.Identifier,
			"owner_name":        unit.OwnerName,
			"floor":             unit.Floor,
			"section_value_bps": unit.SectionValueBps,
		})
	}
	return items
}

func mapMembers(members []dbgen.ListSchemeMembersBySchemeRow, limit int) []map[string]any {
	items := make([]map[string]any, 0, min(limit, len(members)))
	for i, member := range members {
		if i == limit {
			break
		}
		entry := map[string]any{
			"full_name": member.FullName,
			"email":     member.Email,
			"role":      member.Role,
		}
		if member.UnitID.Valid {
			entry["unit_id"] = uuid.UUID(member.UnitID.Bytes).String()
		}
		if member.UnitIdentifier.Valid {
			entry["unit_identifier"] = member.UnitIdentifier.String
		}
		items = append(items, entry)
	}
	return items
}

func mapMaintenance(requests []dbgen.MaintenanceRequest, limit int) []map[string]any {
	items := make([]map[string]any, 0, min(limit, len(requests)))
	for i, request := range requests {
		if i == limit {
			break
		}
		items = append(items, map[string]any{
			"title":             request.Title,
			"category":          string(request.Category),
			"status":            string(request.Status),
			"submitted_by_unit": textPointer(request.SubmittedByUnit),
			"contractor_name":   textPointer(request.ContractorName),
			"sla_hours":         request.SlaHours,
			"created_at":        request.CreatedAt.Format(time.RFC3339),
		})
	}
	return items
}

func mapNotices(notices []dbgen.Notice, limit int) []map[string]any {
	items := make([]map[string]any, 0, min(limit, len(notices)))
	for i, notice := range notices {
		if i == limit {
			break
		}
		items = append(items, map[string]any{
			"title":   notice.Title,
			"type":    string(notice.Type),
			"sent_at": notice.SentAt.Format(time.RFC3339),
		})
	}
	return items
}

func mapFinancials(lines []dbgen.BudgetLine, hasReserve bool, reserve dbgen.ReserveFund) map[string]any {
	periods := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	var totalBudgeted int64
	var totalActual int64
	for _, line := range lines {
		totalBudgeted += line.BudgetedCents
		totalActual += line.ActualCents
		if _, ok := seen[line.PeriodLabel]; !ok {
			seen[line.PeriodLabel] = struct{}{}
			periods = append(periods, line.PeriodLabel)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(periods)))
	payload := map[string]any{
		"budget_periods":       periods,
		"budget_lines":         mapBudgetLines(lines, 8),
		"total_budgeted_cents": totalBudgeted,
		"total_actual_cents":   totalActual,
	}
	if hasReserve {
		payload["reserve_fund"] = map[string]any{
			"balance_cents": reserve.BalanceCents,
			"target_cents":  reserve.TargetCents,
			"last_updated":  reserve.LastUpdated.Format(time.RFC3339),
		}
	}
	return payload
}

func mapBudgetLines(lines []dbgen.BudgetLine, limit int) []map[string]any {
	items := make([]map[string]any, 0, min(limit, len(lines)))
	for i, line := range lines {
		if i == limit {
			break
		}
		items = append(items, map[string]any{
			"category":       line.Category,
			"period_label":   line.PeriodLabel,
			"budgeted_cents": line.BudgetedCents,
			"actual_cents":   line.ActualCents,
			"variance_cents": line.BudgetedCents - line.ActualCents,
		})
	}
	return items
}

func mapResolutions(resolutions []dbgen.AgmResolution, limit int) []map[string]any {
	items := make([]map[string]any, 0, min(limit, len(resolutions)))
	for i, resolution := range resolutions {
		if i == limit {
			break
		}
		items = append(items, map[string]any{
			"title":          resolution.Title,
			"description":    resolution.Description,
			"votes_for":      resolution.VotesFor,
			"votes_against":  resolution.VotesAgainst,
			"total_eligible": resolution.TotalEligible,
			"status":         string(resolution.Status),
		})
	}
	return items
}

func levyStatusFor(paidCents, amountCents int64, dueDate pgtype.Date) string {
	switch {
	case paidCents >= amountCents && amountCents > 0:
		return "paid"
	case paidCents > 0:
		return "partial"
	case dueDate.Valid && dueDate.Time.Before(startOfDay(time.Now())):
		return "overdue"
	default:
		return "pending"
	}
}

func formatPgDate(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	copy := value.String
	return &copy
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
