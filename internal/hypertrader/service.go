package hypertrader

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	store       Store
	provider    Provider
	agentSigner *AgentSigner
	now         func() time.Time
}

func NewService(store Store) *Service {
	return NewServiceWithProvider(store, nil)
}

func NewServiceWithProvider(store Store, provider Provider) *Service {
	return NewServiceWithProviderAndSigner(store, provider, nil)
}

func NewServiceWithProviderAndSigner(store Store, provider Provider, agentSigner *AgentSigner) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	if provider == nil {
		provider = NewLocalProvider()
	}
	return &Service{store: store, provider: provider, agentSigner: agentSigner, now: time.Now}
}

func (s *Service) ListSymbols(ctx context.Context, query string, category string, limit int) ([]Symbol, error) {
	return s.store.ListSymbols(ctx, query, category, limit)
}

func (s *Service) GetSymbolPreference(ctx context.Context, userID string, symbol string) (SymbolPreference, error) {
	return s.store.GetPreference(ctx, userID, symbol)
}

func (s *Service) UpdateSymbolPreference(ctx context.Context, pref SymbolPreference) (SymbolPreference, error) {
	if strings.TrimSpace(pref.Symbol) == "" {
		return SymbolPreference{}, fmt.Errorf("symbol is required")
	}
	return s.store.SavePreference(ctx, pref)
}

func (s *Service) Account(ctx context.Context, userAddress string) (AccountBalance, error) {
	if strings.TrimSpace(userAddress) != "" {
		account, err := s.provider.Account(ctx, userAddress)
		if err == nil && (account.Balance != "" || len(account.Positions) > 0) {
			if state, ok := s.store.(StateStore); ok {
				_ = state.SaveAccountSnapshot(ctx, userAddress, account)
			}
			return account, nil
		}
		if state, ok := s.store.(StateStore); ok {
			if snapshot, snapshotErr := state.GetAccountSnapshot(ctx, userAddress); snapshotErr == nil {
				return snapshot, nil
			}
		}
	}
	symbols, err := s.store.ListSymbols(ctx, "", "", 2)
	if err != nil {
		return AccountBalance{}, err
	}
	positions := make([]Position, 0, len(symbols))
	for i, symbol := range symbols {
		side := "long"
		if i%2 == 1 {
			side = "short"
		}
		positions = append(positions, seedPosition(userAddress, symbol, side, s.now().UTC()))
	}
	return AccountBalance{
		Balance:             "250000",
		OneDayChange:        "2180",
		OneDayPercentChange: "0.87",
		RawUSD:              "250000",
		Positions:           positions,
	}, nil
}

func (s *Service) TradeHistory(ctx context.Context, userAddress string, limit int) ([]TradeHistory, error) {
	trades, err := s.provider.TradeHistory(ctx, userAddress, limit)
	if err == nil {
		if state, ok := s.store.(StateStore); ok {
			_ = state.AppendFills(ctx, userAddress, trades)
		}
		return trades, nil
	}
	if state, ok := s.store.(StateStore); ok {
		if snapshot, snapshotErr := state.ListFills(ctx, userAddress, limit); snapshotErr == nil && len(snapshot) > 0 {
			return snapshot, nil
		}
	}
	return nil, err
}

func (s *Service) OpenOrders(ctx context.Context, userAddress string) ([]OpenOrder, error) {
	orders, err := s.provider.OpenOrders(ctx, userAddress)
	if err == nil {
		if state, ok := s.store.(StateStore); ok {
			_ = state.SaveOpenOrdersSnapshot(ctx, userAddress, orders)
		}
		return orders, nil
	}
	if state, ok := s.store.(StateStore); ok {
		if snapshot, snapshotErr := state.ListOpenOrdersSnapshot(ctx, userAddress); snapshotErr == nil {
			return snapshot, nil
		}
	}
	return nil, err
}

