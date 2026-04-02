package levy

import (
	"context"
	"errors"
	"math"
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

//nolint:govet // Keep API response fields grouped by meaning rather than field packing.
type PeriodInfo struct {
	ID          string    `json:"id"`
	SchemeID    string    `json:"scheme_id"`
	Label       string    `json:"label"`
	DueDate     string    `json:"due_date"`
	AmountCents int64     `json:"amount_cents"`
	CreatedAt   time.Time `json:"created_at"`
}

//nolint:govet // Keep API response fields grouped by meaning rather than field packing.
type AccountInfo struct {
	PaidDate       *string `json:"paid_date"`
	ID             string  `json:"id"`
	UnitID         string  `json:"unit_id"`
	UnitIdentifier string  `json:"unit_identifier"`
	OwnerName      string  `json:"owner_name"`
	PeriodID       string  `json:"period_id"`
	AmountCents    int64   `json:"amount_cents"`
	PaidCents      int64   `json:"paid_cents"`
	Status         string  `json:"status"`
	DueDate        string  `json:"due_date"`
}

//nolint:govet // Keep API response fields grouped by meaning rather than field packing.
type PaymentInfo struct {
	BankRef       *string   `json:"bank_ref"`
	ID            string    `json:"id"`
	LevyAccountID string    `json:"levy_account_id"`
	AmountCents   int64     `json:"amount_cents"`
	PaymentDate   string    `json:"payment_date"`
	Reference     string    `json:"reference"`
	CreatedAt     time.Time `json:"created_at"`
}

type CollectionTrendPoint struct {
	Label string `json:"label"`
	Pct   int    `json:"pct"`
}

//nolint:govet // Keep API response fields grouped by meaning rather than field packing.
type DashboardResponse struct {
	CurrentPeriod       *PeriodInfo            `json:"current_period"`
	MyAccount           *AccountInfo           `json:"my_account"`
	CollectionTrend     []CollectionTrendPoint `json:"collection_trend"`
	LevyRoll            []AccountInfo          `json:"levy_roll"`
	MyPayments          []PaymentInfo          `json:"my_payments"`
	Role                string                 `json:"role"`
	CollectionRatePct   int                    `json:"collection_rate_pct"`
	OverdueCount        int                    `json:"overdue_count"`
	TotalCollectedCents int64                  `json:"total_collected_cents"`
}

type CreatePeriodInput struct {
	Label       string
	DueDate     time.Time
	AmountCents int64
}

type ReconcilePaymentInput struct {
	AccountID   string
	PaymentDate time.Time
	Reference   string
	BankRef     *string
	AmountCents int64
}

type ReconcileResult struct {
	UpdatedAccountIDs []string `json:"updated_account_ids"`
	AppliedCount      int      `json:"applied_count"`
	SkippedCount      int      `json:"skipped_count"`
}

type Service struct {
	db *database.Pool
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) Dashboard(ctx context.Context, identity auth.Identity, schemeID string) (*DashboardResponse, error) {
	scheme, role, memberUnitID, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	resp := &DashboardResponse{
		Role:            role,
		CollectionTrend: []CollectionTrendPoint{},
		LevyRoll:        []AccountInfo{},
		MyPayments:      []PaymentInfo{},
	}

	periods, err := s.db.Q.ListLevyPeriodsByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}
	if len(periods) == 0 {
		return resp, nil
	}

	currentPeriod := periods[0]
	resp.CurrentPeriod = pointerToPeriod(mapPeriod(currentPeriod))

	accounts, err := s.db.Q.ListLevyAccountsByPeriod(ctx, currentPeriod.ID)
	if err != nil {
		return nil, err
	}

	for _, account := range accounts {
		mapped := mapAccountRow(account)
		resp.LevyRoll = append(resp.LevyRoll, mapped)
		resp.TotalCollectedCents += minInt64(mapped.PaidCents, mapped.AmountCents)
		if mapped.Status == "overdue" {
			resp.OverdueCount++
		}
	}
	resp.CollectionRatePct = collectionPct(resp.LevyRoll)
	resp.CollectionTrend, err = s.buildTrend(ctx, periods)
	if err != nil {
		return nil, err
	}

	if memberUnitID != nil {
		account, accountErr := s.db.Q.GetLevyAccountByUnitAndPeriod(ctx, dbgen.GetLevyAccountByUnitAndPeriodParams{
			UnitID:   *memberUnitID,
			PeriodID: currentPeriod.ID,
		})
		if accountErr == nil {
			unit, unitErr := s.db.Q.GetUnit(ctx, account.UnitID)
			if unitErr != nil {
				return nil, unitErr
			}
			resp.MyAccount = &AccountInfo{
				ID:             account.ID.String(),
				UnitID:         account.UnitID.String(),
				UnitIdentifier: unit.Identifier,
				OwnerName:      unit.OwnerName,
				PeriodID:       account.PeriodID.String(),
				AmountCents:    account.AmountCents,
				PaidCents:      account.PaidCents,
				Status:         statusFor(account.PaidCents, account.AmountCents, account.DueDate),
				DueDate:        formatDate(account.DueDate),
				PaidDate:       optionalDate(account.PaidDate),
			}
		} else if !errors.Is(accountErr, pgx.ErrNoRows) {
			return nil, accountErr
		}

		payments, paymentsErr := s.db.Q.ListLevyPaymentsByUnit(ctx, *memberUnitID)
		if paymentsErr != nil {
			return nil, paymentsErr
		}
		resp.MyPayments = make([]PaymentInfo, 0, len(payments))
		for _, payment := range payments {
			resp.MyPayments = append(resp.MyPayments, mapPayment(payment))
		}
	}

	if role == string(auth.RoleResident) {
		resp.LevyRoll = []AccountInfo{}
		resp.CollectionTrend = []CollectionTrendPoint{}
		resp.TotalCollectedCents = 0
		resp.OverdueCount = 0
		resp.CollectionRatePct = 0
	}

	return resp, nil
}

