package auth

import "testing"

func TestHashRefreshToken_Deterministic(t *testing.T) {
	token := "abc123"
	h1 := HashRefreshToken(token)
	h2 := HashRefreshToken(token)
	if h1 != h2 {
		t.Error("hash should be deterministic")
	}
}

func TestHashRefreshToken_DifferentTokens(t *testing.T) {
	h1 := HashRefreshToken("token1")
	h2 := HashRefreshToken("token2")
	if h1 == h2 {
		t.Error("different tokens should produce different hashes")
	}
}

func TestHashRefreshToken_NotPlaintext(t *testing.T) {
	token := "super-secret-token"
	h := HashRefreshToken(token)
	if h == token {
		t.Error("hash should not equal plaintext token")
	}
}