func (s *Service) CreateOrder(ctx context.Context, input CreateOrderInput) (FuturesOrder, error) {
	order, err := s.normalizeOrder(input)
	if err != nil {
		return FuturesOrder{}, err
	}
	if len(input.ExchangePayload) == 0 {
		if signed, signErr := s.signExchangeAction(ctx, AgentSignInput{
			UserID:         order.UserID,
			UserAddress:    order.UserAddress,
			Action:         "order",
			Symbol:         order.Symbol,
			Payload:        order.RawPayload,
			ExchangeAction: exchangeActionFromPayload(order.RawPayload),
		}); signErr == nil {
			order.RawPayload["exchangePayload"] = signed.ExchangePayload
			order.ResponsePayload["agentSigner"] = agentSignedPayload(signed)
		} else if s.agentSigner != nil && s.agentSigner.Enabled() {
			_, _ = s.audit(ctx, order.UserID, order.UserAddress, "hyperliquid.agent_sign_failed", "high", map[string]any{"action": "order", "error": signErr.Error()})
			return FuturesOrder{}, signErr
		}
	}
	created, err := s.store.CreateOrder(ctx, order)
	if err != nil {
		return FuturesOrder{}, err
	}
	if created.ID != order.ID || created.ProviderOrderID != "" || created.Status != "pending" {
		return created, nil
	}
	result, err := s.provider.SubmitOrder(ctx, created)
	if err != nil {
		_, _ = s.audit(ctx, created.UserID, created.UserAddress, "hyperliquid.order_submit_failed", "high", map[string]any{"orderId": created.ID, "error": err.Error()})
		return FuturesOrder{}, err
	}
	submittedAt := result.SubmittedAt
	updated, err := s.store.UpdateOrder(ctx, FuturesOrder{
		ID:              created.ID,
		Status:          "submitted",
		Provider:        result.Provider,
		ProviderOrderID: result.RequestID,
		ResponsePayload: providerResultPayload(result),
		SubmittedAt:     &submittedAt,
	})
	if err != nil {
		return FuturesOrder{}, err
	}
	_, _ = s.audit(ctx, updated.UserID, updated.UserAddress, "hyperliquid.order_submitted", "high", map[string]any{"orderId": updated.ID, "symbol": updated.Symbol, "side": updated.Side, "size": updated.Size, "providerOrderId": updated.ProviderOrderID})
	return updated, nil
}

func (s *Service) CancelOrder(ctx context.Context, input CancelOrderInput) (FuturesOrder, error) {
	if strings.TrimSpace(input.OrderID) == "" {
		return FuturesOrder{}, fmt.Errorf("orderId is required")
	}
	order, err := s.store.GetOrder(ctx, input.OrderID)
	if err != nil {
		return FuturesOrder{}, err
	}
	input.UserID = firstNonEmpty(input.UserID, order.UserID)
	input.UserAddress = firstNonEmpty(input.UserAddress, order.UserAddress)
	input.Symbol = firstNonEmpty(input.Symbol, order.Symbol)
	input.Cloid = firstNonEmpty(input.Cloid, order.Cloid)
	if len(input.ExchangePayload) == 0 {
		if signed, signErr := s.signExchangeAction(ctx, AgentSignInput{
			UserID:         input.UserID,
			UserAddress:    input.UserAddress,
			Action:         "cancel",
			Symbol:         input.Symbol,
			Payload:        map[string]any{"orderId": input.OrderID, "cloid": input.Cloid, "symbol": input.Symbol, "exchangeAction": input.ExchangeAction},
			ExchangeAction: input.ExchangeAction,
		}); signErr == nil {
			input.ExchangePayload = signed.ExchangePayload
		} else if s.agentSigner != nil && s.agentSigner.Enabled() {
			_, _ = s.audit(ctx, input.UserID, input.UserAddress, "hyperliquid.agent_sign_failed", "high", map[string]any{"action": "cancel", "orderId": input.OrderID, "error": signErr.Error()})
			return FuturesOrder{}, signErr
		}
	}
	result, err := s.provider.CancelOrder(ctx, input)
	if err != nil {
		_, _ = s.audit(ctx, input.UserID, input.UserAddress, "hyperliquid.order_cancel_failed", "high", map[string]any{"orderId": input.OrderID, "error": err.Error()})
		return FuturesOrder{}, err
	}
	cancelledAt := result.SubmittedAt
	updated, err := s.store.UpdateOrder(ctx, FuturesOrder{
		ID:              order.ID,
		Status:          "cancelled",
		Provider:        result.Provider,
		ProviderOrderID: firstNonEmpty(order.ProviderOrderID, result.RequestID),
		ResponsePayload: providerResultPayload(result),
		CancelledAt:     &cancelledAt,
	})
	if err != nil {
		return FuturesOrder{}, err
	}
	_, _ = s.audit(ctx, updated.UserID, updated.UserAddress, "hyperliquid.order_cancelled", "high", map[string]any{"orderId": updated.ID, "providerOrderId": updated.ProviderOrderID})
	return updated, nil
}

