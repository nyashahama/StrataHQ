//go:build integration

package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/compliance"
)

func newComplianceHandler(t *testing.T) *compliance.Handler {
	t.Helper()
	return compliance.NewHandler(compliance.NewService(testPool))
}

func TestComplianceDashboard(t *testing.T) {
	h := newComplianceHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitID := createUnitRecord(t, schemeID, "4B")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unitID)
	trusteeEmail := uniqueEmail(t)
	trusteeUserID := createMemberRecord(t, orgID, schemeID, trusteeEmail, "Trustee User", string(auth.RoleTrustee), nil)

	schemeUUID := mustParseUUID(schemeID)
	assessedAt := time.Now().UTC()
	for _, item := range []struct {
		category dbgen.ComplianceCategory
		title    string
		status   dbgen.ComplianceStatus
		dueDate  *time.Time
	}{
		{
			category: dbgen.ComplianceCategoryFinancial,
			title:    "Reserve fund minimum contribution",
			status:   dbgen.ComplianceStatusAtRisk,
			dueDate:  timePointer(time.Date(2025, time.December, 1, 0, 0, 0, 0, time.UTC)),
		},
		{
			category: dbgen.ComplianceCategoryAdministrative,
			title:    "Scheme rules registered with CSOS",
			status:   dbgen.ComplianceStatusNonCompliant,
			dueDate:  timePointer(time.Date(2025, time.November, 30, 0, 0, 0, 0, time.UTC)),
		},
		{
			category: dbgen.ComplianceCategoryInsurance,
			title:    "Building insurance in force",
			status:   dbgen.ComplianceStatusCompliant,
		},
	} {
		dueDate := pgtype.Date{}
		if item.dueDate != nil {
			dueDate = pgtype.Date{Time: *item.dueDate, Valid: true}
		}
		if _, err := testQ.CreateComplianceItem(t.Context(), dbgen.CreateComplianceItemParams{
			SchemeID:    schemeUUID,
			Category:    item.category,
			Title:       item.title,
			Requirement: "required",
			Status:      item.status,
			Detail:      "detail",
			Action:      "action",
			DueDate:     dueDate,
			AssessedAt:  assessedAt,
		}); err != nil {
			t.Fatalf("create compliance item: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/compliance/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), trusteeUserID, orgID, string(auth.RoleTrustee)))
	w := httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("trustee compliance dashboard: status=%d body=%s", w.Code, w.Body)
	}

	dashboard := decodeSuccess[compliance.DashboardResponse](t, w)
	if dashboard.Total != 3 || dashboard.CompliantCount != 1 || dashboard.AtRiskCount != 1 || dashboard.NonCompliantCount != 1 {
		t.Fatalf("unexpected compliance counts: %+v", dashboard)
	}
	if dashboard.Score != 50 {
		t.Fatalf("unexpected compliance score: %+v", dashboard)
	}

	req = httptest.NewRequest(http.MethodGet, "/compliance/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("resident compliance dashboard should be forbidden: status=%d body=%s", w.Code, w.Body)
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}
