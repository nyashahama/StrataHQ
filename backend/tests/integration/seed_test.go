//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stratahq/backend/internal/seed"
)

func TestSeedDemo_IsIdempotent(t *testing.T) {
	service := seed.NewService(testPool)

	first, err := service.SeedDemo(context.Background())
	if err != nil {
		t.Fatalf("first seed failed: %v", err)
	}
	if first.OrgID == "" || first.SchemeID == "" {
		t.Fatalf("expected org and scheme ids to be populated, got org=%q scheme=%q", first.OrgID, first.SchemeID)
	}
	if first.UnitsSeeded != 8 {
		t.Fatalf("expected 8 units seeded, got %d", first.UnitsSeeded)
	}

	second, err := service.SeedDemo(context.Background())
	if err != nil {
		t.Fatalf("second seed failed: %v", err)
	}
	if !second.AlreadyExisted {
		t.Fatal("expected second seed run to report existing demo data")
	}
	if second.OrgID != first.OrgID {
		t.Fatalf("expected same org id on second seed run, got %q and %q", first.OrgID, second.OrgID)
	}
}
