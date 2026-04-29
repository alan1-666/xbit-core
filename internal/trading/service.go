package trading

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/xbit/xbit-backend/pkg/money"
)

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{store: store, now: time.Now}
}

func (s *Service) Quote(ctx context.Context, req QuoteRequest) (Quote, error) {
	req.UserID = strings.TrimSpace(req.UserID)
	req.ChainType = strings.TrimSpace(req.ChainType)
	req.InputToken = strings.TrimSpace(req.InputToken)
	req.OutputToken = strings.TrimSpace(req.OutputToken)
	req.InputAmount = strings.TrimSpace(req.InputAmount)
	if req.ChainType == "" || req.InputToken == "" || req.OutputToken == "" || req.InputAmount == "" {
		return Quote{}, fmt.Errorf("chain type, input token, output token and input amount are required")
	}
	if req.SlippageBps <= 0 {
		req.SlippageBps = 100
	}
	if req.SlippageBps > 5000 {
		return Quote{}, fmt.Errorf("slippage too high")
	}
	if _, err := money.Parse(req.InputAmount); err != nil {
		return Quote{}, err
	}

	platformFee, err := money.MultiplyBps(req.InputAmount, 30)
	if err != nil {
		return Quote{}, err
	}
	outputAmount, err := money.Sub(req.InputAmount, platformFee)
	if err != nil {
		return Quote{}, err
	}
	slippageAmount, err := money.MultiplyBps(outputAmount, int64(req.SlippageBps))
	if err != nil {
		return Quote{}, err
	}
	minOutput, err := money.Sub(outputAmount, slippageAmount)
	if err != nil {
		return Quote{}, err
	}

	now := s.now().UTC()
	quote := Quote{
		RouteID:           "internal-mvp-" + uuid.NewString(),
		UserID:            req.UserID,
		ChainType:         req.ChainType,
		InputToken:        req.InputToken,
		OutputToken:       req.OutputToken,
		InputAmount:       req.InputAmount,
		OutputAmount:      outputAmount,
		MinOutputAmount:   minOutput,
		SlippageBps:       req.SlippageBps,
		PlatformFeeAmount: platformFee,
		RouteSnapshot: map[string]any{
			"provider":       "internal-mvp",
			"platformFeeBps": 30,
			"slippageBps":    req.SlippageBps,
		},
		ExpiresAt: now.Add(30 * time.Second),
		CreatedAt: now,
	}
	return s.store.SaveQuote(ctx, quote)
}

func (s *Service) CreateOrder(ctx context.Context, input CreateOrderInput) (Order, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.ChainType = strings.TrimSpace(input.ChainType)
	input.WalletAddress = strings.TrimSpace(input.WalletAddress)
	input.OrderType = strings.ToLower(strings.TrimSpace(input.OrderType))
	input.Side = strings.ToLower(strings.TrimSpace(input.Side))
	input.InputToken = strings.TrimSpace(input.InputToken)
	input.OutputToken = strings.TrimSpace(input.OutputToken)
	input.InputAmount = strings.TrimSpace(input.InputAmount)
	input.ClientRequestID = strings.TrimSpace(input.ClientRequestID)

	if input.UserID == "" || input.ChainType == "" || input.WalletAddress == "" {
		return Order{}, fmt.Errorf("user id, chain type and wallet address are required")
	}
	if input.OrderType == "" {
		input.OrderType = OrderTypeMarket
	}
	if input.OrderType != OrderTypeMarket && input.OrderType != OrderTypeLimit {
		return Order{}, fmt.Errorf("unsupported order type")
	}
	if input.Side != SideBuy && input.Side != SideSell {
		return Order{}, fmt.Errorf("side must be buy or sell")
	}
	if input.ClientRequestID != "" {
		if existing, err := s.store.FindOrderByClientRequestID(ctx, input.UserID, input.ClientRequestID); err == nil {
			return existing, nil
		}
	}

	quote, err := s.Quote(ctx, QuoteRequest{
		UserID:      input.UserID,
		ChainType:   input.ChainType,
		InputToken:  input.InputToken,
		OutputToken: input.OutputToken,
		InputAmount: input.InputAmount,
		SlippageBps: input.SlippageBps,
	})
	if err != nil {
		return Order{}, err
	}
	if input.RouteSnapshot == nil {
		input.RouteSnapshot = quote.RouteSnapshot
	}
	input.RouteSnapshot["quoteId"] = quote.ID
	input.RouteSnapshot["routeId"] = quote.RouteID

	now := s.now().UTC()
	order := Order{
		ID:                   uuid.NewString(),
		UserID:               input.UserID,
		ChainType:            input.ChainType,
		WalletAddress:        input.WalletAddress,
		OrderType:            input.OrderType,
		Side:                 input.Side,
		InputToken:           input.InputToken,
		OutputToken:          input.OutputToken,
		InputAmount:          input.InputAmount,
		ExpectedOutputAmount: quote.OutputAmount,
		MinOutputAmount:      quote.MinOutputAmount,
		SlippageBps:          quote.SlippageBps,
		RouteSnapshot:        input.RouteSnapshot,
		Status:               OrderStatusPending,
		ClientRequestID:      input.ClientRequestID,
		CreatedAt:            now,
		UpdatedAt:            now,
		ExpiredAt:            &quote.ExpiresAt,
	}
	created, err := s.store.CreateOrder(ctx, order)
	if err != nil {
		return Order{}, err
	}
	_ = s.store.AppendOrderEvent(ctx, created.ID, "created", map[string]any{"status": created.Status}, now)
	return created, nil
}

