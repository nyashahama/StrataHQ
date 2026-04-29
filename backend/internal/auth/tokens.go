package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ContextKey is the type for auth-related context keys.
// Defined here (not in middleware) to avoid import cycles.
type ContextKey string

const (
	UserIDKey ContextKey = "user_id"
	OrgIDKey  ContextKey = "org_id"
	RoleKey   ContextKey = "role"
)

// Claims holds JWT payload fields beyond the registered set.
type Claims struct {
	OrgID string `json:"org_id"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a signed HS256 JWT for the given user.
func GenerateAccessToken(userID, orgID, role, issuer, audience, secret string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		OrgID: orgID,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken returns a 32-byte cryptographically random hex string.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ValidateAccessToken parses and validates a JWT, returning its claims.
func ValidateAccessToken(tokenStr, secret string, issuer, audience string) (*Claims, error) {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"HS256"}),
	}
	if issuer != "" {
		opts = append(opts, jwt.WithIssuer(issuer))
	}
	if audience != "" {
		opts = append(opts, jwt.WithAudience(audience))
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	}, opts...)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
