package identity

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrNotFound = errors.New("not found")

type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	DeviceID         string
	UserAgent        string
	IPAddress        string
	ExpiresAt        time.Time
	RotatedAt        *time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
}

type Store interface {
	SaveNonce(ctx context.Context, challenge NonceChallenge) error
	UpsertUser(ctx context.Context, userID string) (string, error)
	CreateSession(ctx context.Context, session Session) error
	FindActiveSessionByRefreshHash(ctx context.Context, refreshHash string, now time.Time) (Session, error)
	RotateSession(ctx context.Context, sessionID string, newRefreshHash string, expiresAt time.Time, rotatedAt time.Time) error
	RevokeSession(ctx context.Context, sessionID string, revokedAt time.Time) error
}

type MemoryStore struct {
	mu       sync.RWMutex
	users    map[string]struct{}
	nonces   map[string]NonceChallenge
	sessions map[string]Session
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:    map[string]struct{}{},
		nonces:   map[string]NonceChallenge{},
		sessions: map[string]Session{},
	}
}

func (s *MemoryStore) SaveNonce(_ context.Context, challenge NonceChallenge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonces[challenge.Nonce] = challenge
	return nil
}

func (s *MemoryStore) UpsertUser(_ context.Context, userID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[userID] = struct{}{}
	return userID, nil
}

func (s *MemoryStore) CreateSession(_ context.Context, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *MemoryStore) FindActiveSessionByRefreshHash(_ context.Context, refreshHash string, now time.Time) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, session := range s.sessions {
		if session.RefreshTokenHash != refreshHash {
			continue
		}
		if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
			return Session{}, ErrNotFound
		}
		return session, nil
	}
	return Session{}, ErrNotFound
}

func (s *MemoryStore) RotateSession(_ context.Context, sessionID string, newRefreshHash string, expiresAt time.Time, rotatedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return ErrNotFound
	}
	session.RefreshTokenHash = newRefreshHash
	session.ExpiresAt = expiresAt
	session.RotatedAt = &rotatedAt
	s.sessions[sessionID] = session
	return nil
}

func (s *MemoryStore) RevokeSession(_ context.Context, sessionID string, revokedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return ErrNotFound
	}
	session.RevokedAt = &revokedAt
	s.sessions[sessionID] = session
	return nil
}
