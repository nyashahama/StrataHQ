package levy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/jobs"
)

var (
	ErrDuplicateImport = fmt.Errorf("duplicate import")
	ErrImportNotFound  = fmt.Errorf("import not found")
)

type BankStatementSource string

const BankStatementSourceFNB BankStatementSource = "fnb"

type ParsedBankStatementRow struct {
	RowNumber       int
	TransactionDate time.Time
	AmountCents     int64
	Reference       string
	Description     string
	NormalizedRef   string
	RowFingerprint  string
	RawData         map[string]string
}

type candidateLevyAccount struct {
	LevyAccountID    uuid.UUID
	UnitIdentifier   string
	OwnerName        string
	DueCents         int64
	PaidCents        int64
	OutstandingCents int64
}

type matchedBankStatementRow struct {
	Status               dbgen.BankStatementRowStatus
	MatchedLevyAccountID *uuid.UUID
	Confidence           int32
	Reason               string
}

type BankStatementImportInput struct {
	BankName         string
	OriginalFilename string
	RawCSV           []byte
}

type BankStatementManualMatchInput struct {
	RowID       string  `json:"row_id"`
	AccountID   string  `json:"account_id"`
	PaymentDate string  `json:"payment_date"`
	AmountCents int64   `json:"amount_cents"`
	Reference   string  `json:"reference"`
	BankRef     *string `json:"bank_ref"`
}

type BankStatementImportResponse struct {
	ID            string `json:"id"`
	SchemeID      string `json:"scheme_id"`
	BankName      string `json:"bank_name"`
	Status        string `json:"status"`
	TotalRows     int64  `json:"total_rows"`
	MatchedRows   int64  `json:"matched_rows"`
	AmbiguousRows int64  `json:"ambiguous_rows"`
	UnmatchedRows int64  `json:"unmatched_rows"`
	AppliedRows   int64  `json:"applied_rows"`
}

type BankStatementImportDetails struct {
	BankStatementImportResponse
	Rows []BankStatementImportRow `json:"rows"`
}

type BankStatementImportRow struct {
	ID                   string  `json:"id"`
	RowNumber            int     `json:"row_number"`
	TransactionDate      string  `json:"transaction_date"`
	AmountCents          int64   `json:"amount_cents"`
	Reference            string  `json:"reference"`
	Description          string  `json:"description"`
	Status               string  `json:"status"`
	Confidence           int     `json:"confidence"`
	MatchReason          *string `json:"match_reason,omitempty"`
	MatchedLevyAccountID *string `json:"matched_levy_account_id,omitempty"`
	UnitIdentifier       *string `json:"unit_identifier,omitempty"`
}

var fillerWords = []string{"EFT", "PAYMENT", "REF", "REFERENCE"}

func normalizeBankStatementReference(value string) string {
	upper := strings.ToUpper(value)
	cleaned := strings.Map(func(r rune) rune {
		if r == '/' || r == '-' || r == '\\' || r == '_' || r == ' ' || r == '.' || r == ',' || r == '#' || r == ':' || r == '(' || r == ')' {
			return -1
		}
		return r
	}, upper)
	for _, word := range fillerWords {
		cleaned = strings.ReplaceAll(cleaned, word, "")
	}
	return strings.TrimSpace(cleaned)
}

