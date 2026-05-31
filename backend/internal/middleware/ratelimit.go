package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stratahq/backend/internal/platform/response"
)

var incrementRateLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

func RateLimit(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := clientIP(r)
			key := fmt.Sprintf("ratelimit:%s", ip)
			ctx := r.Context()

			count, err := incrementRateLimit(ctx, rdb, key, window)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if count > int64(limit) {
				recordRateLimitBlocked("global")
				response.ErrorWithRequest(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// PerEndpointRateLimit creates a rate limiter scoped to a specific endpoint prefix.
// The key format is: ratelimit:{prefix}:{ip}
func PerEndpointRateLimit(rdb *redis.Client, prefix string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := clientIP(r)
			key := fmt.Sprintf("ratelimit:%s:%s", prefix, ip)
			ctx := r.Context()

			count, err := incrementRateLimit(ctx, rdb, key, window)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if count > int64(limit) {
				recordRateLimitBlocked(prefix)
				response.ErrorWithRequest(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func incrementRateLimit(ctx context.Context, rdb *redis.Client, key string, window time.Duration) (int64, error) {
	windowMillis := window.Milliseconds()
	if window > 0 && windowMillis == 0 {
		windowMillis = 1
	}
	return incrementRateLimitScript.Run(ctx, rdb, []string{key}, windowMillis).Int64()
}

func clientIP(r *http.Request) string {
	remoteHost := remoteHost(r.RemoteAddr)
	if isTrustedProxy(remoteHost) {
		if forwardedIP := forwardedHeaderIP(r.Header.Get("X-Forwarded-For")); forwardedIP != "" {
			return forwardedIP
		}
		if realIP := forwardedHeaderIP(r.Header.Get("X-Real-Ip")); realIP != "" {
			return realIP
		}
	}

	if remoteHost != "" {
		return remoteHost
	}

	return strings.TrimSpace(r.RemoteAddr)
}

func forwardedHeaderIP(value string) string {
	if value == "" {
		return ""
	}

	first := strings.TrimSpace(strings.Split(value, ",")[0])
	if first == "" {
		return ""
	}
	if ip := net.ParseIP(first); ip != nil {
		return ip.String()
	}

	host, _, err := net.SplitHostPort(first)
	if err != nil {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}

	return ""
}

func remoteHost(remoteAddr string) string {
	trimmed := strings.TrimSpace(remoteAddr)
	if trimmed == "" {
		return ""
	}

	host, _, err := net.SplitHostPort(trimmed)
	if err == nil {
		return strings.Trim(host, "[]")
	}
	if ip := net.ParseIP(strings.Trim(trimmed, "[]")); ip != nil {
		return ip.String()
	}
	return trimmed
}

var (
	trustedCIDRs     []*net.IPNet
	trustedCIDRsOnce sync.Once
)

func loadTrustedCIDRs() []*net.IPNet {
	trustedCIDRsOnce.Do(func() {
		cidrEnv := os.Getenv("TRUSTED_PROXY_CIDRS")
		if cidrEnv == "" {
			return
		}
		for _, s := range strings.Split(cidrEnv, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			_, cidr, err := net.ParseCIDR(s)
			if err != nil {
				continue
			}
			trustedCIDRs = append(trustedCIDRs, cidr)
		}
	})
	return trustedCIDRs
}

func isTrustedProxy(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return false
	}

	cidrs := loadTrustedCIDRs()
	if len(cidrs) > 0 {
		for _, cidr := range cidrs {
			if cidr.Contains(ip) {
				return true
			}
		}
		return false
	}

	return ip.IsLoopback()
}
