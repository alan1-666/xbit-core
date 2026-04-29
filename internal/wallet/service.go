package wallet

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{store: store, now: time.Now}
}

func (s *Service) CreateWallet(ctx context.Context, input CreateWalletInput) (Wallet, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.ChainType = strings.TrimSpace(input.ChainType)
	input.Address = strings.TrimSpace(input.Address)
	input.WalletType = strings.TrimSpace(input.WalletType)
	input.Name = strings.TrimSpace(input.Name)
	if input.UserID == "" {
		return Wallet{}, fmt.Errorf("user id is required")
	}
	if input.ChainType == "" {
		return Wallet{}, fmt.Errorf("chain type is required")
	}
	if input.Address == "" {
		return Wallet{}, fmt.Errorf("address is required")
	}
	if input.WalletType == "" {
		input.WalletType = "embedded"
	}
	return s.store.CreateWallet(ctx, input, s.now().UTC())
}

func (s *Service) ListWallets(ctx context.Context, userID string) ([]Wallet, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}
	return s.store.ListWallets(ctx, userID)
}

func (s *Service) UpdateWalletName(ctx context.Context, userID string, walletID string, name string) (Wallet, error) {
	userID = strings.TrimSpace(userID)
	walletID = strings.TrimSpace(walletID)
	name = strings.TrimSpace(name)
	if userID == "" || walletID == "" {
		return Wallet{}, fmt.Errorf("user id and wallet id are required")
	}
	return s.store.UpdateWalletName(ctx, userID, walletID, name, s.now().UTC())
}

func (s *Service) UpdateWalletOrder(ctx context.Context, input UpdateWalletOrderInput) error {
	input.UserID = strings.TrimSpace(input.UserID)
	if input.UserID == "" {
		return fmt.Errorf("user id is required")
	}
	if len(input.Items) == 0 {
		return fmt.Errorf("items are required")
	}
	for _, item := range input.Items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("wallet id is required")
		}
	}
	return s.store.UpdateWalletOrder(ctx, input.UserID, input.Items, s.now().UTC())
}

func (s *Service) AddWhitelist(ctx context.Context, input AddWhitelistInput) (WhitelistEntry, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.ChainType = strings.TrimSpace(input.ChainType)
	input.Address = strings.TrimSpace(input.Address)
	input.Label = strings.TrimSpace(input.Label)
	if input.UserID == "" || input.ChainType == "" || input.Address == "" {
		return WhitelistEntry{}, fmt.Errorf("user id, chain type and address are required")
	}
	return s.store.AddWhitelist(ctx, input, s.now().UTC())
}

func (s *Service) ListWhitelist(ctx context.Context, userID string) ([]WhitelistEntry, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}
	return s.store.ListWhitelist(ctx, userID)
}

func (s *Service) RecordSecurityEvent(ctx context.Context, input RecordSecurityEventInput) (SecurityEvent, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.WalletID = strings.TrimSpace(input.WalletID)
	input.Action = strings.TrimSpace(input.Action)
	input.RiskLevel = strings.TrimSpace(input.RiskLevel)
	if input.UserID == "" || input.Action == "" {
		return SecurityEvent{}, fmt.Errorf("user id and action are required")
	}
	return s.store.RecordSecurityEvent(ctx, input, s.now().UTC())
}