func parseFNBStatementCSV(data []byte) ([]ParsedBankStatementRow, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}

	if len(records) < 2 {
		return []ParsedBankStatementRow{}, nil
	}

	header := records[0]
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	amountIdx, hasAmount := colIndex["amount"]
	dateIdx, hasDate := colIndex["date"]
	descIdx, hasDesc := colIndex["description"]
	refIdx, hasRef := colIndex["reference"]

	if !hasAmount || !hasDate {
		return nil, fmt.Errorf("csv must contain Date and Amount columns")
	}

	rows := make([]ParsedBankStatementRow, 0, len(records)-1)
	for rowNum, record := range records[1:] {
		row := ParsedBankStatementRow{
			RowNumber: rowNum + 2,
			RawData:   make(map[string]string),
		}

		for i, col := range header {
			if i < len(record) {
				row.RawData[strings.TrimSpace(col)] = record[i]
			}
		}

		dateStr := strings.TrimSpace(record[dateIdx])
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			row.TransactionDate = t
		} else if t, err := time.Parse("02/01/2006", dateStr); err == nil {
			row.TransactionDate = t
		} else {
			return nil, fmt.Errorf("row %d: cannot parse date %q", row.RowNumber, dateStr)
		}

		amountStr := strings.TrimSpace(record[amountIdx])
		cleanAmount := strings.NewReplacer(",", "", "R", "").Replace(amountStr)
		floatVal, err := strconv.ParseFloat(cleanAmount, 64)
		if err != nil {
			return nil, fmt.Errorf("row %d: cannot parse amount %q", row.RowNumber, amountStr)
		}
		if floatVal <= 0 {
			continue
		}
		cents := int64(floatVal * 100)
		row.AmountCents = cents

		if hasRef && refIdx < len(record) {
			row.Reference = strings.TrimSpace(record[refIdx])
		}

		if hasDesc && descIdx < len(record) {
			row.Description = strings.TrimSpace(record[descIdx])
		}

		row.NormalizedRef = normalizeBankStatementReference(row.Reference)
		row.RowFingerprint = fingerprintBankStatementRow("fnb", row)

		rows = append(rows, row)
	}

	return rows, nil
}

func fingerprintBankStatementRow(bankName string, row ParsedBankStatementRow) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%d|%s", bankName, row.TransactionDate.Format("2006-01-02"), row.AmountCents, row.NormalizedRef)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func marshalBankStatementRawData(rawData map[string]string) ([]byte, error) {
	rawDataJSON, err := json.Marshal(rawData)
	if err != nil {
		return nil, fmt.Errorf("marshal bank statement raw data: %w", err)
	}
	return rawDataJSON, nil
}

func matchBankStatementRow(row ParsedBankStatementRow, accounts []candidateLevyAccount) matchedBankStatementRow {
	normalRef := row.NormalizedRef
	if normalRef == "" {
		normalRef = normalizeBankStatementReference(row.Reference)
	}
	normalDesc := normalizeBankStatementReference(row.Description)
	var candidates []candidateLevyAccount
	for _, acct := range accounts {
		unitUpper := strings.ToUpper(acct.UnitIdentifier)
		normalUnit := normalizeBankStatementReference(unitUpper)
		if normalUnit == "" {
			continue
		}
		refContainsUnit := strings.Contains(normalRef, normalUnit)
		unitContainsRef := normalRef != "" && strings.Contains(normalUnit, normalRef)
		descContainsUnit := strings.Contains(normalDesc, normalUnit)
		if refContainsUnit || descContainsUnit || unitContainsRef || strings.Contains(normalRef, unitUpper) || strings.Contains(normalDesc, unitUpper) {
			candidates = append(candidates, acct)
		}
	}

	if len(candidates) == 1 {
		if row.AmountCents > candidates[0].OutstandingCents {
			return matchedBankStatementRow{
				Status:     dbgen.BankStatementRowStatusAmbiguous,
				Confidence: 50,
				Reason:     fmt.Sprintf("unit %s matched but amount %d exceeds outstanding %d", candidates[0].UnitIdentifier, row.AmountCents, candidates[0].OutstandingCents),
			}
		}
		return matchedBankStatementRow{
			Status:               dbgen.BankStatementRowStatusMatched,
			MatchedLevyAccountID: &candidates[0].LevyAccountID,
			Confidence:           90,
			Reason:               fmt.Sprintf("unit identifier %s matched", candidates[0].UnitIdentifier),
		}
	}

	if len(candidates) > 1 {
		exactAmountCands := []candidateLevyAccount{}
		for _, c := range candidates {
			if c.OutstandingCents == row.AmountCents {
				exactAmountCands = append(exactAmountCands, c)
			}
		}
		if len(exactAmountCands) == 1 {
			return matchedBankStatementRow{
				Status:               dbgen.BankStatementRowStatusMatched,
				MatchedLevyAccountID: &exactAmountCands[0].LevyAccountID,
				Confidence:           70,
				Reason:               fmt.Sprintf("unit identifier %s matched with exact amount", exactAmountCands[0].UnitIdentifier),
			}
		}
		ids := make([]string, len(candidates))
		for i, c := range candidates {
			ids[i] = c.UnitIdentifier
		}
		return matchedBankStatementRow{
			Status:     dbgen.BankStatementRowStatusAmbiguous,
			Confidence: 50,
			Reason:     fmt.Sprintf("multiple units matched: %s", strings.Join(ids, ", ")),
		}
	}

	ownerCandidates := []candidateLevyAccount{}
	for _, acct := range accounts {
		ownerNorm := normalizeBankStatementReference(acct.OwnerName)
		if ownerNorm == "" {
			continue
		}
		if strings.Contains(normalDesc, ownerNorm) {
			ownerCandidates = append(ownerCandidates, acct)
		}
	}

	if len(ownerCandidates) == 1 && ownerCandidates[0].OutstandingCents == row.AmountCents {
		return matchedBankStatementRow{
			Status:               dbgen.BankStatementRowStatusMatched,
			MatchedLevyAccountID: &ownerCandidates[0].LevyAccountID,
			Confidence:           60,
			Reason:               fmt.Sprintf("owner name %s matched with exact amount", ownerCandidates[0].OwnerName),
		}
	}

	return matchedBankStatementRow{
		Status:     dbgen.BankStatementRowStatusUnmatched,
		Confidence: 0,
		Reason:     "no matching unit or owner found",
	}
}