func (s *Service) SyncOrderStatus(ctx context.Context, input OrderStatusInput) (FuturesOrder, error) {
	input.OrderID = strings.TrimSpace(input.OrderID)
	if input.OrderID == "" {
		return FuturesOrder{}, fmt.Errorf("orderId is required")
	}
	order, err := s.store.GetOrder(ctx, input.OrderID)
	if err != nil {
		return FuturesOrder{}, err
	}

	input.UserID = firstNonEmpty(input.UserID, order.UserID)
	input.UserAddress = firstNonEmpty(input.UserAddress, order.UserAddress)
	input.ProviderOrderID = firstNonEmpty(input.ProviderOrderID, order.ProviderOrderID)
	input.Cloid = firstNonEmpty(input.Cloid, order.Cloid)
	input.Symbol = firstNonEmpty(input.Symbol, order.Symbol)

	status, err := s.provider.OrderStatus(ctx, input)
	if err != nil {
		_, _ = s.audit(ctx, input.UserID, input.UserAddress, "hyperliquid.order_status_sync_failed", "medium", map[string]any{"orderId": input.OrderID, "providerOrderId": input.ProviderOrderID, "cloid": input.Cloid, "error": err.Error()})
		return FuturesOrder{}, err
	}

	nextStatus := mergeSyncedOrderStatus(order.Status, normalizeProviderOrderStatus(status.Status))
	updatedAt := status.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = s.now().UTC()
	}
	var cancelledAt *time.Time
	if nextStatus == "cancelled" && order.CancelledAt == nil {
		cancelledAt = &updatedAt
	}

	updated, err := s.store.UpdateOrder(ctx, FuturesOrder{
		ID:              order.ID,
		Status:          nextStatus,
		Provider:        s.provider.Name(),
		ProviderOrderID: firstNonEmpty(status.ProviderOrderID, order.ProviderOrderID),
		ResponsePayload: map[string]any{"orderStatus": orderStatusPayload(status)},
		CancelledAt:     cancelledAt,
	})
	if err != nil {
		return FuturesOrder{}, err
	}
	_, _ = s.audit(ctx, updated.UserID, updated.UserAddress, "hyperliquid.order_status_synced", "medium", map[string]any{"orderId": updated.ID, "providerOrderId": updated.ProviderOrderID, "status": updated.Status, "providerStatus": status.Status})
	return updated, nil
}

func (s *Service) Orders(ctx context.Context, filter OrderFilter) ([]FuturesOrder, error) {
	return s.store.ListOrders(ctx, filter)
}

func (s *Service) UpdateLeverage(ctx context.Context, input UpdateLeverageInput) (ProviderActionResult, error) {
	input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
	if input.Symbol == "" {
		return ProviderActionResult{}, fmt.Errorf("symbol is required")
	}
	if input.Leverage <= 0 || input.Leverage > 100 {
		return ProviderActionResult{}, fmt.Errorf("leverage must be between 1 and 100")
	}
	if len(input.ExchangePayload) == 0 {
		if signed, signErr := s.signExchangeAction(ctx, AgentSignInput{
			UserID:         input.UserID,
			UserAddress:    input.UserAddress,
			Action:         "updateLeverage",
			Symbol:         input.Symbol,
			Leverage:       input.Leverage,
			Payload:        map[string]any{"symbol": input.Symbol, "leverage": input.Leverage, "isCross": input.IsCross, "exchangeAction": input.ExchangeAction},
			ExchangeAction: input.ExchangeAction,
		}); signErr == nil {
			input.ExchangePayload = signed.ExchangePayload
		} else if s.agentSigner != nil && s.agentSigner.Enabled() {
			_, _ = s.audit(ctx, input.UserID, input.UserAddress, "hyperliquid.agent_sign_failed", "high", map[string]any{"action": "updateLeverage", "symbol": input.Symbol, "leverage": input.Leverage, "error": signErr.Error()})
			return ProviderActionResult{}, signErr
		}
	}
	result, err := s.provider.UpdateLeverage(ctx, input)
	if err != nil {
		_, _ = s.audit(ctx, input.UserID, input.UserAddress, "hyperliquid.update_leverage_failed", "high", map[string]any{"symbol": input.Symbol, "error": err.Error()})
		return ProviderActionResult{}, err
	}
	_, _ = s.audit(ctx, input.UserID, input.UserAddress, "hyperliquid.update_leverage", "high", map[string]any{"symbol": input.Symbol, "leverage": input.Leverage, "isCross": input.IsCross, "requestId": result.RequestID})
	return result, nil
}

