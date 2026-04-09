package middleware

import (
	"net/http"
	"strings"
)

const (
	contentSecurityPolicy = "default-src 'self'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'"
	permissionsPolicy     = "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()"
	hstsPolicy            = "max-age=63072000; includeSubDomains; preload"
)

func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := w.Header()
			headers.Set("Content-Security-Policy", contentSecurityPolicy)
			headers.Set("Permissions-Policy", permissionsPolicy)
			headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-Frame-Options", "DENY")

			if requestUsesHTTPS(r) {
				headers.Set("Strict-Transport-Security", hstsPolicy)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func requestUsesHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}
