package hypertrader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type HTTPProvider struct {
	baseURL string
	client  *http.Client
	now     func() time.Time
}

func NewHTTPProvider(baseURL string, timeout time.Duration) *HTTPProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.hyperliquid.xyz"
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &HTTPProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
		now:     time.Now,
	}
}

func (p *HTTPProvider) Name() string {
	return "hyperliquid-http"
}

func (p *HTTPProvider) WalletStatus(ctx context.Context, userAddress string) (HyperliquidWalletStatus, error) {
	if strings.TrimSpace(userAddress) == "" {
		return HyperliquidWalletStatus{}, fmt.Errorf("userAddress is required for http provider wallet status")
	}
	var raw map[string]any
	if err := p.info(ctx, map[string]any{"type": "clearinghouseState", "user": userAddress}, &raw); err != nil {
		return HyperliquidWalletStatus{}, err
	}
	return HyperliquidWalletStatus{
		ApprovedAgent:     true,
		SetReferral:       raw["marginSummary"] != nil,
		SetFeeBuilder:     false,
		Agent:             "",
		AgentName:         "Hyperliquid HTTP",
		FeeBuilderAddress: "",
		FeeBuilderPercent: "0",
		ReferralCode:      "",
	}, nil
}

func (p *HTTPProvider) Account(ctx context.Context, userAddress string) (AccountBalance, error) {
	if strings.TrimSpace(userAddress) == "" {
		return AccountBalance{}, fmt.Errorf("userAddress is required")
	}
	var raw map[string]any
	if err := p.info(ctx, map[string]any{"type": "clearinghouseState", "user": userAddress}, &raw); err != nil {
		return AccountBalance{}, err
	}
	return accountFromClearinghouseState(userAddress, raw, p.now().UTC()), nil
}

func (p *HTTPProvider) Sign(_ context.Context, _ string, _ string, _ map[string]any) (ProviderActionResult, error) {
	return ProviderActionResult{}, fmt.Errorf("http provider cannot create Hyperliquid signatures; pass signed exchange payloads from wallet or agent signer")
}

func (p *HTTPProvider) SubmitOrder(ctx context.Context, order FuturesOrder) (ProviderActionResult, error) {
	payload, err := signedExchangePayload(order.RawPayload)
	if err != nil {
		return ProviderActionResult{}, err
	}
	var raw map[string]any
	if err := p.exchange(ctx, payload, &raw); err != nil {
		return ProviderActionResult{}, err
	}
	return providerActionFromHTTP("submitOrder", p.Name(), raw, payload, p.now().UTC()), nil
}

func (p *HTTPProvider) CancelOrder(ctx context.Context, input CancelOrderInput) (ProviderActionResult, error) {
	payload, err := signedExchangePayload(map[string]any{"exchangePayload": input.ExchangePayload})
	if err != nil {
		return ProviderActionResult{}, err
	}
	var raw map[string]any
	if err := p.exchange(ctx, payload, &raw); err != nil {
		return ProviderActionResult{}, err
	}
	return providerActionFromHTTP("cancelOrder", p.Name(), raw, payload, p.now().UTC()), nil
}

func (p *HTTPProvider) UpdateLeverage(ctx context.Context, input UpdateLeverageInput) (ProviderActionResult, error) {
	payload, err := signedExchangePayload(map[string]any{"exchangePayload": input.ExchangePayload})
	if err != nil {
		return ProviderActionResult{}, err
	}
	var raw map[string]any
	if err := p.exchange(ctx, payload, &raw); err != nil {
		return ProviderActionResult{}, err
	}
	return providerActionFromHTTP("updateLeverage", p.Name(), raw, payload, p.now().UTC()), nil
}

func (p *HTTPProvider) FundingRates(ctx context.Context, symbol string, limit int) ([]FundingRate, error) {
	if limit <= 0 || limit > 100 {
		limit = 8
	}
	symbols := []string{"BTC", "ETH", "SOL", "HYPE"}
	if strings.TrimSpace(symbol) != "" {
		symbols = []string{strings.ToUpper(strings.TrimSpace(symbol))}
	}
	out := make([]FundingRate, 0, minInt(limit, len(symbols)))
	startTime := p.now().Add(-24 * time.Hour).UnixMilli()
	for _, coin := range symbols {
		if len(out) >= limit {
			break
		}
		var raw []map[string]any
		if err := p.info(ctx, map[string]any{"type": "fundingHistory", "coin": coin, "startTime": startTime}, &raw); err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			continue
		}
		item := raw[len(raw)-1]
		out = append(out, FundingRate{
			Symbol:          firstNonEmpty(stringFromAny(item["coin"]), coin),
			FundingRate:     stringFromAny(item["fundingRate"]),
			Premium:         stringFromAny(item["premium"]),
			NextFundingTime: int64FromAny(item["time"]),
			UpdatedAt:       p.now().UTC(),
		})
	}
	return out, nil
}

func (p *HTTPProvider) info(ctx context.Context, body map[string]any, out any) error {
	return p.post(ctx, "/info", body, out)
}

func (p *HTTPProvider) exchange(ctx context.Context, body map[string]any, out any) error {
	return p.post(ctx, "/exchange", body, out)
}

