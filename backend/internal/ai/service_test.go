package ai

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
)

func TestMapMembersRedactsPersonalIdentifiers(t *testing.T) {
	items := mapMembers([]dbgen.ListSchemeMembersBySchemeRow{
		{
			FullName: "Resident Owner",
			Email:    "resident@example.com",
			Role:     "resident",
			UnitID:   pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UnitIdentifier: pgtype.Text{
				String: "7B",
				Valid:  true,
			},
		},
	}, 5)

	if len(items) != 1 {
		t.Fatalf("mapMembers() returned %d items, want 1", len(items))
	}
	if _, ok := items[0]["full_name"]; ok {
		t.Fatalf("mapMembers() leaked full_name: %+v", items[0])
	}
	if _, ok := items[0]["email"]; ok {
		t.Fatalf("mapMembers() leaked email: %+v", items[0])
	}
	if _, ok := items[0]["unit_id"]; ok {
		t.Fatalf("mapMembers() leaked unit_id: %+v", items[0])
	}
	if items[0]["role"] != "resident" {
		t.Fatalf("mapMembers() role = %v, want resident", items[0]["role"])
	}
	if items[0]["unit_identifier"] != "7B" {
		t.Fatalf("mapMembers() unit_identifier = %v, want 7B", items[0]["unit_identifier"])
	}
}

func TestMapUnitsRedactsOwnerName(t *testing.T) {
	items := mapUnits([]dbgen.Unit{
		{
			Identifier:      "7B",
			OwnerName:       "Resident Owner",
			Floor:           7,
			SectionValueBps: 1250,
		},
	}, 5)

	if len(items) != 1 {
		t.Fatalf("mapUnits() returned %d items, want 1", len(items))
	}
	if _, ok := items[0]["owner_name"]; ok {
		t.Fatalf("mapUnits() leaked owner_name: %+v", items[0])
	}
}

func TestTopOverdueAccountsRedactsOwnerName(t *testing.T) {
	items := topOverdueAccounts([]dbgen.ListLevyAccountsByPeriodRow{
		{
			UnitIdentifier: "7B",
			OwnerName:      "Resident Owner",
			AmountCents:    245000,
			PaidCents:      0,
			DueDate:        pgtype.Date{Time: time.Now().AddDate(0, 0, -2), Valid: true},
		},
	}, 5)

	if len(items) != 1 {
		t.Fatalf("topOverdueAccounts() returned %d items, want 1", len(items))
	}
	if _, ok := items[0]["owner_name"]; ok {
		t.Fatalf("topOverdueAccounts() leaked owner_name: %+v", items[0])
	}
}

func TestMapMaintenanceRedactsContractorName(t *testing.T) {
	items := mapMaintenance([]dbgen.MaintenanceRequest{
		{
			Title:    "Fix leak",
			Category: dbgen.MaintenanceCategoryPlumbing,
			Status:   dbgen.MaintenanceStatusOpen,
			ContractorName: pgtype.Text{
				String: "Acme Plumbing",
				Valid:  true,
			},
			SubmittedByUnit: pgtype.Text{
				String: "7B",
				Valid:  true,
			},
			SlaHours:  24,
			CreatedAt: time.Now(),
		},
	}, 5)

	if len(items) != 1 {
		t.Fatalf("mapMaintenance() returned %d items, want 1", len(items))
	}
	if _, ok := items[0]["contractor_name"]; ok {
		t.Fatalf("mapMaintenance() leaked contractor_name: %+v", items[0])
	}
}