func (s *Service) GetOrder(ctx context.Context, orderID string) (Order, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return Order{}, fmt.Errorf("order id is required")
	}
	return s.store.GetOrder(ctx, orderID)
}

func (s *Service) ListOrders(ctx context.Context, input SearchOrdersInput) ([]Order, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	if input.UserID == "" {
		return nil, fmt.Errorf("user id is required")
	}
	if input.Limit <= 0 || input.Limit > 100 {
		input.Limit = 50
	}
	return s.store.ListOrders(ctx, input)
}

func (s *Service) UpdateOrderStatus(ctx context.Context, orderID string, update UpdateOrderStatusInput) (Order, error) {
	orderID = strings.TrimSpace(orderID)
	update.Status = strings.ToLower(strings.TrimSpace(update.Status))
	if orderID == "" || update.Status == "" {
		return Order{}, fmt.Errorf("order id and status are required")
	}
	if !validStatus(update.Status) {
		return Order{}, fmt.Errorf("unsupported order status")
	}
	now := s.now().UTC()
	order, err := s.store.UpdateOrderStatus(ctx, orderID, update, now)
	if err != nil {
		return Order{}, err
	}
	payload := update.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	payload["status"] = order.Status
	payload["txHash"] = order.TxHash
	payload["failureCode"] = order.FailureCode
	_ = s.store.AppendOrderEvent(ctx, order.ID, "status_updated", payload, now)
	return order, nil
}

func (s *Service) CancelOrder(ctx context.Context, orderID string) (Order, error) {
	return s.UpdateOrderStatus(ctx, orderID, UpdateOrderStatusInput{Status: OrderStatusCancelled})
}

func (s *Service) GetNetworkFee(ctx context.Context, chainType string) (NetworkFee, error) {
	chainType = strings.TrimSpace(chainType)
	if chainType == "" {
		return NetworkFee{}, fmt.Errorf("chain type is required")
	}
	if fee, err := s.store.LatestNetworkFee(ctx, chainType); err == nil {
		return fee, nil
	}
	fee := defaultNetworkFee(chainType, s.now().UTC())
	_ = s.store.SaveNetworkFee(ctx, fee)
	return fee, nil
}

func validStatus(status string) bool {
	switch status {
	case OrderStatusPending, OrderStatusSubmitted, OrderStatusConfirmed, OrderStatusFailed, OrderStatusCancelled:
		return true
	default:
		return false
	}
}

func defaultNetworkFee(chainType string, now time.Time) NetworkFee {
	fee := NetworkFee{
		ChainType:      chainType,
		PlatformFeeBps: 30,
		Source:         "trading-svc-default",
		CreatedAt:      now,
	}
	switch strings.ToLower(chainType) {
	case "solana", "sol":
		fee.MaxComputeUnits = 1_400_000
		fee.PriorityFeePrice = map[string]any{"medium": 0.000005, "high": 0.00001, "veryHigh": 0.00002}
		fee.MinTipFee = "0.000001"
	case "bsc", "mon", "evm", "eth", "arb":
		fee.PriorityFeePrice = map[string]any{"low": "1", "medium": "2", "high": "3"}
		fee.MinTipFee = "0"
	default:
		fee.PriorityFeePrice = map[string]any{"medium": "1"}
	}
	return fee
}
