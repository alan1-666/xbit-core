package hypertrader

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListSymbols(ctx context.Context, query string, category string, limit int) ([]Symbol, error) {
	query = "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	category = strings.ToLower(strings.TrimSpace(category))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT symbol, alias_name, max_leverage, market_cap::text, volume::text, change_percent::text, open_interest::text, current_price::text, symbol_type, quote_symbol, COALESCE(category, ''), created_at
		FROM futures_symbols
		WHERE ($1 = '%%' OR lower(symbol) LIKE $1 OR lower(alias_name) LIKE $1)
		  AND ($2 = '' OR lower(COALESCE(category, '')) = $2)
		ORDER BY futures_symbols.volume DESC, futures_symbols.open_interest DESC, futures_symbols.symbol ASC
		LIMIT $3
	`, query, category, limit)
	if err != nil {
		return nil, fmt.Errorf("list futures symbols: %w", err)
	}
	defer rows.Close()

	symbols := make([]Symbol, 0)
	for rows.Next() {
		symbol, err := scanSymbol(rows)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}
	return symbols, rows.Err()
}

func (s *PostgresStore) GetSymbol(ctx context.Context, symbol string) (Symbol, error) {
	out, err := scanSymbol(s.pool.QueryRow(ctx, `
		SELECT symbol, alias_name, max_leverage, market_cap::text, volume::text, change_percent::text, open_interest::text, current_price::text, symbol_type, quote_symbol, COALESCE(category, ''), created_at
		FROM futures_symbols
		WHERE lower(symbol) = lower($1)
	`, strings.TrimSpace(symbol)))
	if stderrors.Is(err, pgx.ErrNoRows) {
		return Symbol{}, ErrNotFound
	}
	if err != nil {
		return Symbol{}, fmt.Errorf("get futures symbol: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) GetPreference(ctx context.Context, userID string, symbol string) (SymbolPreference, error) {
	var pref SymbolPreference
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, symbol, is_favorite, leverage, is_cross
		FROM symbol_preferences
		WHERE user_id = $1 AND lower(symbol) = lower($2)
	`, strings.TrimSpace(userID), strings.TrimSpace(symbol)).Scan(&pref.UserID, &pref.Symbol, &pref.IsFavorite, &pref.Leverage, &pref.IsCross)
	if stderrors.Is(err, pgx.ErrNoRows) {
		return SymbolPreference{UserID: userID, Symbol: strings.ToUpper(symbol), IsFavorite: false, Leverage: 5, IsCross: true}, nil
	}
	if err != nil {
		return SymbolPreference{}, fmt.Errorf("get symbol preference: %w", err)
	}
	return pref, nil
}

func (s *PostgresStore) SavePreference(ctx context.Context, pref SymbolPreference) (SymbolPreference, error) {
	if pref.Leverage <= 0 {
		pref.Leverage = 5
	}
	pref.Symbol = strings.ToUpper(strings.TrimSpace(pref.Symbol))
	err := s.pool.QueryRow(ctx, `
		INSERT INTO symbol_preferences (user_id, symbol, is_favorite, leverage, is_cross, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (user_id, symbol)
		DO UPDATE SET is_favorite = EXCLUDED.is_favorite, leverage = EXCLUDED.leverage, is_cross = EXCLUDED.is_cross, updated_at = EXCLUDED.updated_at
		RETURNING user_id, symbol, is_favorite, leverage, is_cross
	`, strings.TrimSpace(pref.UserID), pref.Symbol, pref.IsFavorite, pref.Leverage, pref.IsCross).Scan(&pref.UserID, &pref.Symbol, &pref.IsFavorite, &pref.Leverage, &pref.IsCross)
	if err != nil {
		return SymbolPreference{}, fmt.Errorf("save symbol preference: %w", err)
	}
	return pref, nil
}

