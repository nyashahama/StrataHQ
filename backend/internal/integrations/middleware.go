package integrations

import (
	"context"
	"net/http"
)

type contextKey string

const identityKey contextKey = "integration_identity"

type Identity struct {
	ClientID  string
	OrgID     string
	Scopes    []string
	SchemeIDs []string
}

func ContextWithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey, identity)
}

func IdentityFromRequest(r *http.Request) (Identity, bool) {
	if r == nil {
		return Identity{}, false
	}
	identity, ok := r.Context().Value(identityKey).(Identity)
	return identity, ok && identity.ClientID != "" && identity.OrgID != ""
}

func (i Identity) HasScope(scope string) bool {
	for _, item := range i.Scopes {
		if item == scope {
			return true
		}
	}
	return false
}

func (i Identity) CanAccessScheme(schemeID string) bool {
	for _, item := range i.SchemeIDs {
		if item == schemeID {
			return true
		}
	}
	return false
}