func (p *HTTPProvider) post(ctx context.Context, path string, body any, out any) error {
	rawBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal hyperliquid request: %w", err)
	}
	endpoint, err := url.JoinPath(p.baseURL, path)
	if err != nil {
		return fmt.Errorf("build hyperliquid url: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawBody))
	if err != nil {
		return fmt.Errorf("create hyperliquid request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("hyperliquid request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var payload map[string]any
		_ = json.NewDecoder(res.Body).Decode(&payload)
		return fmt.Errorf("hyperliquid %s returned %d: %v", path, res.StatusCode, payload)
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("decode hyperliquid response: %w", err)
	}
	return nil
}

func signedExchangePayload(input map[string]any) (map[string]any, error) {
	if payload := asMap(input["exchangePayload"]); len(payload) > 0 {
		return requireSignedPayload(payload)
	}
	if input["action"] != nil || input["signature"] != nil || input["nonce"] != nil {
		return requireSignedPayload(input)
	}
	return nil, fmt.Errorf("signed exchangePayload is required for http provider writes")
}

func requireSignedPayload(payload map[string]any) (map[string]any, error) {
	if payload["action"] == nil {
		return nil, fmt.Errorf("exchangePayload.action is required")
	}
	if payload["signature"] == nil {
		return nil, fmt.Errorf("exchangePayload.signature is required")
	}
	if payload["nonce"] == nil {
		return nil, fmt.Errorf("exchangePayload.nonce is required")
	}
	return payload, nil
}

func providerActionFromHTTP(action string, provider string, raw map[string]any, request map[string]any, now time.Time) ProviderActionResult {
	status := stringFromAny(raw["status"])
	if status == "" {
		status = "submitted"
	}
	return ProviderActionResult{
		Action:      action,
		Provider:    provider,
		RequestID:   firstNonEmpty(stringFromAny(raw["hash"]), stringFromAny(raw["requestId"]), uuid.NewString()),
		Status:      status,
		RawPayload:  map[string]any{"request": request, "response": raw},
		SubmittedAt: now,
	}
}

func accountFromClearinghouseState(userAddress string, raw map[string]any, now time.Time) AccountBalance {
	margin := asMap(raw["marginSummary"])
	cross := asMap(raw["crossMarginSummary"])
	accountValue := firstNonEmpty(stringFromAny(margin["accountValue"]), stringFromAny(cross["accountValue"]))
	totalRawUSD := firstNonEmpty(stringFromAny(margin["totalRawUsd"]), stringFromAny(cross["totalRawUsd"]), accountValue)
	assetPositions, _ := raw["assetPositions"].([]any)
	positions := make([]Position, 0, len(assetPositions))
	for _, item := range assetPositions {
		position := asMap(asMap(item)["position"])
		if len(position) == 0 {
			continue
		}
		leverage := asMap(position["leverage"])
		coin := stringFromAny(position["coin"])
		szi := stringFromAny(position["szi"])
		side := "long"
		if strings.HasPrefix(szi, "-") {
			side = "short"
		}
		positions = append(positions, Position{
			Address:                    userAddress,
			Coin:                       coin,
			CreatedAt:                  now,
			UpdatedAt:                  now,
			PositionType:               side,
			Szi:                        szi,
			LeverageType:               stringFromAny(leverage["type"]),
			LeverageValue:              int(int64FromAny(leverage["value"])),
			EntryPx:                    stringFromAny(position["entryPx"]),
			PositionValue:              stringFromAny(position["positionValue"]),
			UnrealizedPnl:              stringFromAny(position["unrealizedPnl"]),
			ReturnOnEquity:             stringFromAny(position["returnOnEquity"]),
			LiquidationPx:              stringFromAny(position["liquidationPx"]),
			MarginUsed:                 stringFromAny(position["marginUsed"]),
			MaxLeverage:                int(int64FromAny(position["maxLeverage"])),
			OpenTime:                   int64FromAny(position["openTime"]),
			CumFundingAllTime:          stringFromAny(asMap(position["cumFunding"])["allTime"]),
			CumFundingSinceOpen:        stringFromAny(asMap(position["cumFunding"])["sinceOpen"]),
			CumFundingSinceChange:      stringFromAny(asMap(position["cumFunding"])["sinceChange"]),
			AccountValue:               accountValue,
			CrossMaintenanceMarginUsed: stringFromAny(raw["crossMaintenanceMarginUsed"]),
			CrossMarginRatio:           "",
			Side:                       side,
			Time:                       int64FromAny(raw["time"]),
			StartPosition:              szi,
			Dir:                        "Open " + strings.Title(side),
			Hash:                       "",
			Crossed:                    strings.EqualFold(stringFromAny(leverage["type"]), "cross"),
		})
	}
	return AccountBalance{
		Balance:             accountValue,
		OneDayChange:        "0",
		OneDayPercentChange: "0",
		RawUSD:              totalRawUSD,
		Positions:           positions,
	}
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return fmt.Sprintf("%v", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		if value == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func int64FromAny(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		parsed, _ := v.Int64()
		return parsed
	case string:
		var parsed int64
		_, _ = fmt.Sscan(v, &parsed)
		return parsed
	default:
		return 0
	}
}

var _ Provider = (*HTTPProvider)(nil)
