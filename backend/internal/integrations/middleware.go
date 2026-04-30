package integrations

import (
	"context"
	"net/http"
	"strings"

	"github.com/stratahq/backend/internal/platform/response"
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

func (s *Service) APIKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing integration api key")
			return
		}
		identity, err := s.AuthenticateAPIKey(r.Context(), parts[1])
		if err != nil {
			response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid integration api key")
			return
		}
		next.ServeHTTP(w, r.WithContext(ContextWithIdentity(r.Context(), identity)))
	})
}
