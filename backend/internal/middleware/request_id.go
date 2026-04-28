package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := sanitizeRequestID(r.Header.Get(RequestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		w.Header().Set(RequestIDHeader, requestID)
		r.Header.Set(RequestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(requestIDContextKey{}).(string)
	return value
}

func sanitizeRequestID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 || strings.ContainsAny(value, "\r\n\t ") {
		return ""
	}
	return value
}
