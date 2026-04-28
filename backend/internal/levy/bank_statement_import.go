package levy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	BankName        string
	OriginalFilename string
	RawCSV          []byte
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
		cents := int64(floatVal * 100)
		if floatVal < 0 {
			cents = -cents
		}
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

func matchBankStatementRow(row ParsedBankStatementRow, accounts []candidateLevyAccount) matchedBankStatementRow {
	normalRef := row.NormalizedRef
	if normalRef == "" {
		normalRef = normalizeBankStatementReference(row.Reference)
	}
	normalDesc := normalizeBankStatementReference(row.Description)
	var candidates []struct {
		account  candidateLevyAccount
		strength int
	}
	for _, acct := range accounts {
		unitUpper := strings.ToUpper(acct.UnitIdentifier)
		normalUnit := normalizeBankStatementReference(unitUpper)
		if normalUnit == "" {
			continue
		}
		refContainsUnit := strings.Contains(normalRef, normalUnit)
		unitContainsRef := strings.Contains(normalUnit, normalRef)
		descContainsUnit := strings.Contains(normalDesc, normalUnit)
		descContainsUnitUpper := strings.Contains(normalDesc, unitUpper)
		refContainsUnitUpper := strings.Contains(normalRef, unitUpper)

		if refContainsUnit || descContainsUnit || unitContainsRef {
			candidates = append(candidates, struct {
				account  candidateLevyAccount
				strength int
			}{acct, 1})
		} else if refContainsUnitUpper || descContainsUnitUpper {
			candidates = append(candidates, struct {
				account  candidateLevyAccount
				strength int
			}{acct, 2})
		}
	}
	for _, acct := range accounts {
		unitUpper := strings.ToUpper(acct.UnitIdentifier)
		normalUnit := normalizeBankStatementReference(unitUpper)
		if normalUnit == "" {
			continue
		}
		if strings.Contains(row.NormalizedRef, normalUnit) || strings.Contains(normalDesc, normalUnit) {
			candidates = append(candidates, struct {
				account  candidateLevyAccount
				strength int
			}{acct, 1})
			continue
		}
		if strings.Contains(row.NormalizedRef, unitUpper) || strings.Contains(normalDesc, unitUpper) {
			candidates = append(candidates, struct {
				account  candidateLevyAccount
				strength int
			}{acct, 2})
		}
	}

	if len(candidates) == 1 {
		return matchedBankStatementRow{
			Status:               dbgen.BankStatementRowStatusMatched,
			MatchedLevyAccountID: &candidates[0].account.LevyAccountID,
			Confidence:           90,
			Reason:               fmt.Sprintf("unit identifier %s matched", candidates[0].account.UnitIdentifier),
		}
	}

	if len(candidates) > 1 {
		exactAmountCands := []candidateLevyAccount{}
		for _, c := range candidates {
			if c.account.OutstandingCents == row.AmountCents {
				exactAmountCands = append(exactAmountCands, c.account)
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
			ids[i] = c.account.UnitIdentifier
		}
		return matchedBankStatementRow{
			Status:  dbgen.BankStatementRowStatusAmbiguous,
			Confidence: 50,
			Reason:  fmt.Sprintf("multiple units matched: %s", strings.Join(ids, ", ")),
		}
	}

	ownerCandidates := []struct {
		account  candidateLevyAccount
		strength int
	}{}
	for _, acct := range accounts {
		ownerNorm := normalizeBankStatementReference(acct.OwnerName)
		if ownerNorm == "" {
			continue
		}
		if strings.Contains(normalDesc, ownerNorm) {
			ownerCandidates = append(ownerCandidates, struct {
				account  candidateLevyAccount
				strength int
			}{acct, 1})
		}
	}

	if len(ownerCandidates) == 1 && ownerCandidates[0].account.OutstandingCents == row.AmountCents {
		return matchedBankStatementRow{
			Status:               dbgen.BankStatementRowStatusMatched,
			MatchedLevyAccountID: &ownerCandidates[0].account.LevyAccountID,
			Confidence:           60,
			Reason:               fmt.Sprintf("owner name %s matched with exact amount", ownerCandidates[0].account.OwnerName),
		}
	}

	return matchedBankStatementRow{
		Status:  dbgen.BankStatementRowStatusUnmatched,
		Confidence: 0,
		Reason:  "no matching unit or owner found",
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
			if unit, err := s.db.Q.GetUnit(ctx, uuid.UUID(r.MatchedLevyAccountID.Bytes)); err == nil {
				row.UnitIdentifier = &unit.Identifier
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

	dbRows, err := s.db.Q.ListBankStatementRowsByImport(ctx, importUUID)
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

		accountID, err := uuid.Parse(match.AccountID)
		if err != nil {
			return nil, ErrInvalidInput
		}

		account, err := s.db.Q.GetLevyAccount(ctx, accountID)
		if err != nil {
			return nil, ErrInvalidInput
		}

		paymentDate, err := time.Parse("2006-01-02", match.PaymentDate)
		if err != nil {
			return nil, ErrInvalidInput
		}

		var bankRef pgtype.Text
		if match.BankRef != nil && *match.BankRef != "" {
			bankRef = pgtype.Text{String: *match.BankRef, Valid: true}
		}

		payment, err := s.db.Q.CreateLevyPayment(ctx, dbgen.CreateLevyPaymentParams{
			LevyAccountID: account.ID,
			AmountCents:   match.AmountCents,
			PaymentDate:   dateValue(paymentDate),
			Reference:     match.Reference,
			BankRef:       bankRef,
		})
		if err != nil {
			return nil, err
		}

		newPaid := account.PaidCents + match.AmountCents
		if _, err := s.db.Q.UpdateLevyAccountPaid(ctx, dbgen.UpdateLevyAccountPaidParams{
			ID:        account.ID,
			PaidCents: newPaid,
			Status:    statusFor(newPaid, account.AmountCents, account.DueDate),
			PaidDate:  dateValue(paymentDate),
		}); err != nil {
			return nil, err
		}

		paymentUUID := payment.ID
		if _, err := s.db.Q.UpdateBankStatementRowApplied(ctx, dbgen.UpdateBankStatementRowAppliedParams{
			ID:                   row.ID,
			MatchedLevyPaymentID: pgtype.UUID{Bytes: paymentUUID, Valid: true},
		}); err != nil {
			return nil, err
		}

		appliedCount++
	}

	newApplied := import_.AppliedRows + appliedCount
	now := time.Now().UTC()
	appliedAt := pgtype.Timestamptz{Time: now, Valid: true}

	_, err = s.db.Q.UpdateBankStatementImportStatus(ctx, dbgen.UpdateBankStatementImportStatusParams{
		ID:          importUUID,
		Status:      import_.Status,
		TotalRows:   import_.TotalRows,
		MatchedRows: import_.MatchedRows,
		AmbiguousRows: import_.AmbiguousRows,
		UnmatchedRows: import_.UnmatchedRows,
		AppliedRows: newApplied,
		ParsedAt:    import_.ParsedAt,
		AppliedAt:   appliedAt,
		LastError:   import_.LastError,
	})
	if err != nil {
		return nil, err
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
			return ErrImportNotFound
		}
		return err
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

	bankName := strings.ToLower(import_.BankName)
	var parsedRows []ParsedBankStatementRow
	switch BankStatementSource(bankName) {
	case BankStatementSourceFNB:
		parsedRows, err = parseFNBStatementCSV(import_.RawCsv)
	default:
		parsedRows, err = parseFNBStatementCSV(import_.RawCsv)
	}
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
		return err
	}

	var matchedCount, ambiguousCount, unmatchedCount int32
	for _, parsed := range parsedRows {
		rawDataJSON, _ := json.Marshal(parsed.RawData)
		match := matchBankStatementRow(parsed, accounts)

		rowUUID := uuid.New()

		var matchedAccountID pgtype.UUID
		if match.MatchedLevyAccountID != nil {
			matchedAccountID = pgtype.UUID{Bytes: *match.MatchedLevyAccountID, Valid: true}
		}

		var matchReason pgtype.Text
		if match.Reason != "" {
			matchReason = pgtype.Text{String: match.Reason, Valid: true}
		}

		if _, err := s.db.Q.CreateBankStatementRow(ctx, dbgen.CreateBankStatementRowParams{
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
		}); err != nil {
			return err
		}

		switch match.Status {
		case dbgen.BankStatementRowStatusMatched:
			matchedCount++
			if match.MatchedLevyAccountID != nil {
				account, err := s.db.Q.GetLevyAccount(ctx, *match.MatchedLevyAccountID)
				if err == nil {
					newPaid := account.PaidCents + parsed.AmountCents
					payment, err := s.db.Q.CreateLevyPayment(ctx, dbgen.CreateLevyPaymentParams{
						LevyAccountID: account.ID,
						AmountCents:   parsed.AmountCents,
						PaymentDate:   dateValue(parsed.TransactionDate),
						Reference:     parsed.RowFingerprint,
						BankRef:       pgtype.Text{String: parsed.Reference, Valid: true},
					})
					if err == nil {
						paymentUUID := payment.ID
						_, _ = s.db.Q.UpdateBankStatementRowApplied(ctx, dbgen.UpdateBankStatementRowAppliedParams{
							ID:                   rowUUID,
							MatchedLevyPaymentID: pgtype.UUID{Bytes: paymentUUID, Valid: true},
						})
						_, _ = s.db.Q.UpdateLevyAccountPaid(ctx, dbgen.UpdateLevyAccountPaidParams{
							ID:        account.ID,
							PaidCents: newPaid,
							Status:    statusFor(newPaid, account.AmountCents, account.DueDate),
							PaidDate:  dateValue(parsed.TransactionDate),
						})
					}
				}
			}
		case dbgen.BankStatementRowStatusAmbiguous:
			ambiguousCount++
		case dbgen.BankStatementRowStatusUnmatched:
			unmatchedCount++
		}

		_ = rowUUID
	}

	now := time.Now().UTC()
	parsedAt := pgtype.Timestamptz{Time: now, Valid: true}

	status := dbgen.BankStatementImportStatusReviewRequired
	if ambiguousCount == 0 && unmatchedCount == 0 {
		status = dbgen.BankStatementImportStatusApplied
	}

	_, err = s.db.Q.UpdateBankStatementImportStatus(ctx, dbgen.UpdateBankStatementImportStatusParams{
		ID:            importUUID,
		Status:        status,
		TotalRows:     int32(len(parsedRows)),
		MatchedRows:   matchedCount,
		AmbiguousRows: ambiguousCount,
		UnmatchedRows: unmatchedCount,
		AppliedRows:   matchedCount,
		ParsedAt:      parsedAt,
		AppliedAt:     pgtype.Timestamptz{},
		LastError:     pgtype.Text{},
	})
	if err != nil {
		return err
	}

	return nil
}

func formatRowDate(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}
