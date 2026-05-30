package levy

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func dateTime(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestNormalizeBankStatementReference(t *testing.T) {
	got := normalizeBankStatementReference("  EFT - Unit 12 / Ref 4501 ")
	if got != "UNIT124501" {
		t.Fatalf("got %q", got)
	}
}

func TestParseFNBStatementCSV(t *testing.T) {
	fixture, err := os.ReadFile("testdata/fnb_statement.csv")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	rows, err := parseFNBStatementCSV(fixture)
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	if rows[0].AmountCents != 245000 {
		t.Fatalf("amount = %d, want 245000", rows[0].AmountCents)
	}
}

func TestMatchBankStatementRowPrefersExactUnitToken(t *testing.T) {
	account := candidateLevyAccount{
		LevyAccountID:    uuid.New(),
		UnitIdentifier:   "12A",
		OwnerName:        "A. Adams",
		OutstandingCents: 245000,
	}
	row := ParsedBankStatementRow{
		AmountCents: 245000,
		Reference:   "EFT 12A",
		Description: "Monthly levy payment",
	}
	got := matchBankStatementRow(row, []candidateLevyAccount{account})
	if got.Status != "matched" {
		t.Fatalf("status = %q, want matched", got.Status)
	}
	if got.MatchedLevyAccountID == nil || *got.MatchedLevyAccountID != account.LevyAccountID {
		t.Fatal("expected exact unit token match")
	}
}

func TestMatchBankStatementRowMarksAmbiguousWhenMultipleUnitsMatch(t *testing.T) {
	first := candidateLevyAccount{
		LevyAccountID:    uuid.New(),
		UnitIdentifier:   "12A",
		OwnerName:        "A. Adams",
		OutstandingCents: 245000,
	}
	second := candidateLevyAccount{
		LevyAccountID:    uuid.New(),
		UnitIdentifier:   "12B",
		OwnerName:        "B. Brown",
		OutstandingCents: 245000,
	}
	row := ParsedBankStatementRow{
		AmountCents: 245000,
		Reference:   "EFT 12",
		Description: "Monthly levy payment",
	}
	got := matchBankStatementRow(row, []candidateLevyAccount{first, second})
	if got.Status != "ambiguous" {
		t.Fatalf("status = %q, want ambiguous", got.Status)
	}
	if got.MatchedLevyAccountID != nil {
		t.Fatal("ambiguous row should not have a matched account")
	}
}

func TestFingerprintIsDeterministic(t *testing.T) {
	row1 := ParsedBankStatementRow{
		TransactionDate: dateTime(2026, 4, 1),
		AmountCents:     245000,
		NormalizedRef:   "12A",
	}
	fp1 := fingerprintBankStatementRow("fnb", row1)
	fp2 := fingerprintBankStatementRow("fnb", row1)
	if fp1 != fp2 {
		t.Fatalf("fingerprint mismatch: %s vs %s", fp1, fp2)
	}
}

func TestMarshalBankStatementRawDataPreservesSourceFields(t *testing.T) {
	rawDataJSON, err := marshalBankStatementRawData(map[string]string{
		"Date":        "2026-04-01",
		"Description": "EFT Unit 1A",
		"Amount":      "2450.00",
	})
	if err != nil {
		t.Fatalf("marshal raw data: %v", err)
	}

	var decoded map[string]string
	if err := json.Unmarshal(rawDataJSON, &decoded); err != nil {
		t.Fatalf("unmarshal raw data: %v", err)
	}
	if decoded["Description"] != "EFT Unit 1A" {
		t.Fatalf("description = %q, want source value", decoded["Description"])
	}
}

func TestNormalizeStripsPunctuationAndFillerWords(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"EFT Unit 12A", "UNIT12A"},
		{"PAYMENT FROM SMITH", "FROMSMITH"},
		{"REF 12345 / PAYMENT", "12345"},
		{"Unit-12A", "UNIT12A"},
		{"Unit.12A", "UNIT12A"},
		{"Unit 12A, levy", "UNIT12ALEVY"},
		{"(Unit 12A)", "UNIT12A"},
	}
	for _, tc := range tests {
		got := normalizeBankStatementReference(tc.input)
		if got != tc.want {
			t.Errorf("normalize(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestMatchUnmatchedWhenNoAccountsMatch(t *testing.T) {
	account := candidateLevyAccount{
		LevyAccountID:    uuid.New(),
		UnitIdentifier:   "99Z",
		OwnerName:        "Z. Zed",
		OutstandingCents: 10000,
	}
	row := ParsedBankStatementRow{
		AmountCents:   245000,
		Reference:     "UNKNOWN ENTITY",
		Description:   "Payment from stranger",
		NormalizedRef: normalizeBankStatementReference("UNKNOWN ENTITY"),
	}
	got := matchBankStatementRow(row, []candidateLevyAccount{account})
	if got.Status != "unmatched" {
		t.Fatalf("status = %q, want unmatched", got.Status)
	}
}

func TestMatchAmbiguousResolvedByAmountWhenSingleExactMatch(t *testing.T) {
	first := candidateLevyAccount{
		LevyAccountID:    uuid.New(),
		UnitIdentifier:   "12A",
		OwnerName:        "A. Adams",
		OutstandingCents: 245000,
	}
	second := candidateLevyAccount{
		LevyAccountID:    uuid.New(),
		UnitIdentifier:   "12B",
		OwnerName:        "B. Brown",
		OutstandingCents: 100000,
	}
	row := ParsedBankStatementRow{
		AmountCents:   245000,
		Reference:     "EFT 12",
		Description:   "Monthly levy payment",
		NormalizedRef: normalizeBankStatementReference("EFT 12"),
	}
	got := matchBankStatementRow(row, []candidateLevyAccount{first, second})
	if got.Status != "matched" {
		t.Fatalf("status = %q, want matched", got.Status)
	}
	if got.MatchedLevyAccountID == nil || *got.MatchedLevyAccountID != first.LevyAccountID {
		t.Fatal("expected amount-disambiguated match")
	}
}

func TestMatchMarksOverpayAsAmbiguous(t *testing.T) {
	account := candidateLevyAccount{
		LevyAccountID:    uuid.New(),
		UnitIdentifier:   "5C",
		OwnerName:        "Over Payer",
		OutstandingCents: 100000,
	}
	row := ParsedBankStatementRow{
		AmountCents: 1000000,
		Reference:   "EFT 5C",
		Description: "Overpaid levy",
	}
	got := matchBankStatementRow(row, []candidateLevyAccount{account})
	if got.Status != "ambiguous" {
		t.Fatalf("status = %q, want ambiguous", got.Status)
	}
}

func TestMatchAllowsPartialPaymentOnSingleCandidate(t *testing.T) {
	account := candidateLevyAccount{
		LevyAccountID:    uuid.New(),
		UnitIdentifier:   "3B",
		OwnerName:        "Partial Payer",
		OutstandingCents: 200000,
	}
	row := ParsedBankStatementRow{
		AmountCents: 50000,
		Reference:   "EFT 3B",
		Description: "Partial levy payment",
	}
	got := matchBankStatementRow(row, []candidateLevyAccount{account})
	if got.Status != "matched" {
		t.Fatalf("status = %q, want matched", got.Status)
	}
}
