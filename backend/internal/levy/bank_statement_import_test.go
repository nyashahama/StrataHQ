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

func TestParseCurrencyToCentsHandlesFloatingPointEdgeCases(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"128.14", 12814},
		{"128.17", 12817},
		{"128.20", 12820},
		{"128.23", 12823},
		{"128.39", 12839},
		{"0.29", 29},
		{"0.05", 5},
		{"2450.00", 245000},
		{"1", 100},
		{"-128.20", -12820},
		{"+128.20", 12820},
		{"0.5", 50},
		{"0", 0},
		{"0.00", 0},
	}
	for _, c := range cases {
		got, err := parseCurrencyToCents(c.in)
		if err != nil {
			t.Errorf("parseCurrencyToCents(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseCurrencyToCents(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseCurrencyToCentsRejectsInvalid(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"abc",
		"1.234",
		"1.2.3",
		".",
		"1,2,3.45",
		"1.0e2",
		"-",
	}
	for _, in := range cases {
		if _, err := parseCurrencyToCents(in); err == nil {
			t.Errorf("parseCurrencyToCents(%q) expected error, got nil", in)
		}
	}
}

func TestParseFNBStatementCSVRoundsTwoDecimalCents(t *testing.T) {
	csvData := []byte(`Date,Description,Reference,Amount
2026-04-01,EFT UNIT 12A,12A,128.20
2026-04-02,EFT UNIT 12B,12B,0.29
2026-04-03,EFT UNIT 12C,12C,128.17
`)
	rows, err := parseFNBStatementCSV(csvData)
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	want := []int64{12820, 29, 12817}
	if len(rows) != len(want) {
		t.Fatalf("rows = %d, want %d", len(rows), len(want))
	}
	for i, w := range want {
		if rows[i].AmountCents != w {
			t.Errorf("row %d amount = %d, want %d", i, rows[i].AmountCents, w)
		}
	}
}

func TestParseFNBStatementCSVSkipsNonPositiveRows(t *testing.T) {
	csvData := []byte(`Date,Description,Reference,Amount
2026-04-01,EFT UNIT 12A,12A,2450.00
2026-04-02,EFT UNIT 12A,12A,-2450.00
2026-04-03,EFT UNIT 12A,12A,0.00
2026-04-04,EFT UNIT 12A,12A,1200.00
`)
	rows, err := parseFNBStatementCSV(csvData)
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0].RowNumber != 2 || rows[1].RowNumber != 5 {
		t.Fatalf("row numbers = %d,%d, want 2,5", rows[0].RowNumber, rows[1].RowNumber)
	}
}

func TestValidateManualBankMatches(t *testing.T) {
	rowID := func(s string) string { return s }
	accountID := func(s string) string { return s }

	tests := []struct {
		name    string
		matches []BankStatementManualMatchInput
		wantErr bool
	}{
		{
			name:    "empty",
			matches: nil,
			wantErr: true,
		},
		{
			name: "valid",
			matches: []BankStatementManualMatchInput{
				{RowID: rowID("r1"), AccountID: accountID("a1")},
				{RowID: rowID("r2"), AccountID: accountID("a2")},
			},
		},
		{
			name: "duplicate row_id",
			matches: []BankStatementManualMatchInput{
				{RowID: rowID("r1"), AccountID: accountID("a1")},
				{RowID: rowID("r1"), AccountID: accountID("a2")},
			},
			wantErr: true,
		},
		{
			name: "blank row id",
			matches: []BankStatementManualMatchInput{
				{RowID: "", AccountID: accountID("a1")},
			},
			wantErr: true,
		},
		{
			name: "blank account id",
			matches: []BankStatementManualMatchInput{
				{RowID: rowID("r1"), AccountID: ""},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateManualBankMatches(tt.matches)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseFNBStatementCSVSupportsDDMMYYYY(t *testing.T) {
	csvData := []byte(`Date,Description,Reference,Amount
01/04/2026,EFT UNIT 12A,12A,2450.00
`)
	rows, err := parseFNBStatementCSV(csvData)
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].TransactionDate != dateTime(2026, time.April, 1) {
		t.Fatalf("date = %s, want 2026-04-01", rows[0].TransactionDate.Format("2006-01-02"))
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

func TestMatchBankStatementRowDoesNotMatchWhenReferenceIsBlank(t *testing.T) {
	account := candidateLevyAccount{
		LevyAccountID:    uuid.New(),
		UnitIdentifier:   "12A",
		OwnerName:        "No Match",
		OutstandingCents: 50000,
	}
	row := ParsedBankStatementRow{
		AmountCents: 50000,
		Reference:   "   ",
		Description: "General ledger credit",
	}
	got := matchBankStatementRow(row, []candidateLevyAccount{account})
	if got.Status != "unmatched" {
		t.Fatalf("status = %q, want unmatched", got.Status)
	}
	if got.MatchedLevyAccountID != nil {
		t.Fatal("blank reference should not create an initial unit match")
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
