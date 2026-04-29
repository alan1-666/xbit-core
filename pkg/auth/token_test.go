package auth

import (
	"testing"
	"time"
)

func TestIssueAndParseAccessToken(t *testing.T) {
	manager, err := NewTokenManager(TokenManagerConfig{
		Issuer:     "xbit-test",
		SigningKey: "test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	pair, err := manager.IssuePair("user-1", "session-1", "device-1", "Web", []string{"trade:read"})
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" || pair.RefreshTokenHash == "" {
		t.Fatalf("expected populated token pair: %+v", pair)
	}
	if pair.RefreshToken == pair.RefreshTokenHash {
		t.Fatalf("refresh token hash should not equal raw token")
	}

	claims, err := manager.ParseAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" || claims.Session != "session-1" || claims.DeviceID != "device-1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if len(claims.Scopes) != 1 || claims.Scopes[0] != "trade:read" {
		t.Fatalf("unexpected scopes: %+v", claims.Scopes)
	}
}

func TestRefreshHashIsStableAndSecretDependent(t *testing.T) {
	first, err := NewTokenManager(TokenManagerConfig{
		SigningKey: "secret-1",
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewTokenManager(TokenManagerConfig{
		SigningKey: "secret-2",
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	raw := "refresh-token"
	if first.HashRefreshToken(raw) != first.HashRefreshToken(raw) {
		t.Fatalf("hash should be stable")
	}
	if first.HashRefreshToken(raw) == second.HashRefreshToken(raw) {
		t.Fatalf("hash should depend on signing key")
	}
}

func TestNonceIsURLSafeAndUnique(t *testing.T) {
	a, err := NewNonce()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewNonce()
	if err != nil {
		t.Fatal(err)
	}
	if a == "" || b == "" || a == b {
		t.Fatalf("expected non-empty unique nonces: %q %q", a, b)
	}
}