func (s *PostgresStore) CreateOrder(ctx context.Context, order FuturesOrder) (FuturesOrder, error) {
	if order.ID == "" {
		order.ID = uuid.NewString()
	}
	if order.Status == "" {
		order.Status = "pending"
	}
	rawPayload, err := json.Marshal(order.RawPayload)
	if err != nil {
		return FuturesOrder{}, fmt.Errorf("marshal order payload: %w", err)
	}
	responsePayload, err := json.Marshal(order.ResponsePayload)
	if err != nil {
		return FuturesOrder{}, fmt.Errorf("marshal order response: %w", err)
	}
	out, err := scanFuturesOrder(s.pool.QueryRow(ctx, `
		INSERT INTO hyper_orders (id, user_id, user_address, symbol, side, order_type, price, size, status, cloid, raw_payload, provider, provider_order_id, client_request_id, reduce_only, time_in_force, response_payload, created_at, updated_at)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4, $5, $6, NULLIF($7, '')::numeric, $8, $9, NULLIF($10, ''), $11, $12, NULLIF($13, ''), NULLIF($14, ''), $15, NULLIF($16, ''), $17, $18, $19)
		ON CONFLICT (user_id, client_request_id)
		WHERE client_request_id IS NOT NULL AND client_request_id <> '' AND user_id IS NOT NULL
		DO UPDATE SET updated_at = hyper_orders.updated_at
		RETURNING id::text, COALESCE(user_id, ''), COALESCE(user_address, ''), symbol, side, order_type, COALESCE(price::text, ''), size::text, status, COALESCE(cloid, ''), raw_payload, COALESCE(provider, ''), COALESCE(provider_order_id, ''), COALESCE(client_request_id, ''), reduce_only, COALESCE(time_in_force, ''), response_payload, created_at, updated_at, submitted_at, cancelled_at
	`, order.ID, order.UserID, order.UserAddress, order.Symbol, order.Side, order.OrderType, order.Price, decimal(order.Size), order.Status, order.Cloid, rawPayload, order.Provider, order.ProviderOrderID, order.ClientRequestID, order.ReduceOnly, order.TimeInForce, responsePayload, order.CreatedAt, order.UpdatedAt))
	if err != nil {
		return FuturesOrder{}, fmt.Errorf("create futures order: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) UpdateOrder(ctx context.Context, order FuturesOrder) (FuturesOrder, error) {
	responsePayload, err := json.Marshal(order.ResponsePayload)
	if err != nil {
		return FuturesOrder{}, fmt.Errorf("marshal order response: %w", err)
	}
	out, err := scanFuturesOrder(s.pool.QueryRow(ctx, `
		UPDATE hyper_orders
		SET status = COALESCE(NULLIF($2, ''), status),
		    provider = COALESCE(NULLIF($3, ''), provider),
		    provider_order_id = COALESCE(NULLIF($4, ''), provider_order_id),
		    response_payload = CASE WHEN $5::jsonb = '{}'::jsonb THEN response_payload ELSE response_payload || $5::jsonb END,
		    submitted_at = COALESCE($6, submitted_at),
		    cancelled_at = COALESCE($7, cancelled_at),
		    updated_at = now()
		WHERE id = $1
		RETURNING id::text, COALESCE(user_id, ''), COALESCE(user_address, ''), symbol, side, order_type, COALESCE(price::text, ''), size::text, status, COALESCE(cloid, ''), raw_payload, COALESCE(provider, ''), COALESCE(provider_order_id, ''), COALESCE(client_request_id, ''), reduce_only, COALESCE(time_in_force, ''), response_payload, created_at, updated_at, submitted_at, cancelled_at
	`, order.ID, order.Status, order.Provider, order.ProviderOrderID, responsePayload, order.SubmittedAt, order.CancelledAt))
	if stderrors.Is(err, pgx.ErrNoRows) {
		return FuturesOrder{}, ErrNotFound
	}
	if err != nil {
		return FuturesOrder{}, fmt.Errorf("update futures order: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) GetOrder(ctx context.Context, id string) (FuturesOrder, error) {
	order, err := scanFuturesOrder(s.pool.QueryRow(ctx, `
		SELECT id::text, COALESCE(user_id, ''), COALESCE(user_address, ''), symbol, side, order_type, COALESCE(price::text, ''), size::text, status, COALESCE(cloid, ''), raw_payload, COALESCE(provider, ''), COALESCE(provider_order_id, ''), COALESCE(client_request_id, ''), reduce_only, COALESCE(time_in_force, ''), response_payload, created_at, updated_at, submitted_at, cancelled_at
		FROM hyper_orders
		WHERE id = $1
	`, id))
	if stderrors.Is(err, pgx.ErrNoRows) {
		return FuturesOrder{}, ErrNotFound
	}
	if err != nil {
		return FuturesOrder{}, fmt.Errorf("get futures order: %w", err)
	}
	return order, nil
}

func (s *PostgresStore) ListOrders(ctx context.Context, filter OrderFilter) ([]FuturesOrder, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, COALESCE(user_id, ''), COALESCE(user_address, ''), symbol, side, order_type, COALESCE(price::text, ''), size::text, status, COALESCE(cloid, ''), raw_payload, COALESCE(provider, ''), COALESCE(provider_order_id, ''), COALESCE(client_request_id, ''), reduce_only, COALESCE(time_in_force, ''), response_payload, created_at, updated_at, submitted_at, cancelled_at
		FROM hyper_orders
		WHERE ($1 = '' OR user_id = $1)
		  AND ($2 = '' OR lower(COALESCE(user_address, '')) = lower($2))
		  AND ($3 = '' OR lower(status) = lower($3))
		  AND ($4 = '' OR lower(symbol) = lower($4))
		ORDER BY created_at DESC
		LIMIT $5
	`, strings.TrimSpace(filter.UserID), strings.TrimSpace(filter.UserAddress), strings.TrimSpace(filter.Status), strings.TrimSpace(filter.Symbol), filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list futures orders: %w", err)
	}
	defer rows.Close()

	orders := make([]FuturesOrder, 0)
	for rows.Next() {
		order, err := scanFuturesOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (s *PostgresStore) AppendAuditEvent(ctx context.Context, event AuditEvent) (AuditEvent, error) {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.RiskLevel == "" {
		event.RiskLevel = "low"
	}
	rawPayload, err := json.Marshal(event.Payload)
	if err != nil {
		return AuditEvent{}, fmt.Errorf("marshal audit payload: %w", err)
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO hyper_audit_events (id, user_id, user_address, action, risk_level, payload, created_at)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4, $5, $6, $7)
		RETURNING id::text, COALESCE(user_id, ''), COALESCE(user_address, ''), action, risk_level, payload, created_at
	`, event.ID, event.UserID, event.UserAddress, event.Action, event.RiskLevel, rawPayload, event.CreatedAt).Scan(&event.ID, &event.UserID, &event.UserAddress, &event.Action, &event.RiskLevel, &rawPayload, &event.CreatedAt)
	if err != nil {
		return AuditEvent{}, fmt.Errorf("append hyper audit event: %w", err)
	}
	_ = json.Unmarshal(rawPayload, &event.Payload)
	return event, nil
}

func (s *PostgresStore) ListAuditEvents(ctx context.Context, userID string, limit int) ([]AuditEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, COALESCE(user_id, ''), COALESCE(user_address, ''), action, risk_level, payload, created_at
		FROM hyper_audit_events
		WHERE $1 = '' OR user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, strings.TrimSpace(userID), limit)
	if err != nil {
		return nil, fmt.Errorf("list hyper audit events: %w", err)
	}
	defer rows.Close()

	events := make([]AuditEvent, 0)
	for rows.Next() {
		var event AuditEvent
		var payload []byte
		if err := rows.Scan(&event.ID, &event.UserID, &event.UserAddress, &event.Action, &event.RiskLevel, &payload, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan hyper audit event: %w", err)
		}
		_ = json.Unmarshal(payload, &event.Payload)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *PostgresStore) ListSmartMoney(ctx context.Context, limit int) ([]SmartMoneyTrader, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT user_address, roi::text, net_pnl::text, avg_win_rate::text, max_drawdown::text, period_days, sharpe_ratio::text, profit_loss_ratio::text, profit_factor::text,
		       total_volume::text, avg_daily_volume::text, trading_days, total_trades, unique_coins_count, avg_trades_per_day::text,
		       total_long_pnl::text, total_short_pnl::text, winning_pnl_total::text, losing_pnl_total::text,
		       kol_labels, kol_labels_description, follower_count, remark_name, group_ids, portfolio_data, last_operation, tags
		FROM smart_money_traders
		ORDER BY smart_money_traders.roi DESC, smart_money_traders.net_pnl DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list smart money traders: %w", err)
	}
	defer rows.Close()

	traders := make([]SmartMoneyTrader, 0)
	for rows.Next() {
		trader, err := scanTrader(rows)
		if err != nil {
			return nil, err
		}
		traders = append(traders, trader)
	}
	return traders, rows.Err()
}

func (s *PostgresStore) ListGroups(ctx context.Context, userID string) ([]AddressGroup, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, COALESCE(user_id, ''), is_default, display_order, created_at, updated_at
		FROM address_groups
		WHERE $1 = '' OR user_id IS NULL OR user_id = $1
		ORDER BY display_order ASC, created_at ASC
	`, strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("list address groups: %w", err)
	}
	defer rows.Close()

	groups := make([]AddressGroup, 0)
	for rows.Next() {
		group, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (s *PostgresStore) CreateGroup(ctx context.Context, group AddressGroup) (AddressGroup, error) {
	if group.ID == "" {
		group.ID = uuid.NewString()
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO address_groups (id, name, user_id, is_default, display_order, created_at, updated_at)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, now(), now())
		RETURNING id, name, COALESCE(user_id, ''), is_default, display_order, created_at, updated_at
	`, group.ID, group.Name, group.UserID, group.IsDefault, group.Order).Scan(&group.ID, &group.Name, &group.UserID, &group.IsDefault, &group.Order, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return AddressGroup{}, fmt.Errorf("create address group: %w", err)
	}
	return group, nil
}

func (s *PostgresStore) UpdateGroup(ctx context.Context, group AddressGroup) (AddressGroup, error) {
	err := s.pool.QueryRow(ctx, `
		UPDATE address_groups
		SET name = COALESCE(NULLIF($2, ''), name), is_default = $3, display_order = $4, updated_at = now()
		WHERE id = $1
		RETURNING id, name, COALESCE(user_id, ''), is_default, display_order, created_at, updated_at
	`, group.ID, group.Name, group.IsDefault, group.Order).Scan(&group.ID, &group.Name, &group.UserID, &group.IsDefault, &group.Order, &group.CreatedAt, &group.UpdatedAt)
	if stderrors.Is(err, pgx.ErrNoRows) {
		return AddressGroup{}, ErrNotFound
	}
	if err != nil {
		return AddressGroup{}, fmt.Errorf("update address group: %w", err)
	}
	return group, nil
}

func (s *PostgresStore) DeleteGroup(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM address_groups WHERE id = $1 AND is_default = false`, id)
	if err != nil {
		return fmt.Errorf("delete address group: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListAddresses(ctx context.Context, groupID string) ([]Address, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, address, remark_name, group_ids, COALESCE(owner_user_id, ''), user_address, profit_1d::text, profit_7d::text, profit_30d::text, created_at, updated_at
		FROM followed_addresses
		WHERE $1 = '' OR $1 = ANY(group_ids)
		ORDER BY updated_at DESC, created_at DESC
	`, strings.TrimSpace(groupID))
	if err != nil {
		return nil, fmt.Errorf("list followed addresses: %w", err)
	}
	defer rows.Close()

	addresses := make([]Address, 0)
	for rows.Next() {
		address, err := scanAddress(rows)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, address)
	}
	return addresses, rows.Err()
}

