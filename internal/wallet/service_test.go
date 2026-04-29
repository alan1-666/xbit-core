package wallet

import (
	"context"
	"testing"
)

func TestWalletLifecycle(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()

	created, err := svc.CreateWallet(ctx, CreateWalletInput{
		UserID:     "user-1",
		ChainType:  "EVM",
		Address:    "0xabc",
		WalletType: "embedded",
		Name:       "Main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatalf("wallet id missing")
	}

	wallets, err := svc.ListWallets(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(wallets) != 1 {
		t.Fatalf("wallets = %+v", wallets)
	}

	updated, err := svc.UpdateWalletName(ctx, "user-1", created.ID, "Trading")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Trading" {
		t.Fatalf("name = %s", updated.Name)
	}

	if err := svc.UpdateWalletOrder(ctx, UpdateWalletOrderInput{
		UserID: "user-1",
		Items:  []WalletOrderItem{{ID: created.ID, SortOrder: 7}},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestWhitelistAndSecurityEvent(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()

	entry, err := svc.AddWhitelist(ctx, AddWhitelistInput{
		UserID:    "user-1",
		ChainType: "EVM",
		Address:   "0xdef",
		Label:     "cold",
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID == "" {
		t.Fatalf("entry id missing")
	}

	entries, err := svc.ListWhitelist(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %+v", entries)
	}

	event, err := svc.RecordSecurityEvent(ctx, RecordSecurityEventInput{
		UserID:    "user-1",
		Action:    "wallet_created",
		RiskLevel: "low",
		Metadata:  map[string]any{"wallet": entry.Address},
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.ID == "" || event.Action != "wallet_created" {
		t.Fatalf("event = %+v", event)
	}
}
