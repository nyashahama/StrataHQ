package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/stratahq/backend/internal/audit"
	"github.com/stratahq/backend/internal/auth"
)

func AuditEvents(recorder audit.Recorder, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		if recorder == nil {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !shouldAuditRequest(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			wrapped := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			event := audit.Event{
				Method:       r.Method,
				Path:         r.URL.Path,
				RoutePattern: routePattern(r),
				IPAddress:    clientIP(r),
				UserAgent:    r.UserAgent(),
				OccurredAtNS: time.Now().UTC().UnixNano(),
				StatusCode:   wrapped.statusCode,
			}

			if identity, ok := auth.IdentityFromRequest(r); ok {
				event.ActorUserID = identity.UserID
				event.OrgID = identity.OrgID
				event.ActorRole = identity.Role
			}

			if err := recordAuditEvent(context.WithoutCancel(r.Context()), recorder, event); err != nil {
				logger.Error("record audit event",
					"error", err,
					"method", event.Method,
					"path", event.Path,
					"status", event.StatusCode,
					"request_id", RequestIDFromContext(r.Context()),
				)
			}
		})
	}
}

func recordAuditEvent(ctx context.Context, recorder audit.Recorder, event audit.Event) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = recorder.Record(ctx, event)
		if err == nil {
			return nil
		}
	}
	return err
}

func shouldAuditRequest(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func routePattern(r *http.Request) string {
	if r == nil {
		return ""
	}
	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
		return routeCtx.RoutePattern()
	}
	return ""
}