func (s *PostgresStore) CreateAddress(ctx context.Context, address Address) (Address, error) {
	if address.ID == "" {
		address.ID = uuid.NewString()
	}
	if address.UserAddress == "" {
		address.UserAddress = address.Address
	}
	if len(address.GroupIDs) == 0 {
		address.GroupIDs = []string{"default"}
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO followed_addresses (id, address, remark_name, group_ids, owner_user_id, user_address, profit_1d, profit_7d, profit_30d, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, now(), now())
		RETURNING id, address, remark_name, group_ids, COALESCE(owner_user_id, ''), user_address, profit_1d::text, profit_7d::text, profit_30d::text, created_at, updated_at
	`, address.ID, address.Address, address.RemarkName, address.GroupIDs, address.OwnerUserID, address.UserAddress, decimal(address.Profit1d), decimal(address.Profit7d), decimal(address.Profit30d)).Scan(
		&address.ID, &address.Address, &address.RemarkName, &address.GroupIDs, &address.OwnerUserID, &address.UserAddress, &address.Profit1d, &address.Profit7d, &address.Profit30d, &address.CreatedAt, &address.UpdatedAt,
	)
	if err != nil {
		return Address{}, fmt.Errorf("create followed address: %w", err)
	}
	return address, nil
}

func (s *PostgresStore) UpdateAddress(ctx context.Context, address Address) (Address, error) {
	existing, err := s.getAddress(ctx, address.ID)
	if err != nil {
		return Address{}, err
	}
	if strings.TrimSpace(address.RemarkName) != "" {
		existing.RemarkName = address.RemarkName
	}
	if len(address.GroupIDs) > 0 {
		existing.GroupIDs = address.GroupIDs
	}
	err = s.pool.QueryRow(ctx, `
		UPDATE followed_addresses
		SET remark_name = $2, group_ids = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, address, remark_name, group_ids, COALESCE(owner_user_id, ''), user_address, profit_1d::text, profit_7d::text, profit_30d::text, created_at, updated_at
	`, existing.ID, existing.RemarkName, existing.GroupIDs).Scan(
		&existing.ID, &existing.Address, &existing.RemarkName, &existing.GroupIDs, &existing.OwnerUserID, &existing.UserAddress, &existing.Profit1d, &existing.Profit7d, &existing.Profit30d, &existing.CreatedAt, &existing.UpdatedAt,
	)
	if err != nil {
		return Address{}, fmt.Errorf("update followed address: %w", err)
	}
	return existing, nil
}

func (s *PostgresStore) DeleteAddress(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM followed_addresses WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete followed address: %w", err)
	}
	return nil
}

func (s *PostgresStore) SaveOpenOrdersSnapshot(ctx context.Context, userAddress string, orders []OpenOrder) error {
	userAddress = strings.TrimSpace(userAddress)
	if userAddress == "" {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin open order snapshot: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM hyper_open_order_snapshots WHERE lower(user_address) = lower($1)`, userAddress); err != nil {
		return fmt.Errorf("clear open order snapshot: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO hyper_open_order_snapshot_meta (user_address, updated_at)
		VALUES ($1, now())
		ON CONFLICT (user_address)
		DO UPDATE SET updated_at = EXCLUDED.updated_at
	`, userAddress); err != nil {
		return fmt.Errorf("save open order snapshot meta: %w", err)
	}
	for _, order := range orders {
		order.UserAddress = firstNonEmpty(order.UserAddress, userAddress)
		id := firstNonEmpty(order.ProviderOrderID, order.Cloid, order.ID)
		if id == "" {
			continue
		}
		order.ID = firstNonEmpty(order.ID, id)
		order.ProviderOrderID = firstNonEmpty(order.ProviderOrderID, id)
		payload, err := json.Marshal(order)
		if err != nil {
			return fmt.Errorf("marshal open order snapshot: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO hyper_open_order_snapshots (user_address, provider_order_id, order_payload, status, updated_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (user_address, provider_order_id)
			DO UPDATE SET order_payload = EXCLUDED.order_payload, status = EXCLUDED.status, updated_at = EXCLUDED.updated_at
		`, userAddress, id, payload, firstNonEmpty(order.Status, "open")); err != nil {
			return fmt.Errorf("save open order snapshot: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit open order snapshot: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListOpenOrdersSnapshot(ctx context.Context, userAddress string) ([]OpenOrder, error) {
	userAddress = strings.TrimSpace(userAddress)
	if userAddress == "" {
		return nil, ErrNotFound
	}
	var hasSnapshot bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM hyper_open_order_snapshot_meta
			WHERE lower(user_address) = lower($1)
		)
	`, userAddress).Scan(&hasSnapshot); err != nil {
		return nil, fmt.Errorf("check open order snapshot: %w", err)
	}
	if !hasSnapshot {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `
		SELECT order_payload
		FROM hyper_open_order_snapshots
		WHERE lower(user_address) = lower($1)
		ORDER BY updated_at DESC
	`, userAddress)
	if err != nil {
		return nil, fmt.Errorf("list open order snapshot: %w", err)
	}
	defer rows.Close()
	out := make([]OpenOrder, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan open order snapshot: %w", err)
		}
		var order OpenOrder
		if err := json.Unmarshal(payload, &order); err == nil {
			out = append(out, order)
		}
	}
	return out, rows.Err()
}

func (s *PostgresStore) AppendFills(ctx context.Context, userAddress string, fills []TradeHistory) error {
	userAddress = strings.TrimSpace(userAddress)
	if userAddress == "" {
		return nil
	}
	for _, fill := range fills {
		fillID := ownerKey("", userAddress) + ":" + fillKey(fill)
		payload, err := json.Marshal(fill)
		if err != nil {
			return fmt.Errorf("marshal fill snapshot: %w", err)
		}
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO hyper_fills (id, user_address, symbol, provider_order_id, fill_time, fill_payload, created_at, updated_at)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, now(), now())
			ON CONFLICT (id)
			DO UPDATE SET fill_payload = EXCLUDED.fill_payload, updated_at = EXCLUDED.updated_at
		`, fillID, userAddress, fill.Symbol, fmt.Sprintf("%d", fill.Oid), fill.Time, payload); err != nil {
			return fmt.Errorf("append hyper fill: %w", err)
		}
	}
	return nil
}

