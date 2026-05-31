package agm

import (
	"testing"
	"time"

	"github.com/google/uuid"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
)

func TestCalculateVoteTotalsCountsDelegatedVotes(t *testing.T) {
	granteeID := uuid.New()
	grantorID := uuid.New()

	votesFor, votesAgainst := calculateVoteTotals(
		[]dbgen.AgmVote{
			{VoterUserID: granteeID, Vote: dbgen.VoteChoiceFor},
		},
		[]dbgen.ProxyAssignment{
			{GrantorUserID: grantorID, GranteeUserID: granteeID},
		},
	)

	if votesFor != 2 || votesAgainst != 0 {
		t.Fatalf("calculateVoteTotals() = (%d, %d), want (2, 0)", votesFor, votesAgainst)
	}
}

func TestCalculateVoteTotalsDirectGrantorVoteOverridesDelegation(t *testing.T) {
	granteeID := uuid.New()
	grantorID := uuid.New()

	votesFor, votesAgainst := calculateVoteTotals(
		[]dbgen.AgmVote{
			{VoterUserID: granteeID, Vote: dbgen.VoteChoiceFor},
			{VoterUserID: grantorID, Vote: dbgen.VoteChoiceAgainst},
		},
		[]dbgen.ProxyAssignment{
			{GrantorUserID: grantorID, GranteeUserID: granteeID},
		},
	)

	if votesFor != 1 || votesAgainst != 1 {
		t.Fatalf("calculateVoteTotals() = (%d, %d), want (1, 1)", votesFor, votesAgainst)
	}
}

func TestHasOutgoingProxyAssignment(t *testing.T) {
	grantorID := uuid.New()
	granteeID := uuid.New()

	if !hasOutgoingProxyAssignment([]dbgen.ProxyAssignment{
		{GrantorUserID: grantorID, GranteeUserID: granteeID},
	}, grantorID) {
		t.Fatal("hasOutgoingProxyAssignment() = false, want true")
	}
}

func TestAgmVoteAuditEvent(t *testing.T) {
	event := agmVoteAuditEvent(agmVoteAuditInput{
		SchemeID:     "scheme-1",
		OrgID:        "org-1",
		ActorUserID:  "user-1",
		ActorRole:    "trustee",
		ResolutionID: "resolution-1",
		Choice:       "for",
	})

	if event.Action != "agm.vote_cast" {
		t.Fatalf("action = %q, want agm.vote_cast", event.Action)
	}
	if event.ResourceType != "agm_resolution" {
		t.Fatalf("resource type = %q, want agm_resolution", event.ResourceType)
	}
}

func TestEligibleVoterCountExcludesAdmins(t *testing.T) {
	members := []dbgen.ListSchemeMembersBySchemeRow{
		{Role: string(auth.RoleAdmin)},
		{Role: string(auth.RoleTrustee)},
		{Role: string(auth.RoleResident)},
	}

	if got := eligibleVoterCount(members); got != 2 {
		t.Fatalf("eligibleVoterCount() = %d, want 2", got)
	}
}

func TestValidateScheduleMeetingInputRequiresResolution(t *testing.T) {
	err := validateScheduleMeetingInput(ScheduleMeetingInput{
		MeetingDate:    time.Now().Add(24 * time.Hour),
		QuorumRequired: 1,
		Resolutions:    []ScheduleResolutionInput{},
	}, 1)

	if err != ErrInvalidInput {
		t.Fatalf("validateScheduleMeetingInput() error = %v, want ErrInvalidInput", err)
	}
}

func TestValidateScheduleMeetingInputRejectsBlankResolution(t *testing.T) {
	err := validateScheduleMeetingInput(ScheduleMeetingInput{
		MeetingDate:    time.Now().Add(24 * time.Hour),
		QuorumRequired: 1,
		Resolutions: []ScheduleResolutionInput{
			{Title: "  ", Description: "Approve budget"},
		},
	}, 1)

	if err != ErrInvalidInput {
		t.Fatalf("validateScheduleMeetingInput() error = %v, want ErrInvalidInput", err)
	}
}

func TestValidateScheduleMeetingInputAcceptsVotingMembersOnlyQuorum(t *testing.T) {
	err := validateScheduleMeetingInput(ScheduleMeetingInput{
		MeetingDate:    time.Now().Add(24 * time.Hour),
		QuorumRequired: 2,
		Resolutions: []ScheduleResolutionInput{
			{Title: "Approve budget", Description: "Approve the annual budget"},
		},
	}, 2)

	if err != nil {
		t.Fatalf("validateScheduleMeetingInput() error = %v, want nil", err)
	}
}
