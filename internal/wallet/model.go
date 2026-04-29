package wallet

import "time"

type Wallet struct {
	ID                   string     `json:"id"`
	UserID               string     `json:"userId"`
	ChainType            string     `json:"chainType"`
	Address              string     `json:"address"`
	WalletType           string     `json:"walletType"`
	TurnkeyOrgID         string     `json:"turnkeyOrgId,omitempty"`
	TurnkeyWalletID      string     `json:"turnkeyWalletId,omitempty"`
	Name                 string     `json:"name,omitempty"`
	SortOrder            int        `json:"sortOrder"`
	ExportedPassphraseAt *time.Time `json:"exportedPassphraseAt,omitempty"`
	ExportedPrivateKeyAt *time.Time `json:"exportedPrivateKeyAt,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

type CreateWalletInput struct {
	UserID          string `json:"userId"`
	ChainType       string `json:"chainType"`
	Address         string `json:"address"`
	WalletType      string `json:"walletType"`
	TurnkeyOrgID    string `json:"turnkeyOrgId"`
	TurnkeyWalletID string `json:"turnkeyWalletId"`
	Name            string `json:"name"`
	SortOrder       int    `json:"sortOrder"`
}

type UpdateWalletNameInput struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
}

type WalletOrderItem struct {
	ID        string `json:"id"`
	SortOrder int    `json:"sortOrder"`
}

type UpdateWalletOrderInput struct {
	UserID string            `json:"userId"`
	Items  []WalletOrderItem `json:"items"`
}

type WhitelistEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	ChainType string    `json:"chainType"`
	Address   string    `json:"address"`
	Label     string    `json:"label,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type AddWhitelistInput struct {
	UserID    string `json:"userId"`
	ChainType string `json:"chainType"`
	Address   string `json:"address"`
	Label     string `json:"label"`
}

type SecurityEvent struct {
	ID        string         `json:"id"`
	UserID    string         `json:"userId"`
	WalletID  string         `json:"walletId,omitempty"`
	Action    string         `json:"action"`
	RiskLevel string         `json:"riskLevel"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

type RecordSecurityEventInput struct {
	UserID    string         `json:"userId"`
	WalletID  string         `json:"walletId"`
	Action    string         `json:"action"`
	RiskLevel string         `json:"riskLevel"`
	Metadata  map[string]any `json:"metadata"`
}