func (s *Service) FundingRates(ctx context.Context, symbol string, limit int) ([]FundingRate, error) {
	return s.provider.FundingRates(ctx, symbol, limit)
}

func (s *Service) AuditEvents(ctx context.Context, userID string, limit int) ([]AuditEvent, error) {
	return s.store.ListAuditEvents(ctx, userID, limit)
}

func (s *Service) SmartMoney(ctx context.Context, limit int) ([]SmartMoneyTrader, error) {
	return s.store.ListSmartMoney(ctx, limit)
}

func (s *Service) Groups(ctx context.Context, userID string) ([]AddressGroup, error) {
	return s.store.ListGroups(ctx, userID)
}

func (s *Service) CreateGroup(ctx context.Context, name string, userID string, isDefault bool) (AddressGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return AddressGroup{}, fmt.Errorf("group name is required")
	}
	return s.store.CreateGroup(ctx, AddressGroup{Name: name, UserID: userID, IsDefault: isDefault})
}

func (s *Service) UpdateGroup(ctx context.Context, id string, name string, isDefault bool, order int) (AddressGroup, error) {
	if strings.TrimSpace(id) == "" {
		return AddressGroup{}, fmt.Errorf("group id is required")
	}
	return s.store.UpdateGroup(ctx, AddressGroup{ID: id, Name: name, IsDefault: isDefault, Order: order})
}

func (s *Service) DeleteGroup(ctx context.Context, id string) error {
	return s.store.DeleteGroup(ctx, id)
}

func (s *Service) Addresses(ctx context.Context, groupID string) ([]Address, error) {
	return s.store.ListAddresses(ctx, groupID)
}

func (s *Service) CreateAddress(ctx context.Context, address string, remarkName string, groupIDs []string, userID string) (Address, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return Address{}, fmt.Errorf("address is required")
	}
	if len(groupIDs) == 0 {
		groupIDs = []string{"default"}
	}
	return s.store.CreateAddress(ctx, Address{Address: address, RemarkName: remarkName, GroupIDs: groupIDs, OwnerUserID: userID, UserAddress: address, Profit1d: "0", Profit7d: "0", Profit30d: "0"})
}

func (s *Service) UpdateAddress(ctx context.Context, id string, remarkName string, groupIDs []string) (Address, error) {
	if strings.TrimSpace(id) == "" {
		return Address{}, fmt.Errorf("address id is required")
	}
	return s.store.UpdateAddress(ctx, Address{ID: id, RemarkName: remarkName, GroupIDs: groupIDs})
}

func (s *Service) DeleteAddress(ctx context.Context, id string) error {
	return s.store.DeleteAddress(ctx, id)
}

func (s *Service) WalletStatus(ctx context.Context, userAddress string) (HyperliquidWalletStatus, error) {
	status, err := s.provider.WalletStatus(ctx, userAddress)
	if err != nil {
		return status, err
	}
	if s.agentSigner != nil && s.agentSigner.Enabled() {
		if wallets, listErr := s.agentSigner.ListWallets(ctx, userAddress); listErr == nil {
			for _, wallet := range wallets {
				if wallet.Status == "active" {
					status.ApprovedAgent = true
					status.Agent = wallet.AgentAddress
					status.AgentName = wallet.AgentName
					break
				}
			}
		}
	}
	return status, nil
}

