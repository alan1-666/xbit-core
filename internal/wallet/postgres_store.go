package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func (s *PostgresStore) CreateWallet(ctx context.Context, input CreateWalletInput, now time.Time) (Wallet, error) {
	var wallet Wallet
	err := s.pool.QueryRow(ctx, `
		INSERT INTO wallets (user_id, chain_type, address, wallet_type, turnkey_org_id, turnkey_wallet_id, name, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), $8, $9, $9)
		ON CONFLICT (user_id, chain_type, lower(address))
		DO UPDATE SET
			wallet_type = EXCLUDED.wallet_type,
			turnkey_org_id = EXCLUDED.turnkey_org_id,
			turnkey_wallet_id = EXCLUDED.turnkey_wallet_id,
			name = COALESCE(EXCLUDED.name, wallets.name),
			sort_order = EXCLUDED.sort_order,
			updated_at = EXCLUDED.updated_at
		RETURNING id, user_id, chain_type, address, wallet_type, COALESCE(turnkey_org_id, ''), COALESCE(turnkey_wallet_id, ''), COALESCE(name, ''), sort_order, exported_passphrase_at, exported_private_key_at, created_at, updated_at
	`, input.UserID, input.ChainType, input.Address, input.WalletType, input.TurnkeyOrgID, input.TurnkeyWalletID, input.Name, input.SortOrder, now).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.ChainType,
		&wallet.Address,
		&wallet.WalletType,
		&wallet.TurnkeyOrgID,
		&wallet.TurnkeyWalletID,
		&wallet.Name,
		&wallet.SortOrder,
		&wallet.ExportedPassphraseAt,
		&wallet.ExportedPrivateKeyAt,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	if err != nil {
		return Wallet{}, fmt.Errorf("create wallet: %w", err)
	}
	return wallet, nil
}

func (s *PostgresStore) ListWallets(ctx context.Context, userID string) ([]Wallet, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, chain_type, address, wallet_type, COALESCE(turnkey_org_id, ''), COALESCE(turnkey_wallet_id, ''), COALESCE(name, ''), sort_order, exported_passphrase_at, exported_private_key_at, created_at, updated_at
		FROM wallets
		WHERE user_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list wallets: %w", err)
	}
	defer rows.Close()

	wallets := make([]Wallet, 0)
	for rows.Next() {
		var wallet Wallet
		if err := rows.Scan(
			&wallet.ID,
			&wallet.UserID,
			&wallet.ChainType,
			&wallet.Address,
			&wallet.WalletType,
			&wallet.TurnkeyOrgID,
			&wallet.TurnkeyWalletID,
			&wallet.Name,
			&wallet.SortOrder,
			&wallet.ExportedPassphraseAt,
			&wallet.ExportedPrivateKeyAt,
			&wallet.CreatedAt,
			&wallet.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan wallet: %w", err)
		}
		wallets = append(wallets, wallet)
	}
	return wallets, rows.Err()
}

func (s *PostgresStore) UpdateWalletName(ctx context.Context, userID string, walletID string, name string, now time.Time) (Wallet, error) {
	var wallet Wallet
	err := s.pool.QueryRow(ctx, `
		UPDATE wallets
		SET name = $3, updated_at = $4
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, chain_type, address, wallet_type, COALESCE(turnkey_org_id, ''), COALESCE(turnkey_wallet_id, ''), COALESCE(name, ''), sort_order, exported_passphrase_at, exported_private_key_at, created_at, updated_at
	`, walletID, userID, name, now).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.ChainType,
		&wallet.Address,
		&wallet.WalletType,
		&wallet.TurnkeyOrgID,
		&wallet.TurnkeyWalletID,
		&wallet.Name,
		&wallet.SortOrder,
		&wallet.ExportedPassphraseAt,
		&wallet.ExportedPrivateKeyAt,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrNotFound
	}
	if err != nil {
		return Wallet{}, fmt.Errorf("update wallet name: %w", err)
	}
	return wallet, nil
}

func (s *PostgresStore) UpdateWalletOrder(ctx context.Context, userID string, items []WalletOrderItem, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update wallet order: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		tag, err := tx.Exec(ctx, `
			UPDATE wallets
			SET sort_order = $3, updated_at = $4
			WHERE id = $1 AND user_id = $2
		`, item.ID, userID, item.SortOrder, now)
		if err != nil {
			return fmt.Errorf("update wallet order: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
	}

	return tx.Commit(ctx)
}

func (s *PostgresStore) AddWhitelist(ctx context.Context, input AddWhitelistInput, now time.Time) (WhitelistEntry, error) {
	var entry WhitelistEntry
	err := s.pool.QueryRow(ctx, `
		INSERT INTO wallet_whitelist (user_id, chain_type, address, label, created_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5)
		ON CONFLICT (user_id, chain_type, lower(address))
		DO UPDATE SET label = COALESCE(EXCLUDED.label, wallet_whitelist.label)
		RETURNING id, user_id, chain_type, address, COALESCE(label, ''), created_at
	`, input.UserID, input.ChainType, input.Address, input.Label, now).Scan(
		&entry.ID,
		&entry.UserID,
		&entry.ChainType,
		&entry.Address,
		&entry.Label,
		&entry.CreatedAt,
	)
	if err != nil {
		return WhitelistEntry{}, fmt.Errorf("add whitelist: %w", err)
	}
	return entry, nil
}

func (s *PostgresStore) ListWhitelist(ctx context.Context, userID string) ([]WhitelistEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, chain_type, address, COALESCE(label, ''), created_at
		FROM wallet_whitelist
		WHERE user_id = $1
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list whitelist: %w", err)
	}
	defer rows.Close()

	entries := make([]WhitelistEntry, 0)
	for rows.Next() {
		var entry WhitelistEntry
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.ChainType, &entry.Address, &entry.Label, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan whitelist: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *PostgresStore) RecordSecurityEvent(ctx context.Context, input RecordSecurityEventInput, now time.Time) (SecurityEvent, error) {
	metadata := input.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	rawMetadata, err := json.Marshal(metadata)
	if err != nil {
		return SecurityEvent{}, fmt.Errorf("marshal security metadata: %w", err)
	}
	riskLevel := input.RiskLevel
	if riskLevel == "" {
		riskLevel = "low"
	}

	var event SecurityEvent
	var walletID *string
	err = s.pool.QueryRow(ctx, `
		INSERT INTO wallet_security_events (user_id, wallet_id, action, risk_level, metadata, created_at)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6)
		RETURNING id, user_id, wallet_id, action, risk_level, metadata, created_at
	`, input.UserID, input.WalletID, input.Action, riskLevel, rawMetadata, now).Scan(
		&event.ID,
		&event.UserID,
		&walletID,
		&event.Action,
		&event.RiskLevel,
		&rawMetadata,
		&event.CreatedAt,
	)
	if err != nil {
		return SecurityEvent{}, fmt.Errorf("record security event: %w", err)
	}
	if walletID != nil {
		event.WalletID = *walletID
	}
	_ = json.Unmarshal(rawMetadata, &event.Metadata)
	return event, nil
}
