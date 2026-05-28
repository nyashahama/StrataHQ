package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/billing"
	"github.com/stratahq/backend/internal/platform/response"
)

func Entitlement(svc *billing.Service, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			if isExemptFromEntitlement(path) {
				next.ServeHTTP(w, r)
				return
			}

			identity, ok := auth.IdentityFromRequest(r)
			if !ok {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
				return
			}

			active, err := svc.HasActiveEntitlement(r.Context(), identity)
			if err != nil {
				logger.Error("entitlement check failed", "error", err)
				response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to check subscription")
				return
			}

			if !active {
				response.Error(w, http.StatusForbidden, response.CodeForbidden, "active subscription required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isExemptFromEntitlement(path string) bool {
	exemptPrefixes := []string{
		"/api/v1/auth/",
		"/api/v1/billing/",
		"/api/v1/onboarding",
	}
	for _, prefix := range exemptPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