func (s *Service) Sign(ctx context.Context, userID string, userAddress string, action string, payload map[string]any) (map[string]any, error) {
	if userID == "" {
		userID = "local-user"
	}
	if s.agentSigner != nil && s.agentSigner.Enabled() {
		if strings.EqualFold(action, "approveHyperLiquidApproveAgent") {
			approval, err := s.CreateAgentWallet(ctx, CreateAgentWalletInput{
				UserID:           userID,
				UserAddress:      userAddress,
				AgentName:        stringFromAny(payload["agentName"]),
				HyperliquidChain: stringFromAny(payload["hyperliquidChain"]),
				SignatureChainID: stringFromAny(payload["signatureChainId"]),
				Policy:           agentPolicyFromMap(asMap(payload["policy"])),
			})
			if err != nil {
				_, _ = s.audit(ctx, userID, userAddress, "hyperliquid.agent_create_failed", "high", map[string]any{"action": action, "error": err.Error()})
				return nil, err
			}
			_, _ = s.audit(ctx, userID, userAddress, "hyperliquid.agent_created", "high", map[string]any{"agentAddress": approval.Wallet.AgentAddress, "agentName": approval.Wallet.AgentName})
			return map[string]any{"wallet": approval.Wallet, "approvalPayload": approval.ApprovalPayload, "status": "requires_user_signature"}, nil
		}
		if managedAgentAction(action) {
			if signed, err := s.signExchangeAction(ctx, AgentSignInput{
				UserID:         userID,
				UserAddress:    userAddress,
				Action:         action,
				Symbol:         stringValue(payload, "symbol", "coin"),
				Leverage:       intValue(payload, 0, "leverage"),
				Payload:        payload,
				ExchangeAction: asMap(payload["exchangeAction"]),
				VaultAddress:   stringValue(payload, "vaultAddress"),
				ExpiresAfter:   int64FromAny(payload["expiresAfter"]),
			}); err == nil {
				_, _ = s.audit(ctx, userID, userAddress, "hyperliquid.agent_sign", "high", map[string]any{"action": signed.Action, "agentAddress": signed.AgentWallet.AgentAddress, "nonce": signed.Nonce})
				return agentSignedPayload(signed), nil
			} else {
				_, _ = s.audit(ctx, userID, userAddress, "hyperliquid.agent_sign_failed", "high", map[string]any{"action": action, "error": err.Error()})
				return nil, err
			}
		}
	}
	result, err := s.provider.Sign(ctx, action, userID, payload)
	if err != nil {
		_, _ = s.audit(ctx, userID, userAddress, "hyperliquid.sign_failed", "high", map[string]any{"action": action, "error": err.Error()})
		return nil, err
	}
	_, _ = s.audit(ctx, userID, userAddress, "hyperliquid.sign", "high", map[string]any{"action": action, "requestId": result.RequestID})
	return map[string]any{
		"signature": result.Signature,
		"userId":    userID,
		"provider":  result.Provider,
		"requestId": result.RequestID,
	}, nil
}

func (s *Service) CreateAgentWallet(ctx context.Context, input CreateAgentWalletInput) (AgentApproval, error) {
	if s.agentSigner == nil || !s.agentSigner.Enabled() {
		return AgentApproval{}, fmt.Errorf("agent signer is disabled")
	}
	return s.agentSigner.CreateWallet(ctx, input)
}

func (s *Service) ActivateAgentWallet(ctx context.Context, input ActivateAgentWalletInput) (AgentWallet, error) {
	if s.agentSigner == nil || !s.agentSigner.Enabled() {
		return AgentWallet{}, fmt.Errorf("agent signer is disabled")
	}
	wallet, err := s.agentSigner.ActivateWallet(ctx, input)
	if err == nil {
		_, _ = s.audit(ctx, wallet.UserID, wallet.UserAddress, "hyperliquid.agent_status_updated", "high", map[string]any{"agentAddress": wallet.AgentAddress, "status": wallet.Status})
	}
	return wallet, err
}

