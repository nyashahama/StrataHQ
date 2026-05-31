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
