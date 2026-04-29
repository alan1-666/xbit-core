package hypertrader

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type AgentSignerConfig struct {
	Enabled          bool
	Mode             string
	HyperliquidChain string
	SignatureChainID string
	AgentNamePrefix  string
	DefaultPolicy    AgentPolicy
	Now              func() time.Time
}

type AgentSigner struct {
	store AgentStore
	cfg   AgentSignerConfig
	now   func() time.Time
}

func NewAgentSigner(store AgentStore, cfg AgentSignerConfig) *AgentSigner {
	if store == nil || !cfg.Enabled {
		return nil
	}
	if cfg.Mode == "" {
		cfg.Mode = "dev"
	}
	if cfg.HyperliquidChain == "" {
		cfg.HyperliquidChain = "Mainnet"
	}
	if cfg.SignatureChainID == "" {
		cfg.SignatureChainID = "0xa4b1"
	}
	if cfg.AgentNamePrefix == "" {
		cfg.AgentNamePrefix = "XBIT Agent"
	}
	if cfg.DefaultPolicy.MaxLeverage <= 0 {
		cfg.DefaultPolicy.MaxLeverage = 20
	}
	if len(cfg.DefaultPolicy.AllowedActions) == 0 {
		cfg.DefaultPolicy.AllowedActions = []string{"order", "cancel", "updateLeverage"}
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &AgentSigner{store: store, cfg: cfg, now: now}
}

func (s *AgentSigner) Enabled() bool {
	return s != nil && s.cfg.Enabled
}

func (s *AgentSigner) CreateWallet(ctx context.Context, input CreateAgentWalletInput) (AgentApproval, error) {
	if !s.Enabled() {
		return AgentApproval{}, fmt.Errorf("agent signer is disabled")
	}
	userAddress := strings.ToLower(strings.TrimSpace(input.UserAddress))
	if userAddress == "" {
		return AgentApproval{}, fmt.Errorf("userAddress is required")
	}
	key, err := randomHex(32)
	if err != nil {
		return AgentApproval{}, err
	}
	address := agentAddressFromKey(key)
	now := s.now().UTC()
	policy := mergeAgentPolicy(s.cfg.DefaultPolicy, input.Policy)
	name := strings.TrimSpace(input.AgentName)
	if name == "" {
		name = s.cfg.AgentNamePrefix
	}
	wallet, err := s.store.SaveAgentWallet(ctx, AgentWallet{
		UserID:       strings.TrimSpace(input.UserID),
		UserAddress:  userAddress,
		AgentAddress: address,
		AgentName:    name,
		Status:       "pending_approval",
		KeyRef:       "dev:" + key,
		Policy:       policy,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return AgentApproval{}, err
	}
	nonce := now.UnixMilli()
	chain := firstNonEmpty(strings.TrimSpace(input.HyperliquidChain), s.cfg.HyperliquidChain)
	signatureChainID := firstNonEmpty(strings.TrimSpace(input.SignatureChainID), s.cfg.SignatureChainID)
	action := map[string]any{
		"type":             "approveAgent",
		"hyperliquidChain": chain,
		"signatureChainId": signatureChainID,
		"agentAddress":     wallet.AgentAddress,
		"agentName":        wallet.AgentName,
		"nonce":            nonce,
	}
	return AgentApproval{
		Wallet: wallet,
		ApprovalPayload: map[string]any{
			"action": action,
			"nonce":  nonce,
			"signer": userAddress,
			"status": "requires_user_signature",
		},
	}, nil
}

func (s *AgentSigner) ActivateWallet(ctx context.Context, input ActivateAgentWalletInput) (AgentWallet, error) {
	if !s.Enabled() {
		return AgentWallet{}, fmt.Errorf("agent signer is disabled")
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "disabled" && status != "pending_approval" {
		return AgentWallet{}, fmt.Errorf("unsupported agent wallet status %q", status)
	}
	return s.store.UpdateAgentWalletStatus(ctx, input.UserAddress, input.AgentAddress, status)
}

func (s *AgentSigner) ListWallets(ctx context.Context, userAddress string) ([]AgentWallet, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("agent signer is disabled")
	}
	return s.store.ListAgentWallets(ctx, userAddress)
}

func (s *AgentSigner) Sign(ctx context.Context, input AgentSignInput) (AgentSignedPayload, error) {
	if !s.Enabled() {
		return AgentSignedPayload{}, fmt.Errorf("agent signer is disabled")
	}
	userAddress := strings.ToLower(strings.TrimSpace(input.UserAddress))
	if userAddress == "" {
		return AgentSignedPayload{}, fmt.Errorf("userAddress is required for agent signing")
	}
	wallet, err := s.store.GetActiveAgentWallet(ctx, userAddress)
	if err != nil {
		return AgentSignedPayload{}, fmt.Errorf("active agent wallet is required: %w", err)
	}
	action := input.ExchangeAction
	if len(action) == 0 {
		action = asMap(input.Payload["exchangeAction"])
	}
	if len(action) == 0 {
		action = asMap(input.Payload["action"])
	}
	if len(action) == 0 {
		return AgentSignedPayload{}, fmt.Errorf("exchangeAction is required for agent signing")
	}
	actionType := firstNonEmpty(stringFromAny(action["type"]), input.Action)
	if err := validateAgentPolicy(wallet.Policy, actionType, input.Symbol, input.Leverage); err != nil {
		return AgentSignedPayload{}, err
	}
	nonce, err := s.store.NextAgentNonce(ctx, wallet.AgentAddress, s.now().UTC())
	if err != nil {
		return AgentSignedPayload{}, err
	}
	payload := map[string]any{
		"action":    action,
		"nonce":     nonce,
		"signature": s.sign(wallet, actionType, action, nonce),
	}
	if input.VaultAddress != "" {
		payload["vaultAddress"] = strings.ToLower(strings.TrimSpace(input.VaultAddress))
	}
	if input.ExpiresAfter > 0 {
		payload["expiresAfter"] = input.ExpiresAfter
	}
	signature := payload["signature"].(Signature)
	return AgentSignedPayload{
		AgentWallet:     wallet,
		ExchangePayload: payload,
		Signature:       signature,
		Nonce:           nonce,
		Action:          actionType,
		Status:          "signed",
	}, nil
}

func (s *AgentSigner) sign(wallet AgentWallet, actionType string, action map[string]any, nonce int64) Signature {
	body, _ := json.Marshal(map[string]any{
		"mode":         s.cfg.Mode,
		"keyRef":       wallet.KeyRef,
		"agentAddress": wallet.AgentAddress,
		"actionType":   actionType,
		"action":       action,
		"nonce":        nonce,
	})
	r := sha256.Sum256(append([]byte("r:"), body...))
	sigS := sha256.Sum256(append([]byte("s:"), body...))
	return Signature{
		R: "0x" + hex.EncodeToString(r[:]),
		S: "0x" + hex.EncodeToString(sigS[:]),
		V: 27,
	}
}

func validateAgentPolicy(policy AgentPolicy, action string, symbol string, leverage int) error {
	action = normalizeAgentAction(action)
	if action == "" {
		return fmt.Errorf("action is required for agent signing")
	}
	if !stringAllowed(policy.AllowedActions, action) {
		return fmt.Errorf("agent action %q is not allowed", action)
	}
	if symbol != "" && len(policy.AllowedSymbols) > 0 && !stringAllowed(policy.AllowedSymbols, strings.ToUpper(symbol)) {
		return fmt.Errorf("symbol %q is not allowed for agent wallet", symbol)
	}
	if leverage > 0 && policy.MaxLeverage > 0 && leverage > policy.MaxLeverage {
		return fmt.Errorf("leverage %d exceeds agent max leverage %d", leverage, policy.MaxLeverage)
	}
	return nil
}

func mergeAgentPolicy(base AgentPolicy, override AgentPolicy) AgentPolicy {
	out := base
	if len(override.AllowedActions) > 0 {
		out.AllowedActions = override.AllowedActions
	}
	if len(override.AllowedSymbols) > 0 {
		out.AllowedSymbols = override.AllowedSymbols
	}
	if override.MaxLeverage > 0 {
		out.MaxLeverage = override.MaxLeverage
	}
	return out
}

func normalizeAgentAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "signhyperliquidcreateorder", "createhyperliquidorder", "submithyperliquidorder", "submitorder":
		return "order"
	case "signhyperliquidcancelorder", "cancelhyperliquidorder", "cancelorder":
		return "cancel"
	case "signhyperliquidupdateleverage", "updatehyperliquidleverage":
		return "updateLeverage"
	default:
		return strings.TrimSpace(action)
	}
}

func stringAllowed(values []string, target string) bool {
	target = normalizeAgentAction(target)
	for _, value := range values {
		if strings.EqualFold(normalizeAgentAction(value), target) {
			return true
		}
	}
	return false
}

func randomHex(size int) (string, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate agent key: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

func agentAddressFromKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return "0x" + hex.EncodeToString(sum[len(sum)-20:])
}