func (s *PostgresStore) ListFills(ctx context.Context, userAddress string, limit int) ([]TradeHistory, error) {
	if limit <= 0 || limit > 2000 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT fill_payload
		FROM hyper_fills
		WHERE lower(user_address) = lower($1)
		ORDER BY fill_time DESC, updated_at DESC
		LIMIT $2
	`, strings.TrimSpace(userAddress), limit)
	if err != nil {
		return nil, fmt.Errorf("list hyper fills: %w", err)
	}
	defer rows.Close()
	out := make([]TradeHistory, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan hyper fill: %w", err)
		}
		var fill TradeHistory
		if err := json.Unmarshal(payload, &fill); err == nil {
			out = append(out, fill)
		}
	}
	return out, rows.Err()
}

func (s *PostgresStore) SaveAccountSnapshot(ctx context.Context, userAddress string, account AccountBalance) error {
	userAddress = strings.TrimSpace(userAddress)
	if userAddress == "" {
		return nil
	}
	payload, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("marshal account snapshot: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO hyper_account_snapshots (user_address, account_payload, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (user_address)
		DO UPDATE SET account_payload = EXCLUDED.account_payload, updated_at = EXCLUDED.updated_at
	`, userAddress, payload)
	if err != nil {
		return fmt.Errorf("save account snapshot: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetAccountSnapshot(ctx context.Context, userAddress string) (AccountBalance, error) {
	var payload []byte
	err := s.pool.QueryRow(ctx, `
		SELECT account_payload
		FROM hyper_account_snapshots
		WHERE lower(user_address) = lower($1)
	`, strings.TrimSpace(userAddress)).Scan(&payload)
	if stderrors.Is(err, pgx.ErrNoRows) {
		return AccountBalance{}, ErrNotFound
	}
	if err != nil {
		return AccountBalance{}, fmt.Errorf("get account snapshot: %w", err)
	}
	var account AccountBalance
	if err := json.Unmarshal(payload, &account); err != nil {
		return AccountBalance{}, fmt.Errorf("decode account snapshot: %w", err)
	}
	return account, nil
}

func (s *PostgresStore) UpdateOrderStatusByProvider(ctx context.Context, input OrderStatusInput, status OrderStatus) error {
	providerOrderID := firstNonEmpty(status.ProviderOrderID, input.ProviderOrderID)
	cloid := firstNonEmpty(status.Cloid, input.Cloid)
	if providerOrderID == "" && cloid == "" {
		return nil
	}
	payload, err := json.Marshal(map[string]any{"wsOrderStatus": orderStatusPayload(status)})
	if err != nil {
		return fmt.Errorf("marshal ws order status: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE hyper_orders
		SET status = CASE
		        WHEN status IN ('cancelled', 'filled', 'failed', 'rejected') AND $5 = 'submitted' THEN status
		        ELSE COALESCE(NULLIF($5, ''), status)
		    END,
		    provider_order_id = COALESCE(NULLIF($3, ''), provider_order_id),
		    response_payload = response_payload || $6::jsonb,
		    cancelled_at = CASE WHEN $5 = 'cancelled' THEN COALESCE(cancelled_at, now()) ELSE cancelled_at END,
		    updated_at = now()
		WHERE ($1 = '' OR lower(COALESCE(user_address, '')) = lower($1) OR user_id = $2)
		  AND (($3 <> '' AND provider_order_id = $3) OR ($4 <> '' AND cloid = $4))
	`, strings.TrimSpace(input.UserAddress), strings.TrimSpace(input.UserID), providerOrderID, cloid, normalizeProviderOrderStatus(status.Status), payload)
	if err != nil {
		return fmt.Errorf("update order status by provider: %w", err)
	}
	return nil
}

func (s *PostgresStore) getAddress(ctx context.Context, id string) (Address, error) {
	address, err := scanAddress(s.pool.QueryRow(ctx, `
		SELECT id, address, remark_name, group_ids, COALESCE(owner_user_id, ''), user_address, profit_1d::text, profit_7d::text, profit_30d::text, created_at, updated_at
		FROM followed_addresses
		WHERE id = $1
	`, id))
	if stderrors.Is(err, pgx.ErrNoRows) {
		return Address{}, ErrNotFound
	}
	if err != nil {
		return Address{}, fmt.Errorf("get followed address: %w", err)
	}
	return address, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSymbol(row rowScanner) (Symbol, error) {
	var symbol Symbol
	err := row.Scan(&symbol.Symbol, &symbol.AliasName, &symbol.MaxLeverage, &symbol.MarketCap, &symbol.Volume, &symbol.ChangePercent, &symbol.OpenInterest, &symbol.CurrentPrice, &symbol.Type, &symbol.QuoteSymbol, &symbol.Category, &symbol.CreatedAt)
	if err != nil {
		return Symbol{}, err
	}
	return symbol, nil
}

func scanTrader(row rowScanner) (SmartMoneyTrader, error) {
	var trader SmartMoneyTrader
	var portfolioData []byte
	var lastOperation []byte
	var tags []byte
	err := row.Scan(
		&trader.UserAddress, &trader.ROI, &trader.NetPnL, &trader.AvgWinRate, &trader.MaxDrawdown, &trader.PeriodDays, &trader.SharpeRatio, &trader.ProfitLossRatio, &trader.ProfitFactor,
		&trader.TotalVolume, &trader.AvgDailyVolume, &trader.TradingDays, &trader.TotalTrades, &trader.UniqueCoinsCount, &trader.AvgTradesPerDay,
		&trader.TotalLongPnL, &trader.TotalShortPnL, &trader.WinningPnLTotal, &trader.LosingPnLTotal,
		&trader.KOLLabels, &trader.KOLLabelsDescription, &trader.FollowerCount, &trader.RemarkName, &trader.GroupIDs, &portfolioData, &lastOperation, &tags,
	)
	if err != nil {
		return SmartMoneyTrader{}, err
	}
	_ = json.Unmarshal(portfolioData, &trader.PortfolioData)
	_ = json.Unmarshal(lastOperation, &trader.LastOperation)
	_ = json.Unmarshal(tags, &trader.Tags)
	return trader, nil
}

func scanGroup(row rowScanner) (AddressGroup, error) {
	var group AddressGroup
	err := row.Scan(&group.ID, &group.Name, &group.UserID, &group.IsDefault, &group.Order, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return AddressGroup{}, err
	}
	return group, nil
}

func scanAddress(row rowScanner) (Address, error) {
	var address Address
	err := row.Scan(&address.ID, &address.Address, &address.RemarkName, &address.GroupIDs, &address.OwnerUserID, &address.UserAddress, &address.Profit1d, &address.Profit7d, &address.Profit30d, &address.CreatedAt, &address.UpdatedAt)
	if err != nil {
		return Address{}, err
	}
	return address, nil
}

func scanFuturesOrder(row rowScanner) (FuturesOrder, error) {
	var order FuturesOrder
	var rawPayload []byte
	var responsePayload []byte
	err := row.Scan(
		&order.ID,
		&order.UserID,
		&order.UserAddress,
		&order.Symbol,
		&order.Side,
		&order.OrderType,
		&order.Price,
		&order.Size,
		&order.Status,
		&order.Cloid,
		&rawPayload,
		&order.Provider,
		&order.ProviderOrderID,
		&order.ClientRequestID,
		&order.ReduceOnly,
		&order.TimeInForce,
		&responsePayload,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.SubmittedAt,
		&order.CancelledAt,
	)
	if err != nil {
		return FuturesOrder{}, err
	}
	_ = json.Unmarshal(rawPayload, &order.RawPayload)
	_ = json.Unmarshal(responsePayload, &order.ResponsePayload)
	return order, nil
}

func decimal(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "0"
	}
	return value
}

var _ Store = (*PostgresStore)(nil)
var _ StateStore = (*PostgresStore)(nil)
