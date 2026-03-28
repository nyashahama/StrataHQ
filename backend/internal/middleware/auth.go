package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

// Auth validates the Bearer JWT and injects user_id, org_id, and role
// into the request context. Context keys are defined in the auth package
// to avoid import cycles (handler reads them; middleware writes them).
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format")
				return
			}

			tokenStr := parts[1]
			if tokenStr == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing token")
				return
			}

			claims, err := auth.ValidateAccessToken(tokenStr, jwtSecret)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), auth.UserIDKey, claims.Subject)
			ctx = context.WithValue(ctx, auth.OrgIDKey, claims.OrgID)
			ctx = context.WithValue(ctx, auth.RoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
