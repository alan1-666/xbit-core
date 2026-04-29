package identity

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xbit/xbit-backend/pkg/auth"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	manager, err := auth.NewTokenManager(auth.TokenManagerConfig{
		SigningKey: "test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	return NewService(manager, nil)
}

func TestCreateNonceChallenge(t *testing.T) {
	svc := newTestService(t)
	challenge, err := svc.CreateNonceChallenge(context.Background(), "0xabc", "EVM")
	if err != nil {
		t.Fatal(err)
	}
	if challenge.Nonce == "" || challenge.Message == "" {
		t.Fatalf("expected challenge fields: %+v", challenge)
	}
	for _, want := range []string{"0xabc", "EVM", challenge.Nonce} {
		if !strings.Contains(challenge.Message, want) {
			t.Fatalf("message %q does not contain %q", challenge.Message, want)
		}
	}
}

func TestIssueDevLogin(t *testing.T) {
	svc := newTestService(t)
	pair, err := svc.IssueDevLogin(context.Background(), DevLoginInput{UserID: "user-1", DeviceID: "device-1"})
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected token pair: %+v", pair)
	}
}

func TestRefreshRotatesRefreshToken(t *testing.T) {
	svc := newTestService(t)
	pair, err := svc.IssueDevLogin(context.Background(), DevLoginInput{UserID: "user-1", DeviceID: "device-1"})
	if err != nil {
		t.Fatal(err)
	}

	next, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: pair.RefreshToken})
	if err != nil {
		t.Fatal(err)
	}
	if next.RefreshToken == "" || next.RefreshToken == pair.RefreshToken {
		t.Fatalf("expected rotated refresh token")
	}
	if _, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: pair.RefreshToken}); err == nil {
		t.Fatalf("old refresh token should not be reusable")
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	svc := newTestService(t)
	pair, err := svc.IssueDevLogin(context.Background(), DevLoginInput{UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(context.Background(), LogoutInput{RefreshToken: pair.RefreshToken}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: pair.RefreshToken}); err == nil {
		t.Fatalf("revoked refresh token should fail")
	}
}
