package hypertrader

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Provider interface {
	Name() string
	WalletStatus(ctx context.Context, userAddress string) (HyperliquidWalletStatus, error)
	Account(ctx context.Context, userAddress string) (AccountBalance, error)
	Sign(ctx context.Context, action string, userID string, payload map[string]any) (ProviderActionResult, error)
	SubmitOrder(ctx context.Context, order FuturesOrder) (ProviderActionResult, error)
	CancelOrder(ctx context.Context, input CancelOrderInput) (ProviderActionResult, error)
	OrderStatus(ctx context.Context, input OrderStatusInput) (OrderStatus, error)
	UpdateLeverage(ctx context.Context, input UpdateLeverageInput) (ProviderActionResult, error)
	FundingRates(ctx context.Context, symbol string, limit int) ([]FundingRate, error)
}

type LocalProvider struct {
	now func() time.Time
}

func NewLocalProvider() *LocalProvider {
	return &LocalProvider{now: time.Now}
}

func (p *LocalProvider) Name() string {
	return "local-hyperliquid"
}

func (p *LocalProvider) WalletStatus(_ context.Context, _ string) (HyperliquidWalletStatus, error) {
	return HyperliquidWalletStatus{
		ApprovedAgent:     true,
		SetReferral:       true,
		SetFeeBuilder:     true,
		Agent:             "0xagent",
		AgentName:         "XBIT Local Agent",
		FeeBuilderAddress: "0xfeebuilder",
		FeeBuilderPercent: "0.03",
		ReferralCode:      "XBIT",
	}, nil
}

func (p *LocalProvider) Account(_ context.Context, userAddress string) (AccountBalance, error) {
	now := p.now().UTC()
	symbols := []Symbol{
		{Symbol: "BTC", MaxLeverage: 50, CurrentPrice: "95200"},
		{Symbol: "ETH", MaxLeverage: 50, CurrentPrice: "3200.5"},
	}
	return AccountBalance{
		Balance:             "250000",
		OneDayChange:        "2180",
		OneDayPercentChange: "0.87",
		RawUSD:              "250000",
		Positions: []Position{
			seedPosition(userAddress, symbols[0], "long", now),
			seedPosition(userAddress, symbols[1], "short", now),
		},
	}, nil
}

func (p *LocalProvider) Sign(_ context.Context, action string, userID string, payload map[string]any) (ProviderActionResult, error) {
	if strings.TrimSpace(userID) == "" {
		userID = "local-user"
	}
	now := p.now().UTC()
	return ProviderActionResult{
		Action:      action,
		Provider:    p.Name(),
		RequestID:   uuid.NewString(),
		Status:      "signed",
		Signature:   deterministicSignature(action, userID, payload),
		RawPayload:  payload,
		SubmittedAt: now,
	}, nil
}

func (p *LocalProvider) SubmitOrder(ctx context.Context, order FuturesOrder) (ProviderActionResult, error) {
	payload := map[string]any{
		"id":          order.ID,
		"symbol":      order.Symbol,
		"side":        order.Side,
		"orderType":   order.OrderType,
		"price":       order.Price,
		"size":        order.Size,
		"cloid":       order.Cloid,
		"reduceOnly":  order.ReduceOnly,
		"timeInForce": order.TimeInForce,
	}
	result, err := p.Sign(ctx, "submitOrder", firstNonEmpty(order.UserID, order.UserAddress), payload)
	if err != nil {
		return ProviderActionResult{}, err
	}
	result.Status = "submitted"
	return result, nil
}

func (p *LocalProvider) CancelOrder(ctx context.Context, input CancelOrderInput) (ProviderActionResult, error) {
	payload := map[string]any{
		"orderId": input.OrderID,
		"cloid":   input.Cloid,
		"symbol":  input.Symbol,
	}
	result, err := p.Sign(ctx, "cancelOrder", firstNonEmpty(input.UserID, input.UserAddress), payload)
	if err != nil {
		return ProviderActionResult{}, err
	}
	result.Status = "cancelled"
	return result, nil
}

func (p *LocalProvider) OrderStatus(_ context.Context, input OrderStatusInput) (OrderStatus, error) {
	now := p.now().UTC()
	return OrderStatus{
		OrderID:         strings.TrimSpace(input.OrderID),
		ProviderOrderID: firstNonEmpty(input.ProviderOrderID, input.OrderID),
		Cloid:           strings.TrimSpace(input.Cloid),
		Symbol:          strings.ToUpper(strings.TrimSpace(input.Symbol)),
		Status:          "submitted",
		RawPayload: map[string]any{
			"provider": p.Name(),
			"mode":     "local",
		},
		UpdatedAt: now,
	}, nil
}

func (p *LocalProvider) UpdateLeverage(ctx context.Context, input UpdateLeverageInput) (ProviderActionResult, error) {
	payload := map[string]any{
		"symbol":   input.Symbol,
		"leverage": input.Leverage,
		"isCross":  input.IsCross,
	}
	result, err := p.Sign(ctx, "updateLeverage", firstNonEmpty(input.UserID, input.UserAddress), payload)
	if err != nil {
		return ProviderActionResult{}, err
	}
	result.Status = "accepted"
	return result, nil
}

func (p *LocalProvider) FundingRates(_ context.Context, symbol string, limit int) ([]FundingRate, error) {
	if limit <= 0 || limit > 100 {
		limit = 8
	}
	now := p.now().UTC()
	symbols := []string{"BTC", "ETH", "SOL", "HYPE"}
	if strings.TrimSpace(symbol) != "" {
		symbols = []string{strings.ToUpper(strings.TrimSpace(symbol))}
	}
	out := make([]FundingRate, 0, minInt(limit, len(symbols)))
	for i, coin := range symbols {
		if len(out) >= limit {
			break
		}
		rate := "0.00008"
		if i%2 == 1 {
			rate = "-0.00003"
		}
		out = append(out, FundingRate{
			Symbol:          coin,
			FundingRate:     rate,
			Premium:         "0.00012",
			NextFundingTime: now.Add(time.Duration(i+1) * time.Hour).UnixMilli(),
			UpdatedAt:       now,
		})
	}
	return out, nil
}

func deterministicSignature(action string, userID string, payload map[string]any) Signature {
	sum := sha256.Sum256([]byte(action + ":" + fmt.Sprint(payload) + ":" + userID))
	return Signature{
		R: fmt.Sprintf("0x%x", sum[:16]),
		S: fmt.Sprintf("0x%x", sum[16:]),
		V: 27,
	}
}

var _ Provider = (*LocalProvider)(nil)
