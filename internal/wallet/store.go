package wallet

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	CreateWallet(ctx context.Context, input CreateWalletInput, now time.Time) (Wallet, error)
	ListWallets(ctx context.Context, userID string) ([]Wallet, error)
	UpdateWalletName(ctx context.Context, userID string, walletID string, name string, now time.Time) (Wallet, error)
	UpdateWalletOrder(ctx context.Context, userID string, items []WalletOrderItem, now time.Time) error
	AddWhitelist(ctx context.Context, input AddWhitelistInput, now time.Time) (WhitelistEntry, error)
	ListWhitelist(ctx context.Context, userID string) ([]WhitelistEntry, error)
	RecordSecurityEvent(ctx context.Context, input RecordSecurityEventInput, now time.Time) (SecurityEvent, error)
}

type MemoryStore struct {
	mu             sync.RWMutex
	wallets        map[string]Wallet
	whitelist      map[string]WhitelistEntry
	securityEvents map[string]SecurityEvent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		wallets:        map[string]Wallet{},
		whitelist:      map[string]WhitelistEntry{},
		securityEvents: map[string]SecurityEvent{},
	}
}

func (s *MemoryStore) CreateWallet(_ context.Context, input CreateWalletInput, now time.Time) (Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.wallets {
		if existing.UserID == input.UserID &&
			strings.EqualFold(existing.ChainType, input.ChainType) &&
			strings.EqualFold(existing.Address, input.Address) {
			return existing, nil
		}
	}
	wallet := Wallet{
		ID:              uuid.NewString(),
		UserID:          input.UserID,
		ChainType:       input.ChainType,
		Address:         input.Address,
		WalletType:      input.WalletType,
		TurnkeyOrgID:    input.TurnkeyOrgID,
		TurnkeyWalletID: input.TurnkeyWalletID,
		Name:            input.Name,
		SortOrder:       input.SortOrder,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	s.wallets[wallet.ID] = wallet
	return wallet, nil
}

func (s *MemoryStore) ListWallets(_ context.Context, userID string) ([]Wallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wallets := make([]Wallet, 0)
	for _, wallet := range s.wallets {
		if wallet.UserID == userID {
			wallets = append(wallets, wallet)
		}
	}
	sort.Slice(wallets, func(i, j int) bool {
		if wallets[i].SortOrder == wallets[j].SortOrder {
			return wallets[i].CreatedAt.Before(wallets[j].CreatedAt)
		}
		return wallets[i].SortOrder < wallets[j].SortOrder
	})
	return wallets, nil
}

func (s *MemoryStore) UpdateWalletName(_ context.Context, userID string, walletID string, name string, now time.Time) (Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wallet, ok := s.wallets[walletID]
	if !ok || wallet.UserID != userID {
		return Wallet{}, ErrNotFound
	}
	wallet.Name = name
	wallet.UpdatedAt = now
	s.wallets[walletID] = wallet
	return wallet, nil
}

func (s *MemoryStore) UpdateWalletOrder(_ context.Context, userID string, items []WalletOrderItem, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range items {
		wallet, ok := s.wallets[item.ID]
		if !ok || wallet.UserID != userID {
			return ErrNotFound
		}
		wallet.SortOrder = item.SortOrder
		wallet.UpdatedAt = now
		s.wallets[item.ID] = wallet
	}
	return nil
}

func (s *MemoryStore) AddWhitelist(_ context.Context, input AddWhitelistInput, now time.Time) (WhitelistEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.whitelist {
		if existing.UserID == input.UserID &&
			strings.EqualFold(existing.ChainType, input.ChainType) &&
			strings.EqualFold(existing.Address, input.Address) {
			return existing, nil
		}
	}
	entry := WhitelistEntry{
		ID:        uuid.NewString(),
		UserID:    input.UserID,
		ChainType: input.ChainType,
		Address:   input.Address,
		Label:     input.Label,
		CreatedAt: now,
	}
	s.whitelist[entry.ID] = entry
	return entry, nil
}

func (s *MemoryStore) ListWhitelist(_ context.Context, userID string) ([]WhitelistEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]WhitelistEntry, 0)
	for _, entry := range s.whitelist {
		if entry.UserID == userID {
			entries = append(entries, entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt.Before(entries[j].CreatedAt) })
	return entries, nil
}

func (s *MemoryStore) RecordSecurityEvent(_ context.Context, input RecordSecurityEventInput, now time.Time) (SecurityEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	riskLevel := input.RiskLevel
	if riskLevel == "" {
		riskLevel = "low"
	}
	event := SecurityEvent{
		ID:        uuid.NewString(),
		UserID:    input.UserID,
		WalletID:  input.WalletID,
		Action:    input.Action,
		RiskLevel: riskLevel,
		Metadata:  input.Metadata,
		CreatedAt: now,
	}
	s.securityEvents[event.ID] = event
	return event, nil
}