func (s *Service) AgentWallets(ctx context.Context, userAddress string) ([]AgentWallet, error) {
	if s.agentSigner == nil || !s.agentSigner.Enabled() {
		return nil, fmt.Errorf("agent signer is disabled")
	}
	return s.agentSigner.ListWallets(ctx, userAddress)
}

func (s *Service) AgentSign(ctx context.Context, input AgentSignInput) (AgentSignedPayload, error) {
	signed, err := s.signExchangeAction(ctx, input)
	if err != nil {
		_, _ = s.audit(ctx, input.UserID, input.UserAddress, "hyperliquid.agent_sign_failed", "high", map[string]any{"action": input.Action, "error": err.Error()})
		return AgentSignedPayload{}, err
	}
	_, _ = s.audit(ctx, input.UserID, input.UserAddress, "hyperliquid.agent_sign", "high", map[string]any{"action": signed.Action, "agentAddress": signed.AgentWallet.AgentAddress, "nonce": signed.Nonce})
	return signed, nil
}

func (s *Service) GenerateCloid(count int) map[string]any {
	if count <= 0 || count > 20 {
		count = 1
	}
	cloids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		cloids = append(cloids, uuid.NewString())
	}
	return map[string]any{"count": count, "cloids": cloids}
}

func seedPosition(address string, symbol Symbol, side string, now time.Time) Position {
	positionType := "long"
	szi := "0.2"
	if side == "short" {
		positionType = "short"
		szi = "-4"
	}
	return Position{
		Address: address, Coin: symbol.Symbol, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now, PositionType: positionType, Szi: szi,
		LeverageType: "cross", LeverageValue: 5, EntryPx: symbol.CurrentPrice, PositionValue: "25000", UnrealizedPnl: "420",
		ReturnOnEquity: "0.084", LiquidationPx: "0", MarginUsed: "5000", MaxLeverage: symbol.MaxLeverage, OpenTime: now.Add(-2 * time.Hour).Unix(),
		CumFundingAllTime: "12", CumFundingSinceOpen: "3", CumFundingSinceChange: "1", AccountValue: "250000", CrossMaintenanceMarginUsed: "1200", CrossMarginRatio: "0.12",
		Side: side, Time: now.Unix(), StartPosition: "0", Dir: "Open " + strings.Title(side), ClosedPnl: "0", Hash: "0xposition", Oid: 100, Tid: 200, Crossed: true, Fee: "0.4", TwapID: "",
	}
}