func (s *Service) CreatePeriod(ctx context.Context, identity auth.Identity, schemeID string, input CreatePeriodInput) (*PeriodInfo, error) {
	scheme, role, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if !auth.IsAdminRole(role) {
		return nil, ErrForbidden
	}
	if input.Label == "" || input.AmountCents <= 0 || input.DueDate.IsZero() {
		return nil, ErrInvalidInput
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := s.db.Q.WithTx(tx)
	period, err := q.CreateLevyPeriod(ctx, dbgen.CreateLevyPeriodParams{
		SchemeID:    scheme.ID,
		Label:       input.Label,
		AmountCents: input.AmountCents,
		DueDate:     dateValue(input.DueDate),
	})
	if err != nil {
		return nil, err
	}

	units, err := q.ListUnitsByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}
	for _, unit := range units {
		if _, err := q.CreateLevyAccount(ctx, dbgen.CreateLevyAccountParams{
			UnitID:      unit.ID,
			PeriodID:    period.ID,
			AmountCents: input.AmountCents,
			DueDate:     dateValue(input.DueDate),
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	created := mapPeriod(period)
	return &created, nil
}

func (s *Service) Reconcile(ctx context.Context, identity auth.Identity, schemeID string, payments []ReconcilePaymentInput) (*ReconcileResult, error) {
	scheme, role, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if !auth.IsAdminRole(role) {
		return nil, ErrForbidden
	}
	if len(payments) == 0 {
		return nil, ErrInvalidInput
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := s.db.Q.WithTx(tx)
	result := &ReconcileResult{
		UpdatedAccountIDs: []string{},
	}
	seenAccounts := make(map[string]struct{})

	for _, payment := range payments {
		if payment.AccountID == "" || payment.Reference == "" || payment.AmountCents <= 0 || payment.PaymentDate.IsZero() {
			return nil, ErrInvalidInput
		}

		if _, err := q.GetLevyPaymentByReference(ctx, payment.Reference); err == nil {
			result.SkippedCount++
			continue
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}

		accountID, err := uuid.Parse(payment.AccountID)
		if err != nil {
			return nil, ErrInvalidInput
		}

		account, err := q.GetLevyAccount(ctx, accountID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, err
		}

		period, err := q.GetLevyPeriod(ctx, account.PeriodID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		if period.SchemeID != scheme.ID {
			return nil, ErrForbidden
		}

		bankRef := pgtype.Text{}
		if payment.BankRef != nil && *payment.BankRef != "" {
			bankRef = pgtype.Text{String: *payment.BankRef, Valid: true}
		}

		if _, err := q.CreateLevyPayment(ctx, dbgen.CreateLevyPaymentParams{
			LevyAccountID: account.ID,
			AmountCents:   payment.AmountCents,
			PaymentDate:   dateValue(payment.PaymentDate),
			Reference:     payment.Reference,
			BankRef:       bankRef,
		}); err != nil {
			return nil, err
		}

		newPaid := account.PaidCents + payment.AmountCents
		if _, err := q.UpdateLevyAccountPaid(ctx, dbgen.UpdateLevyAccountPaidParams{
			ID:        account.ID,
			PaidCents: newPaid,
			Status:    statusFor(newPaid, account.AmountCents, account.DueDate),
			PaidDate:  dateValue(payment.PaymentDate),
		}); err != nil {
			return nil, err
		}

		result.AppliedCount++
		if _, ok := seenAccounts[account.ID.String()]; !ok {
			seenAccounts[account.ID.String()] = struct{}{}
			result.UpdatedAccountIDs = append(result.UpdatedAccountIDs, account.ID.String())
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) resolveSchemeAccess(ctx context.Context, identity auth.Identity, schemeID string) (dbgen.Scheme, string, *uuid.UUID, error) {
	id, err := uuid.Parse(schemeID)
	if err != nil {
		return dbgen.Scheme{}, "", nil, ErrInvalidInput
	}

	scheme, err := s.db.Q.GetScheme(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dbgen.Scheme{}, "", nil, ErrNotFound
		}
		return dbgen.Scheme{}, "", nil, err
	}

	if auth.IsAdminRole(identity.Role) {
		orgID, parseErr := uuid.Parse(identity.OrgID)
		if parseErr != nil {
			return dbgen.Scheme{}, "", nil, ErrInvalidInput
		}
		if scheme.OrgID != orgID {
			return dbgen.Scheme{}, "", nil, ErrForbidden
		}
		return scheme, string(auth.RoleAdmin), nil, nil
	}

	userID, parseErr := uuid.Parse(identity.UserID)
	if parseErr != nil {
		return dbgen.Scheme{}, "", nil, ErrInvalidInput
	}

	membership, err := s.db.Q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{
		UserID:   userID,
		SchemeID: id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dbgen.Scheme{}, "", nil, ErrForbidden
		}
		return dbgen.Scheme{}, "", nil, err
	}

	var unitID *uuid.UUID
	if membership.UnitID.Valid {
		value := uuid.UUID(membership.UnitID.Bytes)
		unitID = &value
	}

	return scheme, membership.Role, unitID, nil
}

func (s *Service) buildTrend(ctx context.Context, periods []dbgen.LevyPeriod) ([]CollectionTrendPoint, error) {
	if len(periods) == 0 {
		return []CollectionTrendPoint{}, nil
	}

	limit := min(6, len(periods))
	points := make([]CollectionTrendPoint, 0, limit)
	for i := limit - 1; i >= 0; i-- {
		period := periods[i]
		accounts, err := s.db.Q.ListLevyAccountsByPeriod(ctx, period.ID)
		if err != nil {
			return nil, err
		}
		points = append(points, CollectionTrendPoint{
			Label: trendLabel(period.Label, period.DueDate),
			Pct:   collectionPctFromRows(accounts),
		})
	}
	return points, nil
}

func mapPeriod(period dbgen.LevyPeriod) PeriodInfo {
	return PeriodInfo{
		ID:          period.ID.String(),
		SchemeID:    period.SchemeID.String(),
		Label:       period.Label,
		DueDate:     formatDate(period.DueDate),
		AmountCents: period.AmountCents,
		CreatedAt:   period.CreatedAt,
	}
}

func mapAccountRow(row dbgen.ListLevyAccountsByPeriodRow) AccountInfo {
	return AccountInfo{
		PaidDate:       optionalDate(row.PaidDate),
		ID:             row.ID.String(),
		UnitID:         row.UnitID.String(),
		UnitIdentifier: row.UnitIdentifier,
		OwnerName:      row.OwnerName,
		PeriodID:       row.PeriodID.String(),
		AmountCents:    row.AmountCents,
		PaidCents:      row.PaidCents,
		Status:         statusFor(row.PaidCents, row.AmountCents, row.DueDate),
		DueDate:        formatDate(row.DueDate),
	}
}

func mapPayment(payment dbgen.LevyPayment) PaymentInfo {
	return PaymentInfo{
		BankRef:       textPointer(payment.BankRef),
		ID:            payment.ID.String(),
		LevyAccountID: payment.LevyAccountID.String(),
		AmountCents:   payment.AmountCents,
		PaymentDate:   formatDate(payment.PaymentDate),
		Reference:     payment.Reference,
		CreatedAt:     payment.CreatedAt,
	}
}

func pointerToPeriod(period PeriodInfo) *PeriodInfo {
	return &period
}

func formatDate(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}

func optionalDate(value pgtype.Date) *string {
	if !value.Valid {
		return nil
	}
	formatted := value.Time.Format("2006-01-02")
	return &formatted
}

func dateValue(value time.Time) pgtype.Date {
	return pgtype.Date{Time: value, Valid: true}
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}

func statusFor(paidCents, amountCents int64, dueDate pgtype.Date) string {
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

func collectionPct(accounts []AccountInfo) int {
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

func trendLabel(label string, dueDate pgtype.Date) string {
	if label != "" {
		parsed, err := time.Parse("January 2006", label)
		if err == nil {
			return parsed.Format("Jan")
		}
	}
	if dueDate.Valid {
		return dueDate.Time.Format("Jan")
	}
	return label
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}
