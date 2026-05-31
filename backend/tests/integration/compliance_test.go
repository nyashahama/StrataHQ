//go:build integration

package integration

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestComplianceCreateItemRejectsWhitespaceRequiredTextFields(t *testing.T) {
	h := newComplianceHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)
	trusteeUserID := createMemberRecord(t, orgID, schemeID, uniqueEmail(t), "Trustee User", string(auth.RoleTrustee), nil)

	basePayload := map[string]string{
		"category":    string(dbgen.ComplianceCategoryFinancial),
		"title":       "Reserve fund review",
		"requirement": "Confirm annual reserve fund planning",
		"detail":      "Review the latest approved maintenance plan",
		"action":      "Upload trustee approval evidence",
	}

	for _, field := range []string{"title", "requirement", "detail", "action"} {
		t.Run(field, func(t *testing.T) {
			payload := make(map[string]string, len(basePayload))
			for key, value := range basePayload {
				payload[key] = value
			}
			payload[field] = "   "

			req := complianceCreateItemRequest(t, schemeID, payload)
			req = req.WithContext(auth.ContextWithIdentity(req.Context(), trusteeUserID, orgID, string(auth.RoleTrustee)))
			w := httptest.NewRecorder()
			h.CreateItem(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("create compliance item with blank %s should be rejected: status=%d body=%s", field, w.Code, w.Body)
			}
		})
	}
}

func TestComplianceCreateItemStoresTrimmedRequiredTextFields(t *testing.T) {
	h := newComplianceHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)
	trusteeUserID := createMemberRecord(t, orgID, schemeID, uniqueEmail(t), "Trustee User", string(auth.RoleTrustee), nil)

	req := complianceCreateItemRequest(t, schemeID, map[string]string{
		"category":    " " + string(dbgen.ComplianceCategoryFinancial) + " ",
		"title":       "  Reserve fund review  ",
		"requirement": "  Confirm annual reserve fund planning  ",
		"detail":      "  Review the latest approved maintenance plan  ",
		"action":      "  Upload trustee approval evidence  ",
	})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), trusteeUserID, orgID, string(auth.RoleTrustee)))
	w := httptest.NewRecorder()
	h.CreateItem(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create compliance item with padded fields: status=%d body=%s", w.Code, w.Body)
	}

	item := decodeSuccess[compliance.ItemInfo](t, w)
	if item.Category != string(dbgen.ComplianceCategoryFinancial) ||
		item.Title != "Reserve fund review" ||
		item.Requirement != "Confirm annual reserve fund planning" ||
		item.Detail != "Review the latest approved maintenance plan" ||
		item.Action != "Upload trustee approval evidence" {
		t.Fatalf("expected trimmed compliance item fields, got %+v", item)
	}
}

func complianceCreateItemRequest(t *testing.T, schemeID string, payload map[string]string) *http.Request {
	t.Helper()

	body := `{"category":"` + payload["category"] +
		`","title":"` + payload["title"] +
		`","requirement":"` + payload["requirement"] +
		`","detail":"` + payload["detail"] +
		`","action":"` + payload["action"] + `"}`
	req := httptest.NewRequest(http.MethodPost, "/compliance/"+schemeID+"/items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return withRouteParams(req, map[string]string{"schemeId": schemeID})
}

func timePointer(value time.Time) *time.Time {
	return &value
}
