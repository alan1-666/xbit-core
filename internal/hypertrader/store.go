package hypertrader

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	ListSymbols(ctx context.Context, query string, category string, limit int) ([]Symbol, error)
	GetSymbol(ctx context.Context, symbol string) (Symbol, error)
	GetPreference(ctx context.Context, userID string, symbol string) (SymbolPreference, error)
	SavePreference(ctx context.Context, pref SymbolPreference) (SymbolPreference, error)
	CreateOrder(ctx context.Context, order FuturesOrder) (FuturesOrder, error)
	UpdateOrder(ctx context.Context, order FuturesOrder) (FuturesOrder, error)
	GetOrder(ctx context.Context, id string) (FuturesOrder, error)
	ListOrders(ctx context.Context, filter OrderFilter) ([]FuturesOrder, error)
	AppendAuditEvent(ctx context.Context, event AuditEvent) (AuditEvent, error)
	ListAuditEvents(ctx context.Context, userID string, limit int) ([]AuditEvent, error)
	ListSmartMoney(ctx context.Context, limit int) ([]SmartMoneyTrader, error)
	ListGroups(ctx context.Context, userID string) ([]AddressGroup, error)
	CreateGroup(ctx context.Context, group AddressGroup) (AddressGroup, error)
	UpdateGroup(ctx context.Context, group AddressGroup) (AddressGroup, error)
	DeleteGroup(ctx context.Context, id string) error
	ListAddresses(ctx context.Context, groupID string) ([]Address, error)
	CreateAddress(ctx context.Context, address Address) (Address, error)
	UpdateAddress(ctx context.Context, address Address) (Address, error)
	DeleteAddress(ctx context.Context, id string) error
}

type StateStore interface {
	SaveOpenOrdersSnapshot(ctx context.Context, userAddress string, orders []OpenOrder) error
	ListOpenOrdersSnapshot(ctx context.Context, userAddress string) ([]OpenOrder, error)
	AppendFills(ctx context.Context, userAddress string, fills []TradeHistory) error
	ListFills(ctx context.Context, userAddress string, limit int) ([]TradeHistory, error)
	SaveAccountSnapshot(ctx context.Context, userAddress string, account AccountBalance) error
	GetAccountSnapshot(ctx context.Context, userAddress string) (AccountBalance, error)
	UpdateOrderStatusByProvider(ctx context.Context, input OrderStatusInput, status OrderStatus) error
}

type AgentStore interface {
	SaveAgentWallet(ctx context.Context, wallet AgentWallet) (AgentWallet, error)
	ListAgentWallets(ctx context.Context, userAddress string) ([]AgentWallet, error)
	GetActiveAgentWallet(ctx context.Context, userAddress string) (AgentWallet, error)
	UpdateAgentWalletStatus(ctx context.Context, userAddress string, agentAddress string, status string) (AgentWallet, error)
	NextAgentNonce(ctx context.Context, agentAddress string, now time.Time) (int64, error)
}

type MemoryStore struct {
	mu               sync.RWMutex
	symbols          map[string]Symbol
	preferences      map[string]SymbolPreference
	orders           map[string]FuturesOrder
	audits           []AuditEvent
	traders          []SmartMoneyTrader
	groups           map[string]AddressGroup
	addresses        map[string]Address
	openOrders       map[string]map[string]OpenOrder
	fills            map[string]map[string]TradeHistory
	accountSnapshots map[string]AccountBalance
	agentWallets     map[string]AgentWallet
	agentNonces      map[string]int64
}

func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		symbols:          map[string]Symbol{},
		preferences:      map[string]SymbolPreference{},
		orders:           map[string]FuturesOrder{},
		groups:           map[string]AddressGroup{},
		addresses:        map[string]Address{},
		openOrders:       map[string]map[string]OpenOrder{},
		fills:            map[string]map[string]TradeHistory{},
		accountSnapshots: map[string]AccountBalance{},
		agentWallets:     map[string]AgentWallet{},
		agentNonces:      map[string]int64{},
	}
	store.seed()
	return store
}

