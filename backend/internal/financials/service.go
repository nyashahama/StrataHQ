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
type LevyForecastPointInfo struct {
	PeriodLabel       string `json:"period_label"`
	BilledCents       int64  `json:"billed_cents"`
	CollectedCents    int64  `json:"collected_cents"`
	CollectionRatePct int    `json:"collection_rate_pct"`
	ExpenseCents      int64  `json:"expense_cents"`
}

//nolint:govet // Keep response DTO fields grouped by API meaning rather than field packing.
type LevyForecastInfo struct {
	DataPoints                    []LevyForecastPointInfo `json:"data_points"`
	Notes                         []string                `json:"notes"`
	Status                        string                  `json:"status"`
	Confidence                    string                  `json:"confidence"`
	MonthsProjected               int                     `json:"months_projected"`
	CurrentMonthlyLevyCents       int64                   `json:"current_monthly_levy_cents"`
	AverageCollectionRatePct      int                     `json:"average_collection_rate_pct"`
	AverageMonthlyIncomeCents     int64                   `json:"average_monthly_income_cents"`
	AverageMonthlyExpenseCents    int64                   `json:"average_monthly_expense_cents"`
	ProjectedReserveBalanceCents  int64                   `json:"projected_reserve_balance_cents"`
	ProjectedShortfallCents       int64                   `json:"projected_shortfall_cents"`
	RecommendedMonthlyIncreaseCents int64                 `json:"recommended_monthly_increase_cents"`
	RecommendedIncreasePct        int                     `json:"recommended_increase_pct"`
}

