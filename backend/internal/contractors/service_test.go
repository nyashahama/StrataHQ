package contractors

import (
	"os"
	"strings"
	"testing"
)

func TestValidRating(t *testing.T) {
	for _, rating := range []int32{1, 3, 5} {
		if !validRating(rating) {
			t.Fatalf("rating %d should be valid", rating)
		}
	}
	for _, rating := range []int32{0, 6} {
		if validRating(rating) {
			t.Fatalf("rating %d should be invalid", rating)
		}
	}
}

func TestNormalizeRequired(t *testing.T) {
	got, ok := normalizeRequired("  AquaFix Plumbing  ")
	if !ok {
		t.Fatal("expected non-empty value")
	}
	if got != "AquaFix Plumbing" {
		t.Fatalf("normalized = %q", got)
	}
	if _, ok := normalizeRequired("   "); ok {
		t.Fatal("blank value should be invalid")
	}
}

func TestContractorAggregateQueriesUseDistinctCounts(t *testing.T) {
	sql, err := os.ReadFile("../../db/queries/contractors.sql")
	if err != nil {
		t.Fatalf("read contractors.sql: %v", err)
	}
	content := string(sql)
	for _, expected := range []string{
		"COUNT(DISTINCT cr.id)::int AS review_count",
		"COUNT(DISTINCT mr.id) FILTER (WHERE mr.status = 'resolved')::int AS completed_job_count",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("contractor aggregate query missing %q", expected)
		}
	}
}
