package trading

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	SaveQuote(ctx context.Context, quote Quote) (Quote, error)
	CreateOrder(ctx context.Context, order Order) (Order, error)
	GetOrder(ctx context.Context, orderID string) (Order, error)
	FindOrderByClientRequestID(ctx context.Context, userID string, clientRequestID string) (Order, error)
	ListOrders(ctx context.Context, input SearchOrdersInput) ([]Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, update UpdateOrderStatusInput, now time.Time) (Order, error)
	AppendOrderEvent(ctx context.Context, orderID string, eventType string, payload map[string]any, now time.Time) error
	SaveNetworkFee(ctx context.Context, fee NetworkFee) error
	LatestNetworkFee(ctx context.Context, chainType string) (NetworkFee, error)
}

type MemoryStore struct {
	mu          sync.RWMutex
	quotes      map[string]Quote
	orders      map[string]Order
	networkFees map[string]NetworkFee
	events      []map[string]any
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		quotes:      map[string]Quote{},
		orders:      map[string]Order{},
		networkFees: map[string]NetworkFee{},
		events:      make([]map[string]any, 0),
	}
}

func (s *MemoryStore) SaveQuote(_ context.Context, quote Quote) (Quote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if quote.ID == "" {
		quote.ID = uuid.NewString()
	}
	s.quotes[quote.ID] = quote
	return quote, nil
}

func (s *MemoryStore) CreateOrder(_ context.Context, order Order) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if order.ClientRequestID != "" {
		for _, existing := range s.orders {
			if existing.UserID == order.UserID && existing.ClientRequestID == order.ClientRequestID {
				return existing, nil
			}
		}
	}
	if order.ID == "" {
		order.ID = uuid.NewString()
	}
	s.orders[order.ID] = order
	return order, nil
}

func (s *MemoryStore) GetOrder(_ context.Context, orderID string) (Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[orderID]
	if !ok {
		return Order{}, ErrNotFound
	}
	return order, nil
}

func (s *MemoryStore) FindOrderByClientRequestID(_ context.Context, userID string, clientRequestID string) (Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, order := range s.orders {
		if order.UserID == userID && order.ClientRequestID == clientRequestID {
			return order, nil
		}
	}
	return Order{}, ErrNotFound
}

func (s *MemoryStore) ListOrders(_ context.Context, input SearchOrdersInput) ([]Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orders := make([]Order, 0)
	for _, order := range s.orders {
		if order.UserID != input.UserID {
			continue
		}
		if input.Status != "" && order.Status != input.Status {
			continue
		}
		orders = append(orders, order)
	}
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].CreatedAt.After(orders[j].CreatedAt)
	})
	if input.Limit > 0 && len(orders) > input.Limit {
		orders = orders[:input.Limit]
	}
	return orders, nil
}

func (s *MemoryStore) UpdateOrderStatus(_ context.Context, orderID string, update UpdateOrderStatusInput, now time.Time) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[orderID]
	if !ok {
		return Order{}, ErrNotFound
	}
	order.Status = update.Status
	order.TxHash = strings.TrimSpace(update.TxHash)
	order.FailureCode = strings.TrimSpace(update.FailureCode)
	order.UpdatedAt = now
	if update.Status == OrderStatusConfirmed {
		order.FilledAt = &now
	}
	s.orders[orderID] = order
	return order, nil
}

func (s *MemoryStore) AppendOrderEvent(_ context.Context, orderID string, eventType string, payload map[string]any, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, map[string]any{
		"orderId":   orderID,
		"eventType": eventType,
		"payload":   payload,
		"createdAt": now,
	})
	return nil
}

func (s *MemoryStore) SaveNetworkFee(_ context.Context, fee NetworkFee) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.networkFees[strings.ToLower(fee.ChainType)] = fee
	return nil
}

func (s *MemoryStore) LatestNetworkFee(_ context.Context, chainType string) (NetworkFee, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fee, ok := s.networkFees[strings.ToLower(chainType)]
	if !ok {
		return NetworkFee{}, ErrNotFound
	}
	return fee, nil
}
