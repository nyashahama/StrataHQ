package middleware

import (
	"net/http"
	"strings"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/stratahq/backend/internal/billing"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

func Entitlement(svc *billing.Service) func(http.Handler) http.Handler {
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

			subscription, err := svc.GetSubscription(r.Context(), identity)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, billing.ErrForbidden) {
					response.Error(w, http.StatusForbidden, response.CodeForbidden, "subscription required")
					return
				}
				response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to check subscription")
				return
			}

			if !subscription.EntitlementActive {
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
