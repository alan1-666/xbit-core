package hypertrader

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/xbit/xbit-backend/internal/streambridge"
)

const hyperliquidWSProvider = "hyperliquid-ws"

type StreamBridgeConfig struct {
	WSURL          string
	Users          []string
	Dex            string
	Subscriptions  []string
	ReconnectDelay time.Duration
	Logger         *slog.Logger
}

type wsConn interface {
	WriteJSON(v any) error
	ReadJSON(v any) error
	Close() error
}

type wsDialFunc func(ctx context.Context, url string) (wsConn, error)

type HyperliquidStreamBridge struct {
	cfg    StreamBridgeConfig
	stream *streambridge.Service
	dial   wsDialFunc
	now    func() time.Time
}

func NewHyperliquidStreamBridge(cfg StreamBridgeConfig, stream *streambridge.Service) *HyperliquidStreamBridge {
	if strings.TrimSpace(cfg.WSURL) == "" {
		cfg.WSURL = "wss://api.hyperliquid.xyz/ws"
	}
	if cfg.ReconnectDelay <= 0 {
		cfg.ReconnectDelay = 3 * time.Second
	}
	if len(cfg.Subscriptions) == 0 {
		cfg.Subscriptions = []string{"orderUpdates", "userEvents", "userFills", "userFundings", "userNonFundingLedgerUpdates", "openOrders", "clearinghouseState"}
	}
	cfg.Users = normalizeUsers(cfg.Users)
	return &HyperliquidStreamBridge{
		cfg:    cfg,
		stream: stream,
		dial:   dialHyperliquidWS,
		now:    time.Now,
	}
}

func (b *HyperliquidStreamBridge) Run(ctx context.Context) {
	if b == nil || b.stream == nil {
		return
	}
	if len(b.cfg.Users) == 0 {
		b.logInfo("hyperliquid ws bridge disabled: no users configured")
		return
	}

	var wg sync.WaitGroup
	for _, user := range b.cfg.Users {
		user := user
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.runUserLoop(ctx, user)
		}()
	}
	wg.Wait()
}

func (b *HyperliquidStreamBridge) runUserLoop(ctx context.Context, user string) {
	for {
		if err := b.runUserOnce(ctx, user); err != nil && ctx.Err() == nil {
			b.logWarn("hyperliquid ws disconnected", "user", user, "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(b.cfg.ReconnectDelay):
		}
	}
}

func (b *HyperliquidStreamBridge) runUserOnce(ctx context.Context, user string) error {
	conn, err := b.dial(ctx, b.cfg.WSURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	closed := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-closed:
		}
	}()
	defer close(closed)

	for _, subType := range b.cfg.Subscriptions {
		if err := conn.WriteJSON(map[string]any{"method": "subscribe", "subscription": b.subscription(user, subType)}); err != nil {
			return fmt.Errorf("subscribe %s: %w", subType, err)
		}
	}
	b.logInfo("hyperliquid ws subscribed", "user", user, "subscriptions", strings.Join(b.cfg.Subscriptions, ","))

	for {
		var message map[string]any
		if err := conn.ReadJSON(&message); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if err := b.handleMessage(ctx, user, message); err != nil {
			b.logWarn("hyperliquid ws message skipped", "user", user, "error", err)
		}
	}
}

func (b *HyperliquidStreamBridge) subscription(user string, subType string) map[string]any {
	sub := map[string]any{"type": subType, "user": user}
	if b.cfg.Dex != "" && (subType == "openOrders" || subType == "clearinghouseState") {
		sub["dex"] = b.cfg.Dex
	}
	return sub
}

