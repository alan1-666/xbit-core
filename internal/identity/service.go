package identity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/xbit/xbit-backend/pkg/auth"
)

type Service struct {
	tokens *auth.TokenManager
	store  Store
	now    func() time.Time
}

type NonceChallenge struct {
	WalletAddress string    `json:"walletAddress"`
	ChainType     string    `json:"chainType"`
	Nonce         string    `json:"nonce"`
	Message       string    `json:"message"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

type DevLoginInput struct {
	UserID    string   `json:"userId"`
	DeviceID  string   `json:"deviceId"`
	Platform  string   `json:"platform"`
	Scopes    []string `json:"scopes"`
	UserAgent string   `json:"-"`
	IPAddress string   `json:"-"`
}

type RefreshInput struct {
	RefreshToken string `json:"refreshToken"`
	DeviceID     string `json:"deviceId"`
	Platform     string `json:"platform"`
}

type LogoutInput struct {
	RefreshToken string `json:"refreshToken"`
}

func NewService(tokens *auth.TokenManager, store Store) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{
		tokens: tokens,
		store:  store,
		now:    time.Now,
	}
}

func (s *Service) CreateNonceChallenge(ctx context.Context, walletAddress string, chainType string) (NonceChallenge, error) {
	walletAddress = strings.TrimSpace(walletAddress)
	chainType = strings.TrimSpace(chainType)
	if walletAddress == "" {
		return NonceChallenge{}, fmt.Errorf("wallet address is required")
	}
	if chainType == "" {
		return NonceChallenge{}, fmt.Errorf("chain type is required")
	}

	nonce, err := auth.NewNonce()
	if err != nil {
		return NonceChallenge{}, err
	}
	expiresAt := s.now().UTC().Add(10 * time.Minute)
	message := BuildWalletLoginMessage(walletAddress, chainType, nonce, expiresAt)

	challenge := NonceChallenge{
		WalletAddress: walletAddress,
		ChainType:     chainType,
		Nonce:         nonce,
		Message:       message,
		ExpiresAt:     expiresAt,
	}
	if err := s.store.SaveNonce(ctx, challenge); err != nil {
		return NonceChallenge{}, err
	}
	return challenge, nil
}

func (s *Service) IssueDevLogin(ctx context.Context, input DevLoginInput) (auth.TokenPair, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		userID = uuid.NewString()
	}
	if _, err := s.store.UpsertUser(ctx, userID); err != nil {
		return auth.TokenPair{}, err
	}

	platform := strings.TrimSpace(input.Platform)
	if platform == "" {
		platform = "Web"
	}
	sessionID := uuid.NewString()
	pair, err := s.tokens.IssuePair(userID, sessionID, strings.TrimSpace(input.DeviceID), platform, input.Scopes)
	if err != nil {
		return auth.TokenPair{}, err
	}

	now := s.now().UTC()
	if err := s.store.CreateSession(ctx, Session{
		ID:               sessionID,
		UserID:           userID,
		RefreshTokenHash: pair.RefreshTokenHash,
		DeviceID:         strings.TrimSpace(input.DeviceID),
		UserAgent:        strings.TrimSpace(input.UserAgent),
		IPAddress:        strings.TrimSpace(input.IPAddress),
		ExpiresAt:        pair.RefreshExpiresAt,
		CreatedAt:        now,
	}); err != nil {
		return auth.TokenPair{}, err
	}

	return pair, nil
}

func (s *Service) Refresh(ctx context.Context, input RefreshInput) (auth.TokenPair, error) {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return auth.TokenPair{}, fmt.Errorf("refresh token is required")
	}
	now := s.now().UTC()
	session, err := s.store.FindActiveSessionByRefreshHash(ctx, s.tokens.HashRefreshToken(refreshToken), now)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("invalid refresh token")
	}
	platform := strings.TrimSpace(input.Platform)
	if platform == "" {
		platform = "Web"
	}
	deviceID := strings.TrimSpace(input.DeviceID)
	if deviceID == "" {
		deviceID = session.DeviceID
	}
	pair, err := s.tokens.IssuePair(session.UserID, session.ID, deviceID, platform, nil)
	if err != nil {
		return auth.TokenPair{}, err
	}
	if err := s.store.RotateSession(ctx, session.ID, pair.RefreshTokenHash, pair.RefreshExpiresAt, now); err != nil {
		return auth.TokenPair{}, err
	}
	return pair, nil
}

func (s *Service) Logout(ctx context.Context, input LogoutInput) error {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return fmt.Errorf("refresh token is required")
	}
	now := s.now().UTC()
	session, err := s.store.FindActiveSessionByRefreshHash(ctx, s.tokens.HashRefreshToken(refreshToken), now)
	if err != nil {
		return nil
	}
	return s.store.RevokeSession(ctx, session.ID, now)
}

func (s *Service) VerifyAccessToken(token string) (*auth.AccessClaims, error) {
	return s.tokens.ParseAccessToken(strings.TrimSpace(token))
}

func BuildWalletLoginMessage(walletAddress string, chainType string, nonce string, expiresAt time.Time) string {
	return fmt.Sprintf(
		"KairoX wants you to sign in with your %s wallet:\n%s\n\nNonce: %s\nExpires At: %s",
		chainType,
		walletAddress,
		nonce,
		expiresAt.UTC().Format(time.RFC3339),
	)
}