func (s *MemoryStore) seed() {
	now := time.Now().UTC().Truncate(time.Second)
	for _, symbol := range []Symbol{
		{Symbol: "BTC", AliasName: "Bitcoin", MaxLeverage: 50, MarketCap: "1800000000000", Volume: "42000000000", ChangePercent: "1.82", OpenInterest: "12400000000", CurrentPrice: "95200", Type: "PERP", QuoteSymbol: "USDC", Category: "major", CreatedAt: now},
		{Symbol: "ETH", AliasName: "Ethereum", MaxLeverage: 50, MarketCap: "420000000000", Volume: "18000000000", ChangePercent: "0.95", OpenInterest: "6800000000", CurrentPrice: "3200.5", Type: "PERP", QuoteSymbol: "USDC", Category: "major", CreatedAt: now},
		{Symbol: "SOL", AliasName: "Solana", MaxLeverage: 20, MarketCap: "68000000000", Volume: "3200000000", ChangePercent: "2.41", OpenInterest: "960000000", CurrentPrice: "145.23", Type: "PERP", QuoteSymbol: "USDC", Category: "major", CreatedAt: now},
		{Symbol: "HYPE", AliasName: "Hyperliquid", MaxLeverage: 10, MarketCap: "9000000000", Volume: "890000000", ChangePercent: "4.8", OpenInterest: "320000000", CurrentPrice: "27.4", Type: "PERP", QuoteSymbol: "USDC", Category: "defi", CreatedAt: now},
	} {
		s.symbols[strings.ToUpper(symbol.Symbol)] = symbol
	}

	defaultGroup := AddressGroup{ID: "default", Name: "Default", UserID: "local-user", IsDefault: true, Order: 0, CreatedAt: now, UpdatedAt: now}
	s.groups[defaultGroup.ID] = defaultGroup
	for i, address := range []string{"0xsmart001", "0xsmart002", "0xsmart003"} {
		addr := Address{ID: uuid.NewString(), Address: address, RemarkName: "Trader " + strings.TrimPrefix(address, "0xsmart"), GroupIDs: []string{defaultGroup.ID}, OwnerUserID: "local-user", UserAddress: address, Profit1d: "1200", Profit7d: "8600", Profit30d: "42100", CreatedAt: now.Add(-time.Duration(i) * time.Hour), UpdatedAt: now}
		s.addresses[addr.ID] = addr
	}

	tag := TraderTag{ID: 1, Category: "style", Name: "trend", NameCN: "趋势", Color: "#46C2A9", Priority: 1, Description: "Trend follower", CreatedAt: now}
	s.traders = []SmartMoneyTrader{
		seedTrader("0xsmart001", "82.4", "42100", tag, now),
		seedTrader("0xsmart002", "63.1", "31880", tag, now),
		seedTrader("0xsmart003", "41.9", "22040", tag, now),
	}
}

func seedTrader(address string, roi string, pnl string, tag TraderTag, now time.Time) SmartMoneyTrader {
	return SmartMoneyTrader{
		UserAddress: address, ROI: roi, NetPnL: pnl, AvgWinRate: "0.62", MaxDrawdown: "0.08", PeriodDays: 30,
		SharpeRatio: "2.1", ProfitLossRatio: "1.8", ProfitFactor: "2.4", TotalVolume: "1250000", AvgDailyVolume: "41666",
		TradingDays: 24, TotalTrades: 138, UniqueCoinsCount: 12, AvgTradesPerDay: "5.75", TotalLongPnL: pnl, TotalShortPnL: "1200",
		WinningPnLTotal: pnl, LosingPnLTotal: "-4100", KOLLabels: []string{"trend"}, KOLLabelsDescription: []string{"Trend follower"},
		FollowerCount: 120, RemarkName: "Smart " + address[len(address)-3:], GroupIDs: []string{"default"}, PortfolioData: map[string]any{"accountValue": "250000"},
		LastOperation: TradeHistory{Symbol: "BTC", Time: now.Unix(), PnL: "830", PnLPercent: "0.024", Dir: "Open Long", Hash: "0xop", Oid: 1, Px: "95200", StartPosition: "0", Sz: "0.2", Fee: "1.2", FeeToken: "USDC", Tid: 10},
		Tags:          []TraderTag{tag},
	}
}

