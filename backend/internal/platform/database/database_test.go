package database

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestNewPoolConfigUsesPgBouncerCompatibleSimpleProtocol(t *testing.T) {
	cfg, err := newPoolConfig("postgres://user:pass@localhost:5432/db")
	if err != nil {
		t.Fatalf("newPoolConfig returned error: %v", err)
	}

	if got := cfg.ConnConfig.DefaultQueryExecMode; got != pgx.QueryExecModeSimpleProtocol {
		t.Fatalf("DefaultQueryExecMode = %v, want %v", got, pgx.QueryExecModeSimpleProtocol)
	}
}
