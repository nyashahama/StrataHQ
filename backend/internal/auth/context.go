package auth

import (
	"context"
	"net/http"
)

type Identity struct {
	UserID string
	OrgID  string
	Role   string
}

func ContextWithIdentity(ctx context.Context, userID, orgID, role string) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	ctx = context.WithValue(ctx, OrgIDKey, orgID)
	ctx = context.WithValue(ctx, RoleKey, role)
	return ctx
}

func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	if claims == nil {
		return ctx
	}
	return ContextWithIdentity(ctx, claims.Subject, claims.OrgID, claims.Role)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity := Identity{
		UserID: valueFromContext(ctx, UserIDKey),
		OrgID:  valueFromContext(ctx, OrgIDKey),
		Role:   valueFromContext(ctx, RoleKey),
	}
	if identity.UserID == "" || identity.OrgID == "" || identity.Role == "" {
		return Identity{}, false
	}
	return identity, true
}

func IdentityFromRequest(r *http.Request) (Identity, bool) {
	if r == nil {
		return Identity{}, false
	}
	return IdentityFromContext(r.Context())
}

func valueFromContext(ctx context.Context, key ContextKey) string {
	value, _ := ctx.Value(key).(string)
	return value
}
