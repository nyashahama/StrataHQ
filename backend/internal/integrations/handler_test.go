package integrations

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	dbgen "github.com/stratahq/backend/db/gen"
)

func TestOpenAPIFinancialsReturnsReserveFundErrors(t *testing.T) {
	reserveErr := errors.New("reserve lookup failed")
	_, err := openAPIFinancials(context.Background(), fakeOpenAPIFinancialsQuerier{
		reserveErr: reserveErr,
	}, uuid.New(), "")
	if !errors.Is(err, reserveErr) {
		t.Fatalf("expected reserve error, got %v", err)
	}
}

func TestOpenAPIFinancialsSuppressesMissingReserveFund(t *testing.T) {
	result, err := openAPIFinancials(context.Background(), fakeOpenAPIFinancialsQuerier{
		reserveErr: pgx.ErrNoRows,
	}, uuid.New(), "")
	if err != nil {
		t.Fatalf("missing reserve fund should not fail: %v", err)
	}
	if _, ok := result["reserve_fund"]; ok {
		t.Fatalf("missing reserve fund should be omitted, got %+v", result)
	}
}

type fakeOpenAPIFinancialsQuerier struct {
	reserveErr error
}

func (f fakeOpenAPIFinancialsQuerier) ListOpenAPIBudgetLinesByScheme(context.Context, dbgen.ListOpenAPIBudgetLinesBySchemeParams) ([]dbgen.BudgetLine, error) {
	return []dbgen.BudgetLine{}, nil
}

func (f fakeOpenAPIFinancialsQuerier) GetOpenAPIReserveFundByScheme(context.Context, uuid.UUID) (dbgen.ReserveFund, error) {
	return dbgen.ReserveFund{}, f.reserveErr
}
