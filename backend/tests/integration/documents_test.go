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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/documents"
)

func newDocumentsHandler(t *testing.T) *documents.Handler {
	t.Helper()
	return documents.NewHandler(documents.NewService(testPool))
}

func TestDocuments_CreateListFilterAndDelete(t *testing.T) {
	h := newDocumentsHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitID := createUnitRecord(t, schemeID, "3A")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unitID)

	createBody, _ := json.Marshal(map[string]any{
		"name":        "Conduct Rules",
		"storage_key": "data:application/pdf;base64,VEVTVA==",
		"file_type":   "pdf",
		"category":    "rules",
		"size_bytes":  4,
		"visibility":  "all",
	})
	req := httptest.NewRequest(http.MethodPost, "/documents/"+schemeID, bytes.NewReader(createBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create document: status=%d body=%s", w.Code, w.Body)
	}
	created := decodeSuccess[documents.DocumentInfo](t, w)
	if created.Category != "rules" || created.StorageKey == "" {
		t.Fatalf("unexpected created document: %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/documents/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident list documents: status=%d body=%s", w.Code, w.Body)
	}
	listResp := decodeSuccess[documents.DashboardResponse](t, w)
	if listResp.Total != 1 || len(listResp.Documents) != 1 {
		t.Fatalf("unexpected documents list: %+v", listResp)
	}

	req = httptest.NewRequest(http.MethodGet, "/documents/"+schemeID+"?category=rules", nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("filter documents: status=%d body=%s", w.Code, w.Body)
	}
	filtered := decodeSuccess[documents.DashboardResponse](t, w)
	if filtered.Total != 1 || filtered.Documents[0].Category != "rules" {
		t.Fatalf("unexpected filtered documents: %+v", filtered)
	}

	req = httptest.NewRequest(http.MethodDelete, "/documents/"+schemeID+"/"+created.ID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID, "id": created.ID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Delete(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete document: status=%d body=%s", w.Code, w.Body)
	}

	req = httptest.NewRequest(http.MethodDelete, "/documents/"+schemeID+"/"+created.ID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID, "id": created.ID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.Delete(w, req)
	if w.Code != http.StatusForbidden {
		t.		Fatalf("resident delete should be forbidden: status=%d body=%s", w.Code, w.Body)
	}
}

func TestDocuments_VisibilityFiltering(t *testing.T) {
	ctx := context.Background()
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)
	schemeUUID := uuid.MustParse(schemeID)

	adminClaims, err := auth.ValidateAccessToken(accessToken, testJWTSigningKey, "", "")
	if err != nil {
		t.Fatalf("validate admin token: %v", err)
	}
	_ = adminClaims

	unitID := createUnitRecord(t, schemeID, "5E")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unitID)

	// Create trustee
	trusteeEmail := uniqueEmail(t)
	trusteeUserID := createMemberRecord(t, orgID, schemeID, trusteeEmail, "Trustee User", string(auth.RoleTrustee), nil)

	// Create documents with different visibilities using testQ directly
	_, err = testQ.CreateSchemeDocument(ctx, dbgen.CreateSchemeDocumentParams{
		SchemeID:         schemeUUID,
		Name:             "Public Rules",
		StorageKey:       "/docs/public.pdf",
		FileType:         dbgen.DocumentFileTypePdf,
		Category:         dbgen.DocumentCategoryRules,
		SizeBytes:        100,
		UploadedByUserID: pgtype.UUID{Valid: false},
		Visibility:       dbgen.DocumentVisibilityAll,
	})
	if err != nil {
		t.Fatalf("create public doc: %v", err)
	}

	_, err = testQ.CreateSchemeDocument(ctx, dbgen.CreateSchemeDocumentParams{
		SchemeID:         schemeUUID,
		Name:             "Admin Only Financials",
		StorageKey:       "/docs/admin_finance.pdf",
		FileType:         dbgen.DocumentFileTypePdf,
		Category:         dbgen.DocumentCategoryFinancial,
		SizeBytes:        200,
		UploadedByUserID: pgtype.UUID{Valid: false},
		Visibility:       dbgen.DocumentVisibilityAdmin,
	})
	if err != nil {
		t.Fatalf("create admin doc: %v", err)
	}

	_, err = testQ.CreateSchemeDocument(ctx, dbgen.CreateSchemeDocumentParams{
		SchemeID:         schemeUUID,
		Name:             "Trustee Minutes",
		StorageKey:       "/docs/minutes.pdf",
		FileType:         dbgen.DocumentFileTypePdf,
		Category:         dbgen.DocumentCategoryMinutes,
		SizeBytes:        300,
		UploadedByUserID: pgtype.UUID{Valid: false},
		Visibility:       dbgen.DocumentVisibilityTrustee,
	})
	if err != nil {
		t.Fatalf("create trustee doc: %v", err)
	}

	h := newDocumentsHandler(t)

	// Admin sees all 3
	req := httptest.NewRequest(http.MethodGet, "/documents/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin list: status=%d body=%s", w.Code, w.Body)
	}
	adminDocs := decodeSuccess[documents.DashboardResponse](t, w)
	if adminDocs.Total < 3 {
		t.Fatalf("admin sees %d docs, want >= 3", adminDocs.Total)
	}

	// Resident sees only "all" visibility (1 doc)
	residentToken, err := auth.GenerateAccessToken(
		residentUserID, orgID, string(auth.RoleResident),
		"http://localhost:3000", "stratahq-api",
		testJWTSigningKey, 15*time.Minute,
	)
	if err != nil {
		t.Fatalf("generate resident token: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/documents/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, residentToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident list: status=%d body=%s", w.Code, w.Body)
	}
	residentDocs := decodeSuccess[documents.DashboardResponse](t, w)
	if residentDocs.Total != 1 {
		t.Fatalf("resident sees %d docs, want 1", residentDocs.Total)
	}
	if residentDocs.Documents[0].Name != "Public Rules" {
		t.Fatalf("resident sees doc %q, want 'Public Rules'", residentDocs.Documents[0].Name)
	}

	// Trustee sees "all" + "trustee" (2 docs)
	trusteeToken, err := auth.GenerateAccessToken(
		trusteeUserID, orgID, string(auth.RoleTrustee),
		"http://localhost:3000", "stratahq-api",
		testJWTSigningKey, 15*time.Minute,
	)
	if err != nil {
		t.Fatalf("generate trustee token: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/documents/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, trusteeToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("trustee list: status=%d body=%s", w.Code, w.Body)
	}
	trusteeDocs := decodeSuccess[documents.DashboardResponse](t, w)
	if trusteeDocs.Total < 2 {
		t.Fatalf("trustee sees %d docs, want >= 2", trusteeDocs.Total)
	}
	hasTrusteeDoc := false
	for _, d := range trusteeDocs.Documents {
		if d.Name == "Trustee Minutes" {
			hasTrusteeDoc = true
			break
		}
	}
	if !hasTrusteeDoc {
		t.Fatalf("trustee should see 'Trustee Minutes' doc, got: %+v", trusteeDocs.Documents)
	}
}
