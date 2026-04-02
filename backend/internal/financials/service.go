package financials

import (
	"context"
	"errors"
	"math"
	"slices"
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

//nolint:govet // Keep response DTO fields grouped by API meaning rather than field packing.
type BudgetLineInfo struct {
	ID            string    `json:"id"`
	SchemeID      string    `json:"scheme_id"`
	Category      string    `json:"category"`
	PeriodLabel   string    `json:"period_label"`
	BudgetedCents int64     `json:"budgeted_cents"`
	ActualCents   int64     `json:"actual_cents"`
	VarianceCents int64     `json:"variance_cents"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

//nolint:govet // Keep response DTO fields grouped by API meaning rather than field packing.
type ReserveFundInfo struct {
	SchemeID     string    `json:"scheme_id"`
	BalanceCents int64     `json:"balance_cents"`
	TargetCents  int64     `json:"target_cents"`
	LastUpdated  time.Time `json:"last_updated"`
}

type LevySummaryInfo struct {
	PeriodLabel         string `json:"period_label"`
	TotalBilledCents    int64  `json:"total_billed_cents"`
	TotalCollectedCents int64  `json:"total_collected_cents"`
	CollectionRatePct   int    `json:"collection_rate_pct"`
	OverdueCount        int    `json:"overdue_count"`
}

//nolint:govet // Keep response DTO fields grouped by API meaning rather than field packing.
type DashboardResponse struct {
	ReserveFund        *ReserveFundInfo `json:"reserve_fund"`
	LevySummary        *LevySummaryInfo `json:"levy_summary"`
	BudgetLines        []BudgetLineInfo `json:"budget_lines"`
	AvailablePeriods   []string         `json:"available_periods"`
	Role               string           `json:"role"`
	SelectedPeriod     string           `json:"selected_period"`
	TotalBudgetedCents int64            `json:"total_budgeted_cents"`
	TotalActualCents   int64            `json:"total_actual_cents"`
	SurplusCents       int64            `json:"surplus_cents"`
}

//nolint:govet // Keep input DTO fields grouped by domain meaning rather than field packing.
type UpsertBudgetLineInput struct {
	BudgetedCents int64
	ActualCents   int64
	Category      string
	PeriodLabel   string
}

type UpdateReserveFundInput struct {
	BalanceCents int64
	TargetCents  int64
}

type Service struct {
	db *database.Pool
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) Dashboard(ctx context.Context, identity auth.Identity, schemeID, periodLabel string) (*DashboardResponse, error) {
	scheme, role, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	resp := &DashboardResponse{
		Role:             role,
		BudgetLines:      []BudgetLineInfo{},
		AvailablePeriods: []string{},
	}

	allLines, err := s.db.Q.ListBudgetLinesByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}

	if len(allLines) > 0 {
		periods := make([]string, 0, len(allLines))
		seen := make(map[string]struct{}, len(allLines))
		for _, line := range allLines {
			if _, ok := seen[line.PeriodLabel]; ok {
				continue
			}
			seen[line.PeriodLabel] = struct{}{}
			periods = append(periods, line.PeriodLabel)
		}
		slices.Sort(periods)
		slices.Reverse(periods)
		resp.AvailablePeriods = periods

		selected := periodLabel
		if selected == "" {
			selected = periods[0]
		}
		resp.SelectedPeriod = selected

		for _, line := range allLines {
			if line.PeriodLabel != selected {
				continue
			}
			mapped := mapBudgetLine(line)
			resp.BudgetLines = append(resp.BudgetLines, mapped)
			resp.TotalBudgetedCents += mapped.BudgetedCents
			resp.TotalActualCents += mapped.ActualCents
		}
		resp.SurplusCents = resp.TotalBudgetedCents - resp.TotalActualCents
	}

	reserve, reserveErr := s.db.Q.GetReserveFund(ctx, scheme.ID)
	if reserveErr == nil {
		mapped := mapReserveFund(reserve)
		resp.ReserveFund = &mapped
	} else if !errors.Is(reserveErr, pgx.ErrNoRows) {
		return nil, reserveErr
	}

	levySummary, levyErr := s.buildLevySummary(ctx, scheme.ID)
	if levyErr != nil {
		return nil, levyErr
	}
	resp.LevySummary = levySummary

	return resp, nil
}

func (s *Service) UpsertBudgetLine(ctx context.Context, identity auth.Identity, schemeID string, input UpsertBudgetLineInput) (*BudgetLineInfo, error) {
	scheme, role, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if role == string(auth.RoleResident) {
		return nil, ErrForbidden
	}
	if input.Category == "" || input.PeriodLabel == "" || input.BudgetedCents < 0 || input.ActualCents < 0 {
		return nil, ErrInvalidInput
	}

	line, err := s.db.Q.UpsertBudgetLine(ctx, dbgen.UpsertBudgetLineParams{
		SchemeID:      scheme.ID,
		Category:      input.Category,
		PeriodLabel:   input.PeriodLabel,
		BudgetedCents: input.BudgetedCents,
		ActualCents:   input.ActualCents,
	})
	if err != nil {
		return nil, err
	}

	mapped := mapBudgetLine(line)
	return &mapped, nil
}

func (s *Service) UpdateReserveFund(ctx context.Context, identity auth.Identity, schemeID string, input UpdateReserveFundInput) (*ReserveFundInfo, error) {
	scheme, role, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if role == string(auth.RoleResident) {
		return nil, ErrForbidden
	}
	if input.BalanceCents < 0 || input.TargetCents < 0 {
		return nil, ErrInvalidInput
	}

	reserve, err := s.db.Q.UpsertReserveFund(ctx, dbgen.UpsertReserveFundParams{
		SchemeID:     scheme.ID,
		BalanceCents: input.BalanceCents,
		TargetCents:  input.TargetCents,
	})
	if err != nil {
		return nil, err
	}

	mapped := mapReserveFund(reserve)
	return &mapped, nil
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

	return scheme, membership.Role, nil
}

func (s *Service) buildLevySummary(ctx context.Context, schemeID uuid.UUID) (*LevySummaryInfo, error) {
	periods, err := s.db.Q.ListLevyPeriodsByScheme(ctx, schemeID)
	if err != nil {
		return nil, err
	}
	if len(periods) == 0 {
		return nil, nil
	}

	currentPeriod := periods[0]
	accounts, err := s.db.Q.ListLevyAccountsByPeriod(ctx, currentPeriod.ID)
	if err != nil {
		return nil, err
	}

	summary := &LevySummaryInfo{
		PeriodLabel: currentPeriod.Label,
	}
	for _, account := range accounts {
		summary.TotalBilledCents += account.AmountCents
		summary.TotalCollectedCents += minInt64(account.PaidCents, account.AmountCents)
		if levyStatusFor(account.PaidCents, account.AmountCents, account.DueDate) == "overdue" {
			summary.OverdueCount++
		}
	}
	if summary.TotalBilledCents > 0 {
		summary.CollectionRatePct = int(math.Round(float64(summary.TotalCollectedCents) * 100 / float64(summary.TotalBilledCents)))
	}
	return summary, nil
}

func mapBudgetLine(line dbgen.BudgetLine) BudgetLineInfo {
	return BudgetLineInfo{
		ID:            line.ID.String(),
		SchemeID:      line.SchemeID.String(),
		Category:      line.Category,
		PeriodLabel:   line.PeriodLabel,
		BudgetedCents: line.BudgetedCents,
		ActualCents:   line.ActualCents,
		VarianceCents: line.BudgetedCents - line.ActualCents,
		CreatedAt:     line.CreatedAt,
		UpdatedAt:     line.UpdatedAt,
	}
}

func mapReserveFund(reserve dbgen.ReserveFund) ReserveFundInfo {
	return ReserveFundInfo{
		SchemeID:     reserve.SchemeID.String(),
		BalanceCents: reserve.BalanceCents,
		TargetCents:  reserve.TargetCents,
		LastUpdated:  reserve.LastUpdated,
	}
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
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

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}