func (s *Service) normalizeOrder(input CreateOrderInput) (FuturesOrder, error) {
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	side := strings.ToLower(strings.TrimSpace(input.Side))
	orderType := strings.ToLower(strings.TrimSpace(input.OrderType))
	size := strings.TrimSpace(input.Size)
	if symbol == "" {
		return FuturesOrder{}, fmt.Errorf("symbol is required")
	}
	if side != "buy" && side != "sell" && side != "long" && side != "short" {
		return FuturesOrder{}, fmt.Errorf("side must be buy, sell, long or short")
	}
	if orderType == "" {
		orderType = "market"
	}
	if size == "" {
		return FuturesOrder{}, fmt.Errorf("size is required")
	}
	now := s.now().UTC()
	payload := input.RawPayload
	if payload == nil {
		payload = map[string]any{}
	}
	if len(input.ExchangePayload) > 0 {
		payload["exchangePayload"] = input.ExchangePayload
	}
	if len(input.ExchangeAction) > 0 {
		payload["exchangeAction"] = input.ExchangeAction
	}
	return FuturesOrder{
		ID:              uuid.NewString(),
		UserID:          strings.TrimSpace(input.UserID),
		UserAddress:     strings.TrimSpace(input.UserAddress),
		Symbol:          symbol,
		Side:            side,
		OrderType:       orderType,
		Price:           strings.TrimSpace(input.Price),
		Size:            size,
		Status:          "pending",
		Cloid:           strings.TrimSpace(input.Cloid),
		Provider:        s.provider.Name(),
		ClientRequestID: strings.TrimSpace(input.ClientRequestID),
		ReduceOnly:      input.ReduceOnly,
		TimeInForce:     strings.TrimSpace(input.TimeInForce),
		RawPayload:      payload,
		ResponsePayload: map[string]any{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (s *Service) audit(ctx context.Context, userID string, userAddress string, action string, riskLevel string, payload map[string]any) (AuditEvent, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	return s.store.AppendAuditEvent(ctx, AuditEvent{
		UserID:      strings.TrimSpace(userID),
		UserAddress: strings.TrimSpace(userAddress),
		Action:      action,
		RiskLevel:   riskLevel,
		Payload:     payload,
		CreatedAt:   s.now().UTC(),
	})
}

func providerResultPayload(result ProviderActionResult) map[string]any {
	return map[string]any{
		"action":      result.Action,
		"provider":    result.Provider,
		"requestId":   result.RequestID,
		"status":      result.Status,
		"signature":   result.Signature,
		"submittedAt": result.SubmittedAt,
		"rawPayload":  result.RawPayload,
	}
}

func (s *Service) signExchangeAction(ctx context.Context, input AgentSignInput) (AgentSignedPayload, error) {
	if s.agentSigner == nil || !s.agentSigner.Enabled() {
		return AgentSignedPayload{}, fmt.Errorf("agent signer is disabled")
	}
	return s.agentSigner.Sign(ctx, input)
}

func exchangeActionFromPayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}
	if action := asMap(payload["exchangeAction"]); len(action) > 0 {
		return action
	}
	if exchangePayload := asMap(payload["exchangePayload"]); len(exchangePayload) > 0 {
		return asMap(exchangePayload["action"])
	}
	return asMap(payload["action"])
}

func agentSignedPayload(signed AgentSignedPayload) map[string]any {
	return map[string]any{
		"agentWallet": map[string]any{
			"id":           signed.AgentWallet.ID,
			"userAddress":  signed.AgentWallet.UserAddress,
			"agentAddress": signed.AgentWallet.AgentAddress,
			"agentName":    signed.AgentWallet.AgentName,
			"status":       signed.AgentWallet.Status,
		},
		"exchangePayload": signed.ExchangePayload,
		"signature":       signed.Signature,
		"nonce":           signed.Nonce,
		"action":          signed.Action,
		"status":          signed.Status,
	}
}

func agentPolicyFromMap(input map[string]any) AgentPolicy {
	if len(input) == 0 {
		return AgentPolicy{}
	}
	return AgentPolicy{
		AllowedActions: stringListFromAny(input["allowedActions"]),
		AllowedSymbols: stringListFromAny(input["allowedSymbols"]),
		MaxLeverage:    intValue(input, 0, "maxLeverage"),
	}
}

func stringListFromAny(value any) []string {
	switch v := value.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if text := strings.TrimSpace(stringFromAny(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	case string:
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if text := strings.TrimSpace(part); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func managedAgentAction(action string) bool {
	switch normalizeAgentAction(action) {
	case "order", "cancel", "updateLeverage":
		return true
	default:
		return false
	}
}

func orderStatusPayload(status OrderStatus) map[string]any {
	return map[string]any{
		"orderId":         status.OrderID,
		"providerOrderId": status.ProviderOrderID,
		"cloid":           status.Cloid,
		"symbol":          status.Symbol,
		"status":          status.Status,
		"filledSize":      status.FilledSize,
		"remainingSize":   status.RemainingSize,
		"averagePrice":    status.AveragePrice,
		"rawPayload":      status.RawPayload,
		"updatedAt":       status.UpdatedAt,
	}
}

func normalizeProviderOrderStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case "", "ok", "order", "open", "resting", "submitted", "triggered":
		return "submitted"
	case "filled":
		return "filled"
	case "cancelled", "canceled":
		return "cancelled"
	case "rejected", "failed":
		return "failed"
	case "unknown", "unknownoid":
		return "unknown"
	default:
		if strings.Contains(normalized, "cancel") {
			return "cancelled"
		}
		if strings.Contains(normalized, "reject") {
			return "failed"
		}
		return normalized
	}
}

func mergeSyncedOrderStatus(current string, next string) string {
	if strings.TrimSpace(next) == "" {
		return current
	}
	if terminalOrderStatus(current) && next == "submitted" {
		return current
	}
	return next
}

func terminalOrderStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "cancelled", "filled", "failed", "rejected":
		return true
	default:
		return false
	}
}
