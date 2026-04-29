package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenManager struct {
	issuer     string
	signingKey []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

type TokenManagerConfig struct {
	Issuer     string
	SigningKey string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type AccessClaims struct {
	UserID   string   `json:"uid"`
	DeviceID string   `json:"did,omitempty"`
	Session  string   `json:"sid"`
	Platform string   `json:"p,omitempty"`
	Scopes   []string `json:"scope,omitempty"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken      string    `json:"accessToken"`
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshToken     string    `json:"refreshToken"`
	RefreshTokenHash string    `json:"-"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
	TokenType        string    `json:"tokenType"`
}

func NewTokenManager(cfg TokenManagerConfig) (*TokenManager, error) {
	if cfg.SigningKey == "" {
		return nil, fmt.Errorf("signing key is required")
	}
	if cfg.AccessTTL <= 0 {
		return nil, fmt.Errorf("access ttl must be positive")
	}
	if cfg.RefreshTTL <= 0 {
		return nil, fmt.Errorf("refresh ttl must be positive")
	}
	issuer := cfg.Issuer
	if issuer == "" {
		issuer = "xbit"
	}
	return &TokenManager{
		issuer:     issuer,
		signingKey: []byte(cfg.SigningKey),
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		now:        time.Now,
	}, nil
}

func (m *TokenManager) IssuePair(userID string, sessionID string, deviceID string, platform string, scopes []string) (TokenPair, error) {
	if userID == "" {
		return TokenPair{}, fmt.Errorf("user id is required")
	}
	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	now := m.now().UTC()
	accessExpiresAt := now.Add(m.accessTTL)
	refreshExpiresAt := now.Add(m.refreshTTL)
	claims := AccessClaims{
		UserID:   userID,
		DeviceID: deviceID,
		Session:  sessionID,
		Platform: platform,
		Scopes:   scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.signingKey)
	if err != nil {
		return TokenPair{}, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken, err := NewOpaqueToken(32)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshTokenHash: m.HashRefreshToken(refreshToken),
		RefreshExpiresAt: refreshExpiresAt,
		TokenType:        "Bearer",
	}, nil
}

func (m *TokenManager) ParseAccessToken(tokenString string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return m.signingKey, nil
	}, jwt.WithIssuer(m.issuer))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (m *TokenManager) HashRefreshToken(refreshToken string) string {
	mac := hmac.New(sha256.New, m.signingKey)
	_, _ = mac.Write([]byte(refreshToken))
	return hex.EncodeToString(mac.Sum(nil))
}

func NewOpaqueToken(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("token size must be positive")
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func NewNonce() (string, error) {
	return NewOpaqueToken(24)
}