func (s *MemoryStore) ListSymbols(_ context.Context, query string, category string, limit int) ([]Symbol, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query = strings.ToUpper(strings.TrimSpace(query))
	category = strings.ToLower(strings.TrimSpace(category))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	out := make([]Symbol, 0, len(s.symbols))
	for _, symbol := range s.symbols {
		if query != "" && !strings.Contains(symbol.Symbol+" "+strings.ToUpper(symbol.AliasName), query) {
			continue
		}
		if category != "" && strings.ToLower(symbol.Category) != category {
			continue
		}
		out = append(out, symbol)
	}
	sort.Slice(out, func(i, j int) bool {
		return decimalFloat(out[i].Volume) > decimalFloat(out[j].Volume)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) GetSymbol(_ context.Context, symbol string) (Symbol, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out, ok := s.symbols[strings.ToUpper(strings.TrimSpace(symbol))]
	if !ok {
		return Symbol{}, ErrNotFound
	}
	return out, nil
}

func (s *MemoryStore) GetPreference(_ context.Context, userID string, symbol string) (SymbolPreference, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := prefKey(userID, symbol)
	if pref, ok := s.preferences[key]; ok {
		return pref, nil
	}
	return SymbolPreference{UserID: userID, Symbol: strings.ToUpper(symbol), IsFavorite: false, Leverage: 5, IsCross: true}, nil
}

func (s *MemoryStore) SavePreference(_ context.Context, pref SymbolPreference) (SymbolPreference, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pref.Symbol = strings.ToUpper(strings.TrimSpace(pref.Symbol))
	if pref.Leverage <= 0 {
		pref.Leverage = 5
	}
	s.preferences[prefKey(pref.UserID, pref.Symbol)] = pref
	return pref, nil
}

func (s *MemoryStore) CreateOrder(_ context.Context, order FuturesOrder) (FuturesOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if order.ClientRequestID != "" {
		for _, existing := range s.orders {
			if existing.ClientRequestID == order.ClientRequestID && sameOwner(existing, order) {
				return existing, nil
			}
		}
	}
	now := time.Now().UTC()
	if order.ID == "" {
		order.ID = uuid.NewString()
	}
	if order.Status == "" {
		order.Status = "pending"
	}
	if order.CreatedAt.IsZero() {
		order.CreatedAt = now
	}
	order.UpdatedAt = now
	s.orders[order.ID] = order
	return order, nil
}

func (s *MemoryStore) UpdateOrder(_ context.Context, order FuturesOrder) (FuturesOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.orders[order.ID]
	if !ok {
		return FuturesOrder{}, ErrNotFound
	}
	existing.Status = firstNonEmpty(order.Status, existing.Status)
	existing.Provider = firstNonEmpty(order.Provider, existing.Provider)
	existing.ProviderOrderID = firstNonEmpty(order.ProviderOrderID, existing.ProviderOrderID)
	existing.ResponsePayload = mergePayload(existing.ResponsePayload, order.ResponsePayload)
	if order.SubmittedAt != nil {
		existing.SubmittedAt = order.SubmittedAt
	}
	if order.CancelledAt != nil {
		existing.CancelledAt = order.CancelledAt
	}
	existing.UpdatedAt = time.Now().UTC()
	s.orders[existing.ID] = existing
	return existing, nil
}

func (s *MemoryStore) GetOrder(_ context.Context, id string) (FuturesOrder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[id]
	if !ok {
		return FuturesOrder{}, ErrNotFound
	}
	return order, nil
}

func (s *MemoryStore) ListOrders(_ context.Context, filter OrderFilter) ([]FuturesOrder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	out := make([]FuturesOrder, 0, len(s.orders))
	for _, order := range s.orders {
		if filter.UserID != "" && order.UserID != filter.UserID {
			continue
		}
		if filter.UserAddress != "" && !strings.EqualFold(order.UserAddress, filter.UserAddress) {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(order.Status, filter.Status) {
			continue
		}
		if filter.Symbol != "" && !strings.EqualFold(order.Symbol, filter.Symbol) {
			continue
		}
		out = append(out, order)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func (s *MemoryStore) AppendAuditEvent(_ context.Context, event AuditEvent) (AuditEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.RiskLevel == "" {
		event.RiskLevel = "low"
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	s.audits = append(s.audits, event)
	return event, nil
}

func (s *MemoryStore) ListAuditEvents(_ context.Context, userID string, limit int) ([]AuditEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	out := make([]AuditEvent, 0, len(s.audits))
	for i := len(s.audits) - 1; i >= 0; i-- {
		event := s.audits[i]
		if userID != "" && event.UserID != userID {
			continue
		}
		out = append(out, event)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *MemoryStore) ListSmartMoney(_ context.Context, limit int) ([]SmartMoneyTrader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	out := append([]SmartMoneyTrader(nil), s.traders...)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) ListGroups(_ context.Context, userID string) ([]AddressGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AddressGroup, 0, len(s.groups))
	for _, group := range s.groups {
		if userID == "" || group.UserID == "" || group.UserID == userID {
			out = append(out, group)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out, nil
}

func (s *MemoryStore) CreateGroup(_ context.Context, group AddressGroup) (AddressGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if group.ID == "" {
		group.ID = uuid.NewString()
	}
	group.CreatedAt = now
	group.UpdatedAt = now
	s.groups[group.ID] = group
	return group, nil
}

func (s *MemoryStore) UpdateGroup(_ context.Context, group AddressGroup) (AddressGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.groups[group.ID]
	if !ok {
		return AddressGroup{}, ErrNotFound
	}
	if group.Name != "" {
		existing.Name = group.Name
	}
	existing.IsDefault = group.IsDefault
	existing.Order = group.Order
	existing.UpdatedAt = time.Now().UTC()
	s.groups[group.ID] = existing
	return existing, nil
}

func (s *MemoryStore) DeleteGroup(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.groups, id)
	return nil
}

func (s *MemoryStore) ListAddresses(_ context.Context, groupID string) ([]Address, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Address, 0, len(s.addresses))
	for _, address := range s.addresses {
		if groupID != "" && !contains(address.GroupIDs, groupID) {
			continue
		}
		out = append(out, address)
	}
	return out, nil
}

func (s *MemoryStore) CreateAddress(_ context.Context, address Address) (Address, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if address.ID == "" {
		address.ID = uuid.NewString()
	}
	address.CreatedAt = now
	address.UpdatedAt = now
	s.addresses[address.ID] = address
	return address, nil
}

func (s *MemoryStore) UpdateAddress(_ context.Context, address Address) (Address, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.addresses[address.ID]
	if !ok {
		return Address{}, ErrNotFound
	}
	if address.RemarkName != "" {
		existing.RemarkName = address.RemarkName
	}
	if len(address.GroupIDs) > 0 {
		existing.GroupIDs = address.GroupIDs
	}
	existing.UpdatedAt = time.Now().UTC()
	s.addresses[address.ID] = existing
	return existing, nil
}

func (s *MemoryStore) DeleteAddress(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.addresses, id)
	return nil
}

func (s *MemoryStore) SaveOpenOrdersSnapshot(_ context.Context, userAddress string, orders []OpenOrder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ownerKey("", userAddress)
	s.openOrders[key] = map[string]OpenOrder{}
	for _, order := range orders {
		order.UserAddress = firstNonEmpty(order.UserAddress, userAddress)
		id := firstNonEmpty(order.ProviderOrderID, order.Cloid, order.ID)
		if id == "" {
			id = uuid.NewString()
		}
		order.ID = firstNonEmpty(order.ID, id)
		order.ProviderOrderID = firstNonEmpty(order.ProviderOrderID, id)
		if order.UpdatedAt.IsZero() {
			order.UpdatedAt = time.Now().UTC()
		}
		s.openOrders[key][id] = order
	}
	return nil
}

func (s *MemoryStore) ListOpenOrdersSnapshot(_ context.Context, userAddress string) ([]OpenOrder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orders, ok := s.openOrders[ownerKey("", userAddress)]
	if !ok {
		return nil, ErrNotFound
	}
	out := make([]OpenOrder, 0, len(orders))
	for _, order := range orders {
		out = append(out, order)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (s *MemoryStore) AppendFills(_ context.Context, userAddress string, fills []TradeHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ownerKey("", userAddress)
	if s.fills[key] == nil {
		s.fills[key] = map[string]TradeHistory{}
	}
	for _, fill := range fills {
		s.fills[key][fillKey(fill)] = fill
	}
	return nil
}

func (s *MemoryStore) ListFills(_ context.Context, userAddress string, limit int) ([]TradeHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 2000 {
		limit = 100
	}
	fills := s.fills[ownerKey("", userAddress)]
	out := make([]TradeHistory, 0, len(fills))
	for _, fill := range fills {
		out = append(out, fill)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time > out[j].Time })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) SaveAccountSnapshot(_ context.Context, userAddress string, account AccountBalance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountSnapshots[ownerKey("", userAddress)] = account
	return nil
}

func (s *MemoryStore) GetAccountSnapshot(_ context.Context, userAddress string) (AccountBalance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accountSnapshots[ownerKey("", userAddress)]
	if !ok {
		return AccountBalance{}, ErrNotFound
	}
	return account, nil
}

func (s *MemoryStore) UpdateOrderStatusByProvider(_ context.Context, input OrderStatusInput, status OrderStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	nextStatus := normalizeProviderOrderStatus(status.Status)
	for id, order := range s.orders {
		if input.UserAddress != "" && order.UserAddress != "" && !strings.EqualFold(order.UserAddress, input.UserAddress) {
			continue
		}
		if input.UserID != "" && order.UserID != "" && order.UserID != input.UserID {
			continue
		}
		matches := false
		if status.ProviderOrderID != "" && order.ProviderOrderID == status.ProviderOrderID {
			matches = true
		}
		if status.Cloid != "" && order.Cloid == status.Cloid {
			matches = true
		}
		if !matches {
			continue
		}
		order.Status = mergeSyncedOrderStatus(order.Status, nextStatus)
		if order.Status == "cancelled" && order.CancelledAt == nil {
			cancelledAt := time.Now().UTC()
			order.CancelledAt = &cancelledAt
		}
		order.ResponsePayload = mergePayload(order.ResponsePayload, map[string]any{"wsOrderStatus": orderStatusPayload(status)})
		order.UpdatedAt = time.Now().UTC()
		s.orders[id] = order
	}
	return nil
}

func (s *MemoryStore) SaveAgentWallet(_ context.Context, wallet AgentWallet) (AgentWallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if wallet.ID == "" {
		wallet.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if wallet.CreatedAt.IsZero() {
		wallet.CreatedAt = now
	}
	wallet.UpdatedAt = now
	if wallet.Status == "" {
		wallet.Status = "pending_approval"
	}
	if wallet.Policy.MaxLeverage == 0 {
		wallet.Policy.MaxLeverage = 20
	}
	s.agentWallets[agentWalletKey(wallet.UserAddress, wallet.AgentAddress)] = wallet
	return wallet, nil
}

func (s *MemoryStore) ListAgentWallets(_ context.Context, userAddress string) ([]AgentWallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AgentWallet, 0)
	for _, wallet := range s.agentWallets {
		if strings.EqualFold(wallet.UserAddress, userAddress) {
			out = append(out, wallet)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (s *MemoryStore) GetActiveAgentWallet(_ context.Context, userAddress string) (AgentWallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest AgentWallet
	for _, wallet := range s.agentWallets {
		if !strings.EqualFold(wallet.UserAddress, userAddress) || wallet.Status != "active" {
			continue
		}
		if latest.ID == "" || wallet.UpdatedAt.After(latest.UpdatedAt) {
			latest = wallet
		}
	}
	if latest.ID == "" {
		return AgentWallet{}, ErrNotFound
	}
	return latest, nil
}

func (s *MemoryStore) UpdateAgentWalletStatus(_ context.Context, userAddress string, agentAddress string, status string) (AgentWallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := agentWalletKey(userAddress, agentAddress)
	wallet, ok := s.agentWallets[key]
	if !ok {
		return AgentWallet{}, ErrNotFound
	}
	wallet.Status = strings.TrimSpace(status)
	if wallet.Status == "" {
		wallet.Status = "active"
	}
	now := time.Now().UTC()
	wallet.UpdatedAt = now
	if wallet.Status == "active" && wallet.ApprovedAt == nil {
		wallet.ApprovedAt = &now
	}
	s.agentWallets[key] = wallet
	return wallet, nil
}

func (s *MemoryStore) NextAgentNonce(_ context.Context, agentAddress string, now time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.ToLower(strings.TrimSpace(agentAddress))
	next := now.UnixMilli()
	if next <= s.agentNonces[key] {
		next = s.agentNonces[key] + 1
	}
	s.agentNonces[key] = next
	return next, nil
}

func prefKey(userID string, symbol string) string {
	return strings.TrimSpace(userID) + ":" + strings.ToUpper(strings.TrimSpace(symbol))
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func decimalFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

func sameOwner(a FuturesOrder, b FuturesOrder) bool {
	if a.UserID != "" && b.UserID != "" {
		return a.UserID == b.UserID
	}
	if a.UserAddress != "" && b.UserAddress != "" {
		return strings.EqualFold(a.UserAddress, b.UserAddress)
	}
	return true
}

func mergePayload(base map[string]any, patch map[string]any) map[string]any {
	if len(base) == 0 {
		base = map[string]any{}
	}
	for key, value := range patch {
		base[key] = value
	}
	return base
}

func ownerKey(userID string, userAddress string) string {
	return strings.ToLower(firstNonEmpty(userID, userAddress))
}

func fillKey(fill TradeHistory) string {
	if fill.Hash != "" && fill.Tid != 0 {
		return fill.Hash + ":" + strconv.FormatInt(fill.Tid, 10)
	}
	return strings.Join([]string{fill.Symbol, strconv.FormatInt(fill.Time, 10), strconv.FormatInt(fill.Oid, 10), strconv.FormatInt(fill.Tid, 10)}, ":")
}

func agentWalletKey(userAddress string, agentAddress string) string {
	return strings.ToLower(strings.TrimSpace(userAddress)) + ":" + strings.ToLower(strings.TrimSpace(agentAddress))
}

var _ StateStore = (*MemoryStore)(nil)
var _ AgentStore = (*MemoryStore)(nil)
