package agm

import (
	"testing"

	"github.com/google/uuid"

	dbgen "github.com/stratahq/backend/db/gen"
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