func (b *HyperliquidStreamBridge) handleMessage(ctx context.Context, defaultUser string, message map[string]any) error {
	channel := stringFromAny(message["channel"])
	switch channel {
	case "", "subscriptionResponse", "pong":
		return nil
	case "orderUpdates":
		return b.publishOrderUpdates(ctx, defaultUser, message["data"])
	case "userFills":
		return b.publishUserFills(ctx, defaultUser, message["data"])
	case "userEvents":
		return b.publishUserEvents(ctx, defaultUser, message["data"])
	case "userFundings":
		return b.publishUserFundings(ctx, defaultUser, message["data"])
	case "userNonFundingLedgerUpdates":
		return b.publishLedgerUpdates(ctx, defaultUser, message["data"])
	case "openOrders":
		return b.publishOpenOrders(ctx, defaultUser, message["data"])
	case "clearinghouseState":
		return b.publishClearinghouseState(ctx, defaultUser, message["data"])
	default:
		return b.publish(ctx, streambridge.Event{
			Type:        streambridge.EventHypertraderRawEvent,
			Source:      hyperliquidWSProvider,
			UserID:      defaultUser,
			AggregateID: channel,
			Payload: map[string]any{
				"provider": hyperliquidWSProvider,
				"channel":  channel,
				"data":     message["data"],
			},
		})
	}
}