func (s *Service) ImportBankStatement(ctx context.Context, identity auth.Identity, schemeID string, input BankStatementImportInput) (*BankStatementImportResponse, error) {
	scheme, role, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if !auth.IsAdminRole(role) {
		return nil, ErrForbidden
	}

	userUUID, err := uuid.Parse(identity.UserID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	var status dbgen.BankStatementImportStatus = dbgen.BankStatementImportStatusQueued
	import_, err := s.db.Q.CreateBankStatementImport(ctx, dbgen.CreateBankStatementImportParams{
		SchemeID:         scheme.ID,
		UploadedByUserID: userUUID,
		BankName:         input.BankName,
		OriginalFilename: input.OriginalFilename,
		RawCsv:           input.RawCSV,
		Status:           status,
	})
	if err != nil {
		return nil, err
	}

	if s.jobs != nil {
		payload := struct {
			ImportID string `json:"importId"`
		}{
			ImportID: import_.ID.String(),
		}
		if _, err := s.jobs.Enqueue(ctx, jobs.EnqueueInput{
			Kind:           jobs.KindBankStatementImport,
			Payload:        payload,
			IdempotencyKey: import_.ID.String(),
			MaxAttempts:    3,
		}); err != nil {
			_, _ = s.db.Q.UpdateBankStatementImportStatus(ctx, dbgen.UpdateBankStatementImportStatusParams{
				ID:            import_.ID,
				Status:        dbgen.BankStatementImportStatusFailed,
				TotalRows:     0,
				MatchedRows:   0,
				AmbiguousRows: 0,
				UnmatchedRows: 0,
				AppliedRows:   0,
				ParsedAt:      pgtype.Timestamptz{},
				AppliedAt:     pgtype.Timestamptz{},
				LastError:     pgtype.Text{String: err.Error(), Valid: true},
			})
			return nil, err
		}
	}

	return &BankStatementImportResponse{
		ID:            import_.ID.String(),
		SchemeID:      import_.SchemeID.String(),
		BankName:      import_.BankName,
		Status:        string(import_.Status),
		TotalRows:     int64(import_.TotalRows),
		MatchedRows:   int64(import_.MatchedRows),
		AmbiguousRows: int64(import_.AmbiguousRows),
		UnmatchedRows: int64(import_.UnmatchedRows),
		AppliedRows:   int64(import_.AppliedRows),
	}, nil
}

func (s *Service) GetBankStatementImport(ctx context.Context, identity auth.Identity, schemeID, importID string) (*BankStatementImportDetails, error) {
	scheme, role, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if !auth.IsAdminRole(role) {
		return nil, ErrForbidden
	}

	importUUID, err := uuid.Parse(importID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	import_, err := s.db.Q.GetBankStatementImport(ctx, dbgen.GetBankStatementImportParams{
		ID:       importUUID,
		SchemeID: scheme.ID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	dbRows, err := s.db.Q.ListBankStatementRowsByImport(ctx, importUUID)
	if err != nil {
		return nil, err
	}

	rows := make([]BankStatementImportRow, len(dbRows))
	for i, r := range dbRows {
		row := BankStatementImportRow{
			ID:              r.ID.String(),
			RowNumber:       int(r.RowNumber),
			TransactionDate: formatRowDate(r.TransactionDate),
			AmountCents:     r.AmountCents,
			Reference:       r.Reference,
			Description:     r.Description,
			Status:          string(r.Status),
			Confidence:      int(r.Confidence),
		}
		if r.MatchReason.Valid {
			row.MatchReason = &r.MatchReason.String
		}
		if r.MatchedLevyAccountID.Valid {
			id := r.MatchedLevyAccountID.String()
			row.MatchedLevyAccountID = &id
			laUUID := uuid.UUID(r.MatchedLevyAccountID.Bytes)
			if la, err := s.db.Q.GetLevyAccount(ctx, laUUID); err == nil {
				if unit, unitErr := s.db.Q.GetUnit(ctx, la.UnitID); unitErr == nil {
					row.UnitIdentifier = &unit.Identifier
				}
			}
		}
		rows[i] = row
	}

	return &BankStatementImportDetails{
		BankStatementImportResponse: BankStatementImportResponse{
			ID:            import_.ID.String(),
			SchemeID:      import_.SchemeID.String(),
			BankName:      import_.BankName,
			Status:        string(import_.Status),
			TotalRows:     int64(import_.TotalRows),
			MatchedRows:   int64(import_.MatchedRows),
			AmbiguousRows: int64(import_.AmbiguousRows),
			UnmatchedRows: int64(import_.UnmatchedRows),
			AppliedRows:   int64(import_.AppliedRows),
		},
		Rows: rows,
	}, nil
}

func (s *Service) ApplyBankStatementImport(ctx context.Context, identity auth.Identity, schemeID, importID string, manualMatches []BankStatementManualMatchInput) (*BankStatementImportResponse, error) {
	scheme, role, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if !auth.IsAdminRole(role) {
		return nil, ErrForbidden
	}

	importUUID, err := uuid.Parse(importID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	import_, err := s.db.Q.GetBankStatementImport(ctx, dbgen.GetBankStatementImportParams{
		ID:       importUUID,
		SchemeID: scheme.ID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := s.db.Q.WithTx(tx)

	dbRows, err := q.ListBankStatementRowsByImport(ctx, importUUID)
	if err != nil {
		return nil, err
	}

	rowMap := make(map[string]dbgen.BankStatementRow)
	for _, r := range dbRows {
		rowMap[r.ID.String()] = r
	}

	appliedCount := int32(0)
	for _, match := range manualMatches {
		row, ok := rowMap[match.RowID]
		if !ok {
			return nil, ErrInvalidInput
		}
		if row.Status == dbgen.BankStatementRowStatusApplied || row.Status == dbgen.BankStatementRowStatusSkipped {
			continue
		}

		accountID, parseErr := uuid.Parse(match.AccountID)
		if parseErr != nil {
			return nil, ErrInvalidInput
		}

		account, getErr := q.GetLevyAccount(ctx, accountID)
		if getErr != nil {
			if errors.Is(getErr, pgx.ErrNoRows) {
				return nil, ErrInvalidInput
			}
			return nil, getErr
		}

		period, getErr := q.GetLevyPeriod(ctx, account.PeriodID)
		if getErr != nil {
			if errors.Is(getErr, pgx.ErrNoRows) {
				return nil, ErrInvalidInput
			}
			return nil, getErr
		}
		if period.SchemeID != scheme.ID {
			return nil, ErrForbidden
		}

		if !row.TransactionDate.Valid {
			return nil, ErrInvalidInput
		}
		paymentDate := row.TransactionDate.Time

		var bankRef pgtype.Text
		if match.BankRef != nil && *match.BankRef != "" {
			bankRef = pgtype.Text{String: *match.BankRef, Valid: true}
		} else if match.Reference != "" {
			bankRef = pgtype.Text{String: match.Reference, Valid: true}
		}

		amountCents := row.AmountCents
		payment, created, paymentErr := ensureLevyPayment(ctx, q, account.ID, amountCents, paymentDate, row.RowFingerprint, bankRef)
		if paymentErr != nil {
			return nil, paymentErr
		}
		if created {
			newPaid := account.PaidCents + amountCents
			_, err = q.UpdateLevyAccountPaid(ctx, dbgen.UpdateLevyAccountPaidParams{
				ID:        account.ID,
				PaidCents: newPaid,
				Status:    statusFor(newPaid, account.AmountCents, account.DueDate),
				PaidDate:  dateValue(paymentDate),
			})
			if err != nil {
				return nil, err
			}
		}

		if _, err = q.UpdateBankStatementRowApplied(ctx, dbgen.UpdateBankStatementRowAppliedParams{
			ID:                   row.ID,
			MatchedLevyPaymentID: pgtype.UUID{Bytes: payment.ID, Valid: true},
		}); err != nil {
			return nil, err
		}

		appliedCount++
	}

	newApplied := import_.AppliedRows + appliedCount
	finalStatus := dbgen.BankStatementImportStatusReviewRequired
	if newApplied >= import_.TotalRows && import_.TotalRows > 0 {
		finalStatus = dbgen.BankStatementImportStatusApplied
	}

	now := time.Now().UTC()
	appliedAt := pgtype.Timestamptz{}
	if finalStatus == dbgen.BankStatementImportStatusApplied {
		appliedAt = pgtype.Timestamptz{Time: now, Valid: true}
	}

	_, err = q.UpdateBankStatementImportStatus(ctx, dbgen.UpdateBankStatementImportStatusParams{
		ID:            importUUID,
		Status:        finalStatus,
		TotalRows:     import_.TotalRows,
		MatchedRows:   import_.MatchedRows,
		AmbiguousRows: import_.AmbiguousRows,
		UnmatchedRows: import_.UnmatchedRows,
		AppliedRows:   newApplied,
		ParsedAt:      import_.ParsedAt,
		AppliedAt:     appliedAt,
		LastError:     import_.LastError,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &BankStatementImportResponse{
		ID:            import_.ID.String(),
		SchemeID:      import_.SchemeID.String(),
		BankName:      import_.BankName,
		Status:        string(finalStatus),
		TotalRows:     int64(import_.TotalRows),
		MatchedRows:   int64(import_.MatchedRows),
		AmbiguousRows: int64(import_.AmbiguousRows),
		UnmatchedRows: int64(import_.UnmatchedRows),
		AppliedRows:   int64(newApplied),
	}, nil
}

func (s *Service) ProcessBankStatementImport(ctx context.Context, importID string) error {
	importUUID, err := uuid.Parse(importID)
	if err != nil {
		return fmt.Errorf("invalid import id: %w", err)
	}

	import_, err := s.db.Q.GetBankStatementImportByID(ctx, importUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("%w: %w", jobs.ErrNonRetryable, ErrImportNotFound)
		}
		return err
	}

	if import_.Status == dbgen.BankStatementImportStatusApplied || import_.Status == dbgen.BankStatementImportStatusReviewRequired {
		rows, rowsErr := s.db.Q.ListBankStatementRowsByImport(ctx, importUUID)
		if rowsErr == nil && len(rows) > 0 {
			return nil
		}
	}
	if import_.Status == dbgen.BankStatementImportStatusFailed {
		return fmt.Errorf("%w: import already failed", jobs.ErrNonRetryable)
	}

	parsedRows, err := parseFNBStatementCSV(import_.RawCsv)
	if err != nil {
		_, _ = s.db.Q.UpdateBankStatementImportStatus(ctx, dbgen.UpdateBankStatementImportStatusParams{
			ID:            importUUID,
			Status:        dbgen.BankStatementImportStatusFailed,
			TotalRows:     0,
			MatchedRows:   0,
			AmbiguousRows: 0,
			UnmatchedRows: 0,
			AppliedRows:   0,
			ParsedAt:      pgtype.Timestamptz{},
			AppliedAt:     pgtype.Timestamptz{},
			LastError:     pgtype.Text{String: err.Error(), Valid: true},
		})
		return fmt.Errorf("%w: %v", jobs.ErrNonRetryable, err)
	}

	openAccounts, err := s.db.Q.ListOpenLevyAccountsByScheme(ctx, import_.SchemeID)
	if err != nil {
		return err
	}

	accounts := make([]candidateLevyAccount, len(openAccounts))
	for i, a := range openAccounts {
		outstanding := a.AmountCents - a.PaidCents
		if outstanding < 0 {
			outstanding = 0
		}
		accounts[i] = candidateLevyAccount{
			LevyAccountID:    a.ID,
			UnitIdentifier:   a.UnitIdentifier,
			OwnerName:        a.OwnerName,
			DueCents:         a.AmountCents,
			PaidCents:        a.PaidCents,
			OutstandingCents: outstanding,
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := s.db.Q.WithTx(tx)

	var matchedCount, ambiguousCount, unmatchedCount int32

	for _, parsed := range parsedRows {
		rawDataJSON, marshalErr := marshalBankStatementRawData(parsed.RawData)
		if marshalErr != nil {
			return fmt.Errorf("%w: %v", jobs.ErrNonRetryable, marshalErr)
		}
		match := matchBankStatementRow(parsed, accounts)

		var matchedAccountID pgtype.UUID
		if match.MatchedLevyAccountID != nil {
			matchedAccountID = pgtype.UUID{Bytes: *match.MatchedLevyAccountID, Valid: true}
		}

		var matchReason pgtype.Text
		if match.Reason != "" {
			matchReason = pgtype.Text{String: match.Reason, Valid: true}
		}

		row, createErr := q.CreateBankStatementRow(ctx, dbgen.CreateBankStatementRowParams{
			ImportID:             importUUID,
			RowNumber:            int32(parsed.RowNumber),
			TransactionDate:      dateValue(parsed.TransactionDate),
			AmountCents:          parsed.AmountCents,
			Reference:            parsed.Reference,
			Description:          parsed.Description,
			NormalizedReference:  parsed.NormalizedRef,
			RowFingerprint:       parsed.RowFingerprint,
			Status:               match.Status,
			Confidence:           match.Confidence,
			MatchReason:          matchReason,
			MatchedLevyAccountID: matchedAccountID,
			MatchedLevyPaymentID: pgtype.UUID{},
			RawData:              rawDataJSON,
		})
		if createErr != nil {
			return createErr
		}

		switch match.Status {
		case dbgen.BankStatementRowStatusMatched:
			matchedCount++
			if match.MatchedLevyAccountID != nil {
				acct, acctErr := q.GetLevyAccount(ctx, *match.MatchedLevyAccountID)
				if acctErr != nil {
					return acctErr
				}

				payment, created, payErr := ensureLevyPayment(ctx, q, acct.ID, parsed.AmountCents, parsed.TransactionDate, parsed.RowFingerprint, pgtype.Text{String: parsed.Reference, Valid: true})
				if payErr != nil {
					return payErr
				}
				if created {
					newPaid := acct.PaidCents + parsed.AmountCents
					if _, updErr := q.UpdateLevyAccountPaid(ctx, dbgen.UpdateLevyAccountPaidParams{
						ID:        acct.ID,
						PaidCents: newPaid,
						Status:    statusFor(newPaid, acct.AmountCents, acct.DueDate),
						PaidDate:  dateValue(parsed.TransactionDate),
					}); updErr != nil {
						return updErr
					}
					for ai := range accounts {
						if accounts[ai].LevyAccountID == acct.ID {
							accounts[ai].PaidCents = newPaid
							remaining := acct.AmountCents - newPaid
							if remaining < 0 {
								remaining = 0
							}
							accounts[ai].OutstandingCents = remaining
							break
						}
					}
				}
				if _, updErr := q.UpdateBankStatementRowApplied(ctx, dbgen.UpdateBankStatementRowAppliedParams{
					ID:                   row.ID,
					MatchedLevyPaymentID: pgtype.UUID{Bytes: payment.ID, Valid: true},
				}); updErr != nil {
					return updErr
				}
			}
		case dbgen.BankStatementRowStatusAmbiguous:
			ambiguousCount++
		case dbgen.BankStatementRowStatusUnmatched:
			unmatchedCount++
		}
	}

	now := time.Now().UTC()
	parsedAt := pgtype.Timestamptz{Time: now, Valid: true}

	status := dbgen.BankStatementImportStatusReviewRequired
	if ambiguousCount == 0 && unmatchedCount == 0 {
		status = dbgen.BankStatementImportStatusApplied
	}
	appliedAt := pgtype.Timestamptz{}
	if status == dbgen.BankStatementImportStatusApplied {
		appliedAt = parsedAt
	}

	_, err = q.UpdateBankStatementImportStatus(ctx, dbgen.UpdateBankStatementImportStatusParams{
		ID:            importUUID,
		Status:        status,
		TotalRows:     int32(len(parsedRows)),
		MatchedRows:   matchedCount,
		AmbiguousRows: ambiguousCount,
		UnmatchedRows: unmatchedCount,
		AppliedRows:   matchedCount,
		ParsedAt:      parsedAt,
		AppliedAt:     appliedAt,
		LastError:     pgtype.Text{},
	})
	if err != nil {
		return err
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return commitErr
	}

	if clearErr := s.db.Q.ClearBankStatementImportRawCsv(ctx, importUUID); clearErr != nil {
		return fmt.Errorf("clear raw csv: %w", clearErr)
	}

	return nil
}

func formatRowDate(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}

func ensureLevyPayment(ctx context.Context, q *dbgen.Queries, accountID uuid.UUID, amountCents int64, paymentDate time.Time, reference string, bankRef pgtype.Text) (dbgen.LevyPayment, bool, error) {
	if existing, err := q.GetLevyPaymentByReference(ctx, reference); err == nil {
		if existing.LevyAccountID == accountID {
			return existing, false, nil
		}
		reference = reference + ":" + accountID.String()[:8]
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return dbgen.LevyPayment{}, false, err
	}

	payment, err := q.CreateLevyPayment(ctx, dbgen.CreateLevyPaymentParams{
		LevyAccountID: accountID,
		AmountCents:   amountCents,
		PaymentDate:   dateValue(paymentDate),
		Reference:     reference,
		BankRef:       bankRef,
	})
	if err != nil {
		if isUniqueViolation(err) {
			existing, getErr := q.GetLevyPaymentByReference(ctx, reference)
			if getErr != nil {
				return dbgen.LevyPayment{}, false, getErr
			}
			if existing.LevyAccountID == accountID {
				return existing, false, nil
			}
			return dbgen.LevyPayment{}, false, fmt.Errorf("reference %s already used by another levy account", reference)
		}
		return dbgen.LevyPayment{}, false, err
	}

	return payment, true, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
