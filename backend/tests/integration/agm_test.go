//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stratahq/backend/internal/agm"
	"github.com/stratahq/backend/internal/auth"
)

func newAgmHandler(t *testing.T) *agm.Handler {
	t.Helper()
	return agm.NewHandler(agm.NewService(testPool))
}

func TestAgm_ScheduleVoteAndAssignProxy(t *testing.T) {
	h := newAgmHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitID := createUnitRecord(t, schemeID, "3C")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident Owner", string(auth.RoleResident), &unitID)
	trusteeEmail := uniqueEmail(t)
	trusteeUserID := createMemberRecord(t, orgID, schemeID, trusteeEmail, "Trustee Member", string(auth.RoleTrustee), nil)

	createBody, _ := json.Marshal(map[string]any{
		"date":            "2026-11-20",
		"quorum_required": 3,
		"resolutions": []map[string]any{
			{
				"title":       "Approve 2027 maintenance budget",
				"description": "Adopt the proposed maintenance budget for 2027.",
			},
			{
				"title":       "Appoint trustee committee",
				"description": "Confirm the next trustee committee term.",
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/agm/"+schemeID+"/meetings", bytes.NewReader(createBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.ScheduleMeeting(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("schedule AGM: status=%d body=%s", w.Code, w.Body)
	}
	meeting := decodeSuccess[agm.MeetingInfo](t, w)
	if len(meeting.Resolutions) != 2 || meeting.Status != "upcoming" {
		t.Fatalf("unexpected scheduled meeting: %+v", meeting)
	}

	req = httptest.NewRequest(http.MethodGet, "/agm/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident dashboard: status=%d body=%s", w.Code, w.Body)
	}
	dashboard := decodeSuccess[agm.DashboardResponse](t, w)
	if dashboard.Upcoming == nil || len(dashboard.Upcoming.Resolutions) != 2 {
		t.Fatalf("unexpected dashboard response: %+v", dashboard)
	}

	voteBody, _ := json.Marshal(map[string]any{"choice": "for"})
	req = httptest.NewRequest(http.MethodPost, "/agm/"+schemeID+"/resolutions/"+dashboard.Upcoming.Resolutions[0].ID+"/vote", bytes.NewReader(voteBody))
	req = withRouteParams(req, map[string]string{
		"schemeId":     schemeID,
		"resolutionId": dashboard.Upcoming.Resolutions[0].ID,
	})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.CastVote(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("cast vote: status=%d body=%s", w.Code, w.Body)
	}
	votedResolution := decodeSuccess[agm.ResolutionInfo](t, w)
	if votedResolution.UserVote == nil || *votedResolution.UserVote != "for" || votedResolution.VotesFor != 1 {
		t.Fatalf("unexpected vote response: %+v", votedResolution)
	}

	proxyBody, _ := json.Marshal(map[string]any{"grantee_user_id": trusteeUserID})
	req = httptest.NewRequest(http.MethodPost, "/agm/"+schemeID+"/meetings/"+dashboard.Upcoming.ID+"/proxy", bytes.NewReader(proxyBody))
	req = withRouteParams(req, map[string]string{
		"schemeId":  schemeID,
		"meetingId": dashboard.Upcoming.ID,
	})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.AssignProxy(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("assign proxy: status=%d body=%s", w.Code, w.Body)
	}

	req = httptest.NewRequest(http.MethodGet, "/agm/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident dashboard after actions: status=%d body=%s", w.Code, w.Body)
	}
	updatedDashboard := decodeSuccess[agm.DashboardResponse](t, w)
	if updatedDashboard.Upcoming == nil || updatedDashboard.Upcoming.UserProxyGranteeID == nil || *updatedDashboard.Upcoming.UserProxyGranteeID != trusteeUserID {
		t.Fatalf("expected proxy assignment in dashboard, got %+v", updatedDashboard)
	}
	if updatedDashboard.Upcoming.Resolutions[0].UserVote == nil || *updatedDashboard.Upcoming.Resolutions[0].UserVote != "for" {
		t.Fatalf("expected vote in dashboard, got %+v", updatedDashboard.Upcoming.Resolutions[0])
	}
}
