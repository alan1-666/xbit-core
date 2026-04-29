package identity

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) SaveNonce(ctx context.Context, challenge NonceChallenge) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO login_nonces (wallet_address, chain_type, nonce, message, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, challenge.WalletAddress, challenge.ChainType, challenge.Nonce, challenge.Message, challenge.ExpiresAt)
	if err != nil {
		return fmt.Errorf("save nonce: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpsertUser(ctx context.Context, userID string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (id)
		VALUES ($1)
		ON CONFLICT (id) DO UPDATE SET updated_at = now()
		RETURNING id
	`, userID).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("upsert user: %w", err)
	}
	return id, nil
}

func (s *PostgresStore) CreateSession(ctx context.Context, session Session) error {
	var ip any
	if session.IPAddress != "" {
		parsed := net.ParseIP(session.IPAddress)
		if parsed != nil {
			ip = parsed.String()
		}
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, refresh_token_hash, device_id, user_agent, ip_address, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, session.ID, session.UserID, session.RefreshTokenHash, session.DeviceID, session.UserAgent, ip, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *PostgresStore) FindActiveSessionByRefreshHash(ctx context.Context, refreshHash string, now time.Time) (Session, error) {
	var session Session
	var ip *net.IPNet
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, refresh_token_hash, device_id, user_agent, ip_address, expires_at, rotated_at, revoked_at, created_at
		FROM sessions
		WHERE refresh_token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > $2
	`, refreshHash, now).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.DeviceID,
		&session.UserAgent,
		&ip,
		&session.ExpiresAt,
		&session.RotatedAt,
		&session.RevokedAt,
		&session.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("find session: %w", err)
	}
	if ip != nil {
		session.IPAddress = ip.IP.String()
	}
	return session, nil
}

func (s *PostgresStore) RotateSession(ctx context.Context, sessionID string, newRefreshHash string, expiresAt time.Time, rotatedAt time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE sessions
		SET refresh_token_hash = $2, expires_at = $3, rotated_at = $4
		WHERE id = $1 AND revoked_at IS NULL
	`, sessionID, newRefreshHash, expiresAt, rotatedAt)
	if err != nil {
		return fmt.Errorf("rotate session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) RevokeSession(ctx context.Context, sessionID string, revokedAt time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`, sessionID, revokedAt)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