//nolint:govet // Keep response DTO fields grouped by API meaning rather than field packing.
type DashboardResponse struct {
	LevyForecast        *LevyForecastInfo `json:"levy_forecast"`
	ReserveFund        *ReserveFundInfo  `json:"reserve_fund"`
	LevySummary        *LevySummaryInfo  `json:"levy_summary"`
	BudgetLines        []BudgetLineInfo  `json:"budget_lines"`
	AvailablePeriods   []string          `json:"available_periods"`
	Role               string            `json:"role"`
	SelectedPeriod     string            `json:"selected_period"`
	TotalBudgetedCents int64             `json:"total_budgeted_cents"`
	TotalActualCents   int64             `json:"total_actual_cents"`
	SurplusCents       int64             `json:"surplus_cents"`
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

type levyForecastPointInput struct {
	PeriodLabel    string
	BilledCents    int64
	CollectedCents int64
	ExpenseCents   int64
	HasExpense     bool
}

type levyForecastInput struct {
	Points                     []levyForecastPointInput
	MonthsProjected            int
	CurrentReserveBalanceCents int64
	ReserveTargetCents         int64
	CurrentMonthlyLevyCents    int64
	LatestUnitCount            int
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

	var reserveInfo *ReserveFundInfo
	reserve, reserveErr := s.db.Q.GetReserveFund(ctx, scheme.ID)
	if reserveErr == nil {
		mapped := mapReserveFund(reserve)
		reserveInfo = &mapped
		resp.ReserveFund = reserveInfo
	} else if !errors.Is(reserveErr, pgx.ErrNoRows) {
		return nil, reserveErr
	}

	levySummary, levyErr := s.buildLevySummary(ctx, scheme.ID)
	if levyErr != nil {
		return nil, levyErr
	}
	resp.LevySummary = levySummary

	if role != string(auth.RoleResident) {
		forecast, forecastErr := s.buildLevyForecastForScheme(ctx, scheme.ID, reserveInfo, allLines)
		if forecastErr != nil {
			return nil, forecastErr
		}
		resp.LevyForecast = forecast
	}

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

func (s *Service) buildLevyForecastForScheme(ctx context.Context, schemeID uuid.UUID, reserve *ReserveFundInfo, budgetLines []dbgen.BudgetLine) (*LevyForecastInfo, error) {
	periods, err := s.db.Q.ListLevyPeriodsByScheme(ctx, schemeID)
	if err != nil {
		return nil, err
	}
	if len(periods) == 0 {
		return nil, nil
	}
	if len(periods) > 12 {
		periods = periods[:12]
	}

	expenseByPeriod := make(map[string]int64)
	for _, line := range budgetLines {
		expenseByPeriod[line.PeriodLabel] += line.ActualCents
	}

	input := levyForecastInput{
		MonthsProjected:         12,
		CurrentMonthlyLevyCents: periods[0].AmountCents,
	}
	if reserve != nil {
		input.CurrentReserveBalanceCents = reserve.BalanceCents
		input.ReserveTargetCents = reserve.TargetCents
	}

	for index, period := range periods {
		accounts, accountErr := s.db.Q.ListLevyAccountsByPeriod(ctx, period.ID)
		if accountErr != nil {
			return nil, accountErr
		}
		if index == 0 {
			input.LatestUnitCount = len(accounts)
		}
		point := levyForecastPointInput{PeriodLabel: period.Label}
		for _, account := range accounts {
			point.BilledCents += account.AmountCents
			point.CollectedCents += minInt64(account.PaidCents, account.AmountCents)
		}
		if expense, ok := expenseByPeriod[period.Label]; ok {
			point.ExpenseCents = expense
			point.HasExpense = true
		}
		input.Points = append(input.Points, point)
	}

	forecast := buildLevyForecast(input)
	if forecast != nil && reserve == nil {
		forecast.Notes = append(forecast.Notes, "No reserve fund is set; projection starts from R0.")
	}
	return forecast, nil
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

func buildLevyForecast(input levyForecastInput) *LevyForecastInfo {
	if len(input.Points) == 0 {
		return nil
	}
	months := input.MonthsProjected
	if months <= 0 {
		months = 12
	}

	forecast := &LevyForecastInfo{
		Status:                  "healthy",
		Confidence:              "low",
		MonthsProjected:         months,
		CurrentMonthlyLevyCents: input.CurrentMonthlyLevyCents,
		DataPoints:              make([]LevyForecastPointInfo, 0, len(input.Points)),
		Notes:                   []string{},
	}

	var totalBilled, totalCollected, totalExpense int64
	expensePoints := 0
	for _, point := range input.Points {
		if point.BilledCents <= 0 {
			continue
		}
		totalBilled += point.BilledCents
		totalCollected += minInt64(point.CollectedCents, point.BilledCents)
		if point.HasExpense {
			totalExpense += point.ExpenseCents
			expensePoints++
		}
		collectionPct := int(math.Round(float64(minInt64(point.CollectedCents, point.BilledCents)) * 100 / float64(point.BilledCents)))
		forecast.DataPoints = append(forecast.DataPoints, LevyForecastPointInfo{
			PeriodLabel:       point.PeriodLabel,
			BilledCents:       point.BilledCents,
			CollectedCents:    minInt64(point.CollectedCents, point.BilledCents),
			CollectionRatePct: collectionPct,
			ExpenseCents:      point.ExpenseCents,
		})
	}

	if totalBilled == 0 || len(forecast.DataPoints) == 0 {
		return nil
	}

	forecast.AverageCollectionRatePct = int(math.Round(float64(totalCollected) * 100 / float64(totalBilled)))
	forecast.AverageMonthlyIncomeCents = int64(math.Round(float64(totalCollected) / float64(len(forecast.DataPoints))))
	if expensePoints > 0 {
		forecast.AverageMonthlyExpenseCents = int64(math.Round(float64(totalExpense) / float64(expensePoints)))
	} else {
		forecast.Notes = append(forecast.Notes, "No matching budget expense periods were found; expense projection uses R0.")
	}

	monthlyNet := forecast.AverageMonthlyIncomeCents - forecast.AverageMonthlyExpenseCents
	forecast.ProjectedReserveBalanceCents = input.CurrentReserveBalanceCents + monthlyNet*int64(months)
	if input.ReserveTargetCents > forecast.ProjectedReserveBalanceCents {
		forecast.ProjectedShortfallCents = input.ReserveTargetCents - forecast.ProjectedReserveBalanceCents
	}

	switch {
	case len(forecast.DataPoints) < 2 || expensePoints == 0:
		forecast.Status = "insufficient_data"
	case forecast.ProjectedShortfallCents > 0:
		forecast.Status = "shortfall_risk"
	case monthlyNet < 0:
		forecast.Status = "watch"
	default:
		forecast.Status = "healthy"
	}

	switch {
	case len(forecast.DataPoints) >= 6 && expensePoints >= 3:
		forecast.Confidence = "high"
	case len(forecast.DataPoints) >= 3 && expensePoints >= 1:
		forecast.Confidence = "medium"
	default:
		forecast.Confidence = "low"
	}

	if len(forecast.DataPoints) < 2 {
		forecast.Notes = append(forecast.Notes, "Projection uses fewer than 2 historical levy periods.")
	}
	if input.ReserveTargetCents == 0 {
		forecast.Notes = append(forecast.Notes, "No reserve target is set; shortfall is calculated against R0.")
	}
	if forecast.ProjectedShortfallCents > 0 && input.LatestUnitCount > 0 {
		rawIncrease := int64(math.Ceil(float64(forecast.ProjectedShortfallCents) / float64(months*input.LatestUnitCount)))
		forecast.RecommendedMonthlyIncreaseCents = roundUpToNearest(rawIncrease, 1000)
		if input.CurrentMonthlyLevyCents > 0 {
			forecast.RecommendedIncreasePct = int(math.Round(float64(forecast.RecommendedMonthlyIncreaseCents) * 100 / float64(input.CurrentMonthlyLevyCents)))
		}
	} else if forecast.ProjectedShortfallCents > 0 {
		forecast.Notes = append(forecast.Notes, "No levy accounts were found in the latest period, so no per-unit increase is recommended.")
	}

	return forecast
}

func roundUpToNearest(value, nearest int64) int64 {
	if value <= 0 || nearest <= 0 {
		return 0
	}
	return ((value + nearest - 1) / nearest) * nearest
}
