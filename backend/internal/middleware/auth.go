package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/stratahq/backend/internal/platform/response"
)

type contextKey string

const UserIDKey contextKey = "user_id"

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

			token := parts[1]
			if token == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing token")
				return
			}

			// TODO: JWT validation will be implemented when auth domain is built.
			// For now, this middleware validates the header format only.
			ctx := context.WithValue(r.Context(), UserIDKey, "placeholder")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
