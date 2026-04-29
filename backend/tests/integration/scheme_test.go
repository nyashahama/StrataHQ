//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/scheme"
)

func newSchemeHandler(t *testing.T) *scheme.Handler {
	t.Helper()
	return scheme.NewHandler(scheme.NewService(testPool))
}

func withRouteParams(req *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestScheme_AdminCoreFlow(t *testing.T) {
	h := newSchemeHandler(t)
	accessToken, _ := setupAgent(t)
	firstSchemeID := setupScheme(t, accessToken)

	req := httptest.NewRequest(http.MethodGet, "/schemes", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list schemes: status=%d body=%s", w.Code, w.Body)
	}
	listResp := decodeSuccess[[]scheme.SchemeSummary](t, w)
	if len(listResp) != 1 {
		t.Fatalf("expected 1 seeded scheme, got %d", len(listResp))
	}
	if listResp[0].Role != "admin" {
		t.Fatalf("expected admin role in summary, got %q", listResp[0].Role)
	}
	if listResp[0].LevyCollectionPct < 0 || listResp[0].LevyCollectionPct > 100 {
		t.Fatalf("expected levy collection pct in [0,100], got %d", listResp[0].LevyCollectionPct)
	}
	if listResp[0].TotalMembers < 0 || listResp[0].TrusteeCount < 0 || listResp[0].ResidentCount < 0 {
		t.Fatalf(
			"expected non-negative membership counts, got total=%d trustee=%d resident=%d",
			listResp[0].TotalMembers,
			listResp[0].TrusteeCount,
			listResp[0].ResidentCount,
		)
	}

	createBody, _ := json.Marshal(map[string]any{
		"name":       "Bluewater Gardens",
		"address":    "22 Beach Road",
		"unit_count": 18,
	})
	req = httptest.NewRequest(http.MethodPost, "/schemes", bytes.NewReader(createBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create scheme: status=%d body=%s", w.Code, w.Body)
	}
	createdScheme := decodeSuccess[scheme.SchemeSummary](t, w)
	if createdScheme.Name != "Bluewater Gardens" {
		t.Fatalf("created scheme name=%q", createdScheme.Name)
	}

	updateBody, _ := json.Marshal(map[string]any{
		"name":       "Bluewater Gardens North",
		"address":    "24 Beach Road",
		"unit_count": 20,
	})
	req = httptest.NewRequest(http.MethodPut, "/schemes/"+createdScheme.ID, bytes.NewReader(updateBody))
	req = withRouteParams(req, map[string]string{"id": createdScheme.ID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Update(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update scheme: status=%d body=%s", w.Code, w.Body)
	}
	updatedScheme := decodeSuccess[scheme.SchemeSummary](t, w)
	if updatedScheme.Name != "Bluewater Gardens North" || updatedScheme.UnitCount != 20 {
		t.Fatalf("unexpected updated scheme: %+v", updatedScheme)
	}

	unitBody, _ := json.Marshal(map[string]any{
		"identifier":        "1A",
		"owner_name":        "A. Adams",
		"floor":             1,
		"section_value_bps": 625,
	})
	req = httptest.NewRequest(http.MethodPost, "/schemes/"+firstSchemeID+"/units", bytes.NewReader(unitBody))
	req = withRouteParams(req, map[string]string{"id": firstSchemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.CreateUnit(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create unit: status=%d body=%s", w.Code, w.Body)
	}
	createdUnit := decodeSuccess[scheme.UnitInfo](t, w)
	if createdUnit.Identifier != "1A" {
		t.Fatalf("created unit identifier=%q", createdUnit.Identifier)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/schemes/"+firstSchemeID, nil)
	detailReq = withRouteParams(detailReq, map[string]string{"id": firstSchemeID})
	detailReq = withAuthContext(detailReq, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Get(w, detailReq)
	if w.Code != http.StatusOK {
		t.Fatalf("get scheme: status=%d body=%s", w.Code, w.Body)
	}
	detail := decodeSuccess[scheme.SchemeDetail](t, w)
	if len(detail.Units) != 1 {
		t.Fatalf("expected 1 unit in scheme detail, got %d", len(detail.Units))
	}

	unitUpdateBody, _ := json.Marshal(map[string]any{
		"identifier":        "1B",
		"owner_name":        "A. Adams",
		"floor":             2,
		"section_value_bps": 650,
	})
	req = httptest.NewRequest(http.MethodPut, "/schemes/"+firstSchemeID+"/units/"+createdUnit.ID, bytes.NewReader(unitUpdateBody))
	req = withRouteParams(req, map[string]string{"id": firstSchemeID, "unitId": createdUnit.ID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.UpdateUnit(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update unit: status=%d body=%s", w.Code, w.Body)
	}
	updatedUnit := decodeSuccess[scheme.UnitInfo](t, w)
	if updatedUnit.Identifier != "1B" || updatedUnit.Floor != 2 {
		t.Fatalf("unexpected updated unit: %+v", updatedUnit)
	}

	req = httptest.NewRequest(http.MethodDelete, "/schemes/"+createdScheme.ID, nil)
	req = withRouteParams(req, map[string]string{"id": createdScheme.ID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Delete(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete scheme: status=%d body=%s", w.Code, w.Body)
	}
}

func TestScheme_ResidentDetailSeesOnlyOwnUnit(t *testing.T) {
	ctx := context.Background()
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)
	schemeUUID := uuid.MustParse(schemeID)

	unitA, err := testQ.CreateUnit(ctx, createUnitParams(schemeID, "1A", "Alice Adams"))
	if err != nil {
		t.Fatalf("create unit A: %v", err)
	}
	unitB, err := testQ.CreateUnit(ctx, createUnitParams(schemeID, "2B", "Bob Brown"))
	if err != nil {
		t.Fatalf("create unit B: %v", err)
	}

	residentEmail := uniqueEmail(t)
	residentUser, err := testQ.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        residentEmail,
		PasswordHash: "test-hash",
		FullName:     "Resident User",
	})
	if err != nil {
		t.Fatalf("create resident user: %v", err)
	}
	_, err = testQ.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
		UserID:   residentUser.ID,
		SchemeID: schemeUUID,
		UnitID:   pgtype.UUID{Bytes: unitA.ID, Valid: true},
		Role:     string(auth.RoleResident),
	})
	if err != nil {
		t.Fatalf("create resident membership: %v", err)
	}

	residentToken, err := auth.GenerateAccessToken(
		residentUser.ID.String(), orgID, string(auth.RoleResident),
		"http://localhost:3000", "stratahq-api",
		testJWTSigningKey, 15*time.Minute,
	)
	if err != nil {
		t.Fatalf("generate resident token: %v", err)
	}

	h := newSchemeHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/schemes/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"id": schemeID})
	req = withAuthContext(req, residentToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.Get(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident detail: status=%d body=%s", w.Code, w.Body)
	}
	detail := decodeSuccess[scheme.SchemeDetail](t, w)

	if len(detail.Units) != 1 {
		t.Fatalf("resident sees %d units, want 1", len(detail.Units))
	}
	if detail.Units[0].Identifier != "1A" {
		t.Fatalf("resident sees unit %q, want 1A", detail.Units[0].Identifier)
	}

	// Trustee should see all units
	trusteeEmail := uniqueEmail(t)
	trusteeUser, err := testQ.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        trusteeEmail,
		PasswordHash: "test-hash",
		FullName:     "Trustee User",
	})
	if err != nil {
		t.Fatalf("create trustee user: %v", err)
	}
	_, err = testQ.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
		UserID:   trusteeUser.ID,
		SchemeID: schemeUUID,
		UnitID:   pgtype.UUID{},
		Role:     string(auth.RoleTrustee),
	})
	if err != nil {
		t.Fatalf("create trustee membership: %v", err)
	}

	trusteeToken, err := auth.GenerateAccessToken(
		trusteeUser.ID.String(), orgID, string(auth.RoleTrustee),
		"http://localhost:3000", "stratahq-api",
		testJWTSigningKey, 15*time.Minute,
	)
	if err != nil {
		t.Fatalf("generate trustee token: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/schemes/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"id": schemeID})
	req = withAuthContext(req, trusteeToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Get(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("trustee detail: status=%d body=%s", w.Code, w.Body)
	}
	trusteeDetail := decodeSuccess[scheme.SchemeDetail](t, w)
	if len(trusteeDetail.Units) < 2 {
		t.Fatalf("trustee sees %d units, want >= 2", len(trusteeDetail.Units))
	}

	_ = unitB
	_ = unitA
	_ = residentEmail
}
