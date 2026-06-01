package compliance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
)

func TestPortfolioDashboardReturnsErrorWhenSchemeDashboardFails(t *testing.T) {
	t.Helper()

	failingSchemeID := uuid.New()
	successSchemeID := uuid.New()
	dashboardErr := errors.New("scheme dashboard failed")

	svc := &Service{
		listSchemesByOrgFn: func(_ context.Context, _ uuid.UUID) ([]dbgen.Scheme, error) {
			return []dbgen.Scheme{
				{
					ID:   successSchemeID,
					Name: "Good Scheme",
				},
				{
					ID:   failingSchemeID,
					Name: "Broken Scheme",
				},
			}, nil
		},
		dashboardForSchemeFn: func(_ context.Context, schemeID uuid.UUID) (*DashboardResponse, error) {
			if schemeID == failingSchemeID {
				return nil, dashboardErr
			}
			return &DashboardResponse{Score: 100}, nil
		},
	}

	identity := auth.Identity{
		UserID: uuid.NewString(),
		OrgID:  uuid.NewString(),
		Role:   string(auth.RoleAdmin),
	}

	_, err := svc.PortfolioDashboard(context.Background(), identity)
	if !errors.Is(err, dashboardErr) {
		t.Fatalf("expected portfolio dashboard to fail with scheme dashboard error, got %v", err)
	}
}

func TestAssessReturnsErrorWhenAssessmentSnapshotPersistenceFails(t *testing.T) {
	t.Helper()

	schemeID := uuid.New()
	orgID := uuid.New()
	identity := auth.Identity{
		UserID: uuid.NewString(),
		OrgID:  orgID.String(),
		Role:   string(auth.RoleAdmin),
	}
	dashboard := &DashboardResponse{
		Score:             50,
		Total:             3,
		CompliantCount:    1,
		AtRiskCount:       1,
		NonCompliantCount: 1,
	}
	persistErr := errors.New("assessment snapshot insert failed")
	var persisted bool

	svc := &Service{
		dashboardFn: func(_ context.Context, gotIdentity auth.Identity, gotSchemeID string) (*DashboardResponse, error) {
			if gotIdentity != identity {
				t.Fatalf("dashboard identity = %+v, want %+v", gotIdentity, identity)
			}
			if gotSchemeID != schemeID.String() {
				t.Fatalf("dashboard schemeID = %q, want %q", gotSchemeID, schemeID.String())
			}
			return dashboard, nil
		},
		resolveSchemeAccessFn: func(_ context.Context, gotIdentity auth.Identity, gotSchemeID string) (dbgen.Scheme, string, error) {
			if gotIdentity != identity {
				t.Fatalf("resolve identity = %+v, want %+v", gotIdentity, identity)
			}
			if gotSchemeID != schemeID.String() {
				t.Fatalf("resolve schemeID = %q, want %q", gotSchemeID, schemeID.String())
			}
			return dbgen.Scheme{ID: schemeID, OrgID: orgID}, string(auth.RoleAdmin), nil
		},
		createAssessmentFn: func(_ context.Context, params dbgen.CreateComplianceAssessmentParams) (dbgen.ComplianceAssessment, error) {
			persisted = true
			if params.SchemeID != schemeID {
				t.Fatalf("assessment schemeID = %s, want %s", params.SchemeID, schemeID)
			}
			if params.Score != 50 || params.TotalItems != 3 || params.CompliantCount != 1 ||
				params.AtRiskCount != 1 || params.NonCompliantCount != 1 {
				t.Fatalf("unexpected assessment params: %+v", params)
			}
			return dbgen.ComplianceAssessment{}, persistErr
		},
	}

	got, err := svc.Assess(context.Background(), identity, schemeID.String())
	if !errors.Is(err, persistErr) {
		t.Fatalf("expected assessment persistence error, got result=%+v err=%v", got, err)
	}
	if got != nil {
		t.Fatalf("expected no dashboard when assessment persistence fails, got %+v", got)
	}
	if !persisted {
		t.Fatal("expected assessment snapshot persistence to be attempted")
	}
}

func TestDashboardForSchemePropagatesCountUpcomingDeadlinesError(t *testing.T) {
	t.Helper()

	schemeID := uuid.New()
	deadlineErr := errors.New("count upcoming deadlines failed")

	svc := &Service{
		listComplianceItemsByScheme: func(_ context.Context, _ uuid.UUID) ([]dbgen.ComplianceItem, error) {
			return nil, nil
		},
		countUpcomingDeadlines: func(_ context.Context, _ uuid.UUID) (int64, error) {
			return 0, deadlineErr
		},
	}

	got, err := svc.dashboardForScheme(context.Background(), schemeID)
	if !errors.Is(err, deadlineErr) {
		t.Fatalf("expected deadline count error, got result=%+v err=%v", got, err)
	}
	if got != nil {
		t.Fatalf("expected nil dashboard when deadline count fails, got %+v", got)
	}
}

func TestDashboardForSchemePropagatesListItemsError(t *testing.T) {
	t.Helper()

	schemeID := uuid.New()
	listErr := errors.New("list items failed")

	svc := &Service{
		listComplianceItemsByScheme: func(_ context.Context, _ uuid.UUID) ([]dbgen.ComplianceItem, error) {
			return nil, listErr
		},
		countUpcomingDeadlines: func(_ context.Context, _ uuid.UUID) (int64, error) {
			return 0, nil
		},
	}

	got, err := svc.dashboardForScheme(context.Background(), schemeID)
	if !errors.Is(err, listErr) {
		t.Fatalf("expected list items error, got result=%+v err=%v", got, err)
	}
	if got != nil {
		t.Fatalf("expected nil dashboard when list items fails, got %+v", got)
	}
}
