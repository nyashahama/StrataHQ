//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/scheme"
)

func createMemberRecord(t *testing.T, orgID, schemeID, email, fullName, role string, unitID *string) string {
	t.Helper()

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		t.Fatalf("parse org id: %v", err)
	}
	schemeUUID, err := uuid.Parse(schemeID)
	if err != nil {
		t.Fatalf("parse scheme id: %v", err)
	}

	user, err := testQ.CreateUser(t.Context(), dbgen.CreateUserParams{
		Email:        email,
		PasswordHash: "test-hash",
		FullName:     fullName,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if _, err := testQ.CreateOrgMembership(t.Context(), dbgen.CreateOrgMembershipParams{
		UserID: user.ID,
		OrgID:  orgUUID,
		Role:   role,
	}); err != nil {
		t.Fatalf("create org membership: %v", err)
	}

	unitValue := pgtype.UUID{}
	if unitID != nil {
		parsedUnitID, parseErr := uuid.Parse(*unitID)
		if parseErr != nil {
			t.Fatalf("parse unit id: %v", parseErr)
		}
		unitValue = pgtype.UUID{Bytes: parsedUnitID, Valid: true}
	}

	if _, err := testQ.UpsertSchemeMembership(t.Context(), dbgen.UpsertSchemeMembershipParams{
		UserID:   user.ID,
		SchemeID: schemeUUID,
		UnitID:   unitValue,
		Role:     role,
	}); err != nil {
		t.Fatalf("create scheme membership: %v", err)
	}

	return user.ID.String()
}

func createUnitRecord(t *testing.T, schemeID, identifier string) string {
	t.Helper()
	schemeUUID, err := uuid.Parse(schemeID)
	if err != nil {
		t.Fatalf("parse scheme id: %v", err)
	}

	unit, err := testQ.CreateUnit(t.Context(), dbgen.CreateUnitParams{
		SchemeID:        schemeUUID,
		Identifier:      identifier,
		OwnerName:       "Owner " + identifier,
		Floor:           1,
		SectionValueBps: 500,
	})
	if err != nil {
		t.Fatalf("create unit: %v", err)
	}

	return unit.ID.String()
}

func TestScheme_MembersListAndUpdate(t *testing.T) {
	h := newSchemeHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unit1ID := createUnitRecord(t, schemeID, "1A")
	unit2ID := createUnitRecord(t, schemeID, "2B")

	trusteeEmail := uniqueEmail(t)
	residentEmail := uniqueEmail(t)
	trusteeUserID := createMemberRecord(t, orgID, schemeID, trusteeEmail, "Trustee User", string(auth.RoleTrustee), nil)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unit1ID)

	req := httptest.NewRequest(http.MethodGet, "/schemes/"+schemeID+"/members", nil)
	req = withRouteParams(req, map[string]string{"id": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.ListMembers(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list members: status=%d body=%s", w.Code, w.Body)
	}
	members := decodeSuccess[[]scheme.MemberInfo](t, w)
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	req = httptest.NewRequest(http.MethodGet, "/schemes/"+schemeID+"/members", nil)
	req = withRouteParams(req, map[string]string{"id": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.ListMembers(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident list members: status=%d body=%s", w.Code, w.Body)
	}
	residentView := decodeSuccess[[]scheme.MemberInfo](t, w)
	if len(residentView) != 1 || residentView[0].UserID != trusteeUserID {
		t.Fatalf("resident should only see trustee committee, got %+v", residentView)
	}

	updateBody, _ := json.Marshal(map[string]any{
		"role":    "trustee",
		"unit_id": nil,
	})
	req = httptest.NewRequest(http.MethodPatch, "/schemes/"+schemeID+"/members/"+residentUserID, bytes.NewReader(updateBody))
	req = withRouteParams(req, map[string]string{"id": schemeID, "userId": residentUserID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.UpdateMember(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update member to trustee: status=%d body=%s", w.Code, w.Body)
	}
	updated := decodeSuccess[scheme.MemberInfo](t, w)
	if updated.Role != string(auth.RoleTrustee) || updated.UnitID != nil {
		t.Fatalf("unexpected updated member: %+v", updated)
	}

	updateBody, _ = json.Marshal(map[string]any{
		"role":    "resident",
		"unit_id": unit2ID,
	})
	req = httptest.NewRequest(http.MethodPatch, "/schemes/"+schemeID+"/members/"+residentUserID, bytes.NewReader(updateBody))
	req = withRouteParams(req, map[string]string{"id": schemeID, "userId": residentUserID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.UpdateMember(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update member to resident: status=%d body=%s", w.Code, w.Body)
	}
	updated = decodeSuccess[scheme.MemberInfo](t, w)
	if updated.Role != string(auth.RoleResident) || updated.UnitIdentifier == nil || *updated.UnitIdentifier != "2B" {
		t.Fatalf("unexpected resident update: %+v", updated)
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		t.Fatalf("parse org id: %v", err)
	}
	residentUUID, err := uuid.Parse(residentUserID)
	if err != nil {
		t.Fatalf("parse resident user id: %v", err)
	}
	orgMembership, err := testQ.GetOrgMembershipByUser(t.Context(), dbgen.GetOrgMembershipByUserParams{
		UserID: residentUUID,
		OrgID:  orgUUID,
	})
	if err != nil {
		t.Fatalf("get org membership: %v", err)
	}
	if orgMembership.Role != string(auth.RoleResident) {
		t.Fatalf("expected org membership role resident, got %q", orgMembership.Role)
	}
}