func (b *HyperliquidStreamBridge) publishOrderUpdates(ctx context.Context, user string, data any) error {
	for _, item := range anySlice(data) {
		raw := asMap(item)
		order := asMap(raw["order"])
		status := stringFromAny(raw["status"])
		aggregateID := firstNonEmpty(stringFromAny(order["oid"]), stringFromAny(raw["oid"]))
		payload := map[string]any{
			"provider":        hyperliquidWSProvider,
			"userAddress":     user,
			"providerOrderId": aggregateID,
			"symbol":          stringFromAny(order["coin"]),
			"side":            hyperliquidSide(stringFromAny(order["side"])),
			"price":           firstNonEmpty(stringFromAny(order["limitPx"]), stringFromAny(order["px"])),
			"size":            stringFromAny(order["sz"]),
			"originalSize":    stringFromAny(order["origSz"]),
			"cloid":           stringFromAny(order["cloid"]),
			"status":          normalizeProviderOrderStatus(status),
			"providerStatus":  status,
			"rawPayload":      raw,
		}
		if err := b.publish(ctx, streambridge.Event{
			Type:        streambridge.EventHypertraderOrderUpdated,
			Source:      hyperliquidWSProvider,
			UserID:      user,
			AggregateID: aggregateID,
			Payload:     payload,
			CreatedAt:   eventTime(raw, b.now().UTC()),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *HyperliquidStreamBridge) publishUserFills(ctx context.Context, defaultUser string, data any) error {
	raw := asMap(data)
	user := firstNonEmpty(stringFromAny(raw["user"]), defaultUser)
	isSnapshot := boolValue(raw, false, "isSnapshot")
	for _, item := range anySlice(raw["fills"]) {
		fill := asMap(item)
		payload := fillPayload(fill, user, isSnapshot)
		if err := b.publish(ctx, streambridge.Event{
			Type:        streambridge.EventHypertraderFillCreated,
			Source:      hyperliquidWSProvider,
			UserID:      user,
			AggregateID: fillAggregateID(fill),
			Payload:     payload,
			CreatedAt:   eventTime(fill, b.now().UTC()),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *HyperliquidStreamBridge) publishUserEvents(ctx context.Context, defaultUser string, data any) error {
	raw := asMap(data)
	if fills := anySlice(raw["fills"]); len(fills) > 0 {
		for _, item := range fills {
			fill := asMap(item)
			if err := b.publish(ctx, streambridge.Event{
				Type:        streambridge.EventHypertraderFillCreated,
				Source:      hyperliquidWSProvider,
				UserID:      defaultUser,
				AggregateID: fillAggregateID(fill),
				Payload:     fillPayload(fill, defaultUser, false),
				CreatedAt:   eventTime(fill, b.now().UTC()),
			}); err != nil {
				return err
			}
		}
	}
	if funding := asMap(raw["funding"]); len(funding) > 0 {
		if err := b.publish(ctx, streambridge.Event{
			Type:        streambridge.EventHypertraderFundingUpdated,
			Source:      hyperliquidWSProvider,
			UserID:      defaultUser,
			AggregateID: stringFromAny(funding["coin"]),
			Payload: map[string]any{
				"provider":    hyperliquidWSProvider,
				"userAddress": defaultUser,
				"funding":     funding,
			},
			CreatedAt: eventTime(funding, b.now().UTC()),
		}); err != nil {
			return err
		}
	}
	if liquidation := asMap(raw["liquidation"]); len(liquidation) > 0 {
		if err := b.publish(ctx, streambridge.Event{
			Type:        streambridge.EventHypertraderPositionUpdated,
			Source:      hyperliquidWSProvider,
			UserID:      defaultUser,
			AggregateID: stringFromAny(liquidation["lid"]),
			Payload: map[string]any{
				"provider":    hyperliquidWSProvider,
				"userAddress": defaultUser,
				"liquidation": liquidation,
			},
			CreatedAt: b.now().UTC(),
		}); err != nil {
			return err
		}
	}
	if cancels := anySlice(raw["nonUserCancel"]); len(cancels) > 0 {
		return b.publish(ctx, streambridge.Event{
			Type:        streambridge.EventHypertraderOrderUpdated,
			Source:      hyperliquidWSProvider,
			UserID:      defaultUser,
			AggregateID: "non-user-cancel",
			Payload: map[string]any{
				"provider":    hyperliquidWSProvider,
				"userAddress": defaultUser,
				"status":      "cancelled",
				"cancels":     cancels,
			},
			CreatedAt: b.now().UTC(),
		})
	}
	return nil
}

func (b *HyperliquidStreamBridge) publishOpenOrders(ctx context.Context, defaultUser string, data any) error {
	raw := asMap(data)
	user := firstNonEmpty(stringFromAny(raw["user"]), defaultUser)
	now := b.now().UTC()
	orders := make([]map[string]any, 0)
	for _, item := range anySlice(raw["orders"]) {
		order := openOrderFromHTTP(user, hyperliquidWSProvider, asMap(item), now)
		orders = append(orders, graphQLOpenOrder(order))
	}
	return b.publish(ctx, streambridge.Event{
		Type:        streambridge.EventHypertraderOpenOrders,
		Source:      hyperliquidWSProvider,
		UserID:      user,
		AggregateID: firstNonEmpty(stringFromAny(raw["dex"]), "default"),
		Payload: map[string]any{
			"provider":    hyperliquidWSProvider,
			"userAddress": user,
			"dex":         stringFromAny(raw["dex"]),
			"orders":      orders,
			"rawPayload":  raw,
		},
		CreatedAt: now,
	})
}

func (b *HyperliquidStreamBridge) publishUserFundings(ctx context.Context, defaultUser string, data any) error {
	raw := asMap(data)
	user := firstNonEmpty(stringFromAny(raw["user"]), defaultUser)
	isSnapshot := boolValue(raw, false, "isSnapshot")
	items := anySlice(raw["fundings"])
	if len(items) == 0 {
		items = anySlice(raw["funding"])
	}
	for _, item := range items {
		funding := asMap(item)
		if err := b.publish(ctx, streambridge.Event{
			Type:        streambridge.EventHypertraderFundingUpdated,
			Source:      hyperliquidWSProvider,
			UserID:      user,
			AggregateID: stringFromAny(funding["coin"]),
			Payload: map[string]any{
				"provider":    hyperliquidWSProvider,
				"userAddress": user,
				"isSnapshot":  isSnapshot,
				"funding":     funding,
			},
			CreatedAt: eventTime(funding, b.now().UTC()),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *HyperliquidStreamBridge) publishLedgerUpdates(ctx context.Context, defaultUser string, data any) error {
	raw := asMap(data)
	user := firstNonEmpty(stringFromAny(raw["user"]), defaultUser)
	isSnapshot := boolValue(raw, false, "isSnapshot")
	items := anySlice(raw["updates"])
	if len(items) == 0 {
		items = anySlice(raw["nonFundingLedgerUpdates"])
	}
	for _, item := range items {
		update := asMap(item)
		if err := b.publish(ctx, streambridge.Event{
			Type:        streambridge.EventHypertraderLedgerUpdated,
			Source:      hyperliquidWSProvider,
			UserID:      user,
			AggregateID: firstNonEmpty(stringFromAny(update["hash"]), stringFromAny(update["time"])),
			Payload: map[string]any{
				"provider":     hyperliquidWSProvider,
				"userAddress":  user,
				"isSnapshot":   isSnapshot,
				"ledgerUpdate": update,
			},
			CreatedAt: eventTime(update, b.now().UTC()),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *HyperliquidStreamBridge) publishClearinghouseState(ctx context.Context, defaultUser string, data any) error {
	raw := asMap(data)
	user := firstNonEmpty(stringFromAny(raw["user"]), defaultUser)
	account := accountFromClearinghouseState(user, raw, b.now().UTC())
	if err := b.publish(ctx, streambridge.Event{
		Type:        streambridge.EventHypertraderAccountUpdated,
		Source:      hyperliquidWSProvider,
		UserID:      user,
		AggregateID: user,
		Payload: map[string]any{
			"provider":    hyperliquidWSProvider,
			"userAddress": user,
			"balance":     account.Balance,
			"rawUSD":      account.RawUSD,
			"positions":   graphQLPositions(account.Positions),
			"rawPayload":  raw,
		},
		CreatedAt: b.now().UTC(),
	}); err != nil {
		return err
	}
	for _, position := range account.Positions {
		if err := b.publish(ctx, streambridge.Event{
			Type:        streambridge.EventHypertraderPositionUpdated,
			Source:      hyperliquidWSProvider,
			UserID:      user,
			AggregateID: position.Coin,
			Payload: map[string]any{
				"provider":    hyperliquidWSProvider,
				"userAddress": user,
				"position":    graphQLPositions([]Position{position})[0],
			},
			CreatedAt: b.now().UTC(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *HyperliquidStreamBridge) publish(ctx context.Context, event streambridge.Event) error {
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}
	_, err := b.stream.Publish(ctx, event)
	return err
}

func fillPayload(fill map[string]any, user string, isSnapshot bool) map[string]any {
	trade := tradeHistoryFromFill(fill)
	return map[string]any{
		"provider":        hyperliquidWSProvider,
		"userAddress":     user,
		"symbol":          trade.Symbol,
		"side":            hyperliquidSide(stringFromAny(fill["side"])),
		"price":           trade.Px,
		"size":            trade.Sz,
		"closedPnl":       trade.PnL,
		"dir":             trade.Dir,
		"hash":            trade.Hash,
		"providerOrderId": stringFromAny(fill["oid"]),
		"tid":             trade.Tid,
		"fee":             trade.Fee,
		"feeToken":        trade.FeeToken,
		"isSnapshot":      isSnapshot,
		"rawPayload":      fill,
	}
}

func fillAggregateID(fill map[string]any) string {
	hash := stringFromAny(fill["hash"])
	tid := stringFromAny(fill["tid"])
	if hash != "" && tid != "" {
		return hash + ":" + tid
	}
	return firstNonEmpty(hash, tid, stringFromAny(fill["oid"]))
}

func eventTime(raw map[string]any, fallback time.Time) time.Time {
	if ts := int64FromAny(raw["statusTimestamp"]); ts > 0 {
		return time.UnixMilli(ts).UTC()
	}
	if ts := int64FromAny(raw["time"]); ts > 0 {
		return time.UnixMilli(ts).UTC()
	}
	if ts := int64FromAny(raw["timestamp"]); ts > 0 {
		return time.UnixMilli(ts).UTC()
	}
	return fallback
}

func anySlice(value any) []any {
	switch v := value.(type) {
	case []any:
		return v
	case []map[string]any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	case nil:
		return nil
	default:
		return []any{v}
	}
}

func normalizeUsers(users []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(users))
	for _, user := range users {
		user = strings.TrimSpace(user)
		if user == "" {
			continue
		}
		key := strings.ToLower(user)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, user)
	}
	return out
}

func dialHyperliquidWS(ctx context.Context, url string) (wsConn, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (b *HyperliquidStreamBridge) logInfo(message string, args ...any) {
	if b.cfg.Logger != nil {
		b.cfg.Logger.Info(message, args...)
	}
}

func (b *HyperliquidStreamBridge) logWarn(message string, args ...any) {
	if b.cfg.Logger != nil {
		b.cfg.Logger.Warn(message, args...)
	}
}
