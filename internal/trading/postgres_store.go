package trading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) SaveQuote(ctx context.Context, quote Quote) (Quote, error) {
	rawSnapshot, err := json.Marshal(quote.RouteSnapshot)
	if err != nil {
		return Quote{}, fmt.Errorf("marshal route snapshot: %w", err)
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO quote_snapshots (user_id, chain_type, input_token, output_token, input_amount, output_amount, min_output_amount, slippage_bps, platform_fee_amount, route_snapshot, expires_at, created_at)
		VALUES (NULLIF($1, ''), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, quote.UserID, quote.ChainType, quote.InputToken, quote.OutputToken, quote.InputAmount, quote.OutputAmount, quote.MinOutputAmount, quote.SlippageBps, quote.PlatformFeeAmount, rawSnapshot, quote.ExpiresAt, quote.CreatedAt).Scan(&quote.ID)
	if err != nil {
		return Quote{}, fmt.Errorf("save quote: %w", err)
	}
	return quote, nil
}

func (s *PostgresStore) CreateOrder(ctx context.Context, order Order) (Order, error) {
	rawSnapshot, err := json.Marshal(order.RouteSnapshot)
	if err != nil {
		return Order{}, fmt.Errorf("marshal route snapshot: %w", err)
	}
	var out Order
	var rawOut []byte
	err = s.pool.QueryRow(ctx, `
		INSERT INTO orders (user_id, chain_type, wallet_address, order_type, side, input_token, output_token, input_amount, expected_output_amount, min_output_amount, slippage_bps, route_snapshot, status, client_request_id, created_at, updated_at, expired_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NULLIF($14, ''), $15, $16, $17)
		ON CONFLICT (user_id, client_request_id)
		WHERE client_request_id IS NOT NULL AND client_request_id <> ''
		DO UPDATE SET updated_at = orders.updated_at
		RETURNING id, user_id, chain_type, wallet_address, order_type, side, input_token, output_token, input_amount::text, expected_output_amount::text, min_output_amount::text, slippage_bps, route_snapshot, status, COALESCE(tx_hash, ''), COALESCE(failure_code, ''), COALESCE(client_request_id, ''), created_at, updated_at, filled_at, expired_at
	`, order.UserID, order.ChainType, order.WalletAddress, order.OrderType, order.Side, order.InputToken, order.OutputToken, order.InputAmount, order.ExpectedOutputAmount, order.MinOutputAmount, order.SlippageBps, rawSnapshot, order.Status, order.ClientRequestID, order.CreatedAt, order.UpdatedAt, order.ExpiredAt).Scan(
		&out.ID,
		&out.UserID,
		&out.ChainType,
		&out.WalletAddress,
		&out.OrderType,
		&out.Side,
		&out.InputToken,
		&out.OutputToken,
		&out.InputAmount,
		&out.ExpectedOutputAmount,
		&out.MinOutputAmount,
		&out.SlippageBps,
		&rawOut,
		&out.Status,
		&out.TxHash,
		&out.FailureCode,
		&out.ClientRequestID,
		&out.CreatedAt,
		&out.UpdatedAt,
		&out.FilledAt,
		&out.ExpiredAt,
	)
	if err != nil {
		return Order{}, fmt.Errorf("create order: %w", err)
	}
	_ = json.Unmarshal(rawOut, &out.RouteSnapshot)
	return out, nil
}

func (s *PostgresStore) GetOrder(ctx context.Context, orderID string) (Order, error) {
	return s.scanOrder(ctx, `
		SELECT id, user_id, chain_type, wallet_address, order_type, side, input_token, output_token, input_amount::text, expected_output_amount::text, min_output_amount::text, slippage_bps, route_snapshot, status, COALESCE(tx_hash, ''), COALESCE(failure_code, ''), COALESCE(client_request_id, ''), created_at, updated_at, filled_at, expired_at
		FROM orders
		WHERE id = $1
	`, orderID)
}

func (s *PostgresStore) FindOrderByClientRequestID(ctx context.Context, userID string, clientRequestID string) (Order, error) {
	return s.scanOrder(ctx, `
		SELECT id, user_id, chain_type, wallet_address, order_type, side, input_token, output_token, input_amount::text, expected_output_amount::text, min_output_amount::text, slippage_bps, route_snapshot, status, COALESCE(tx_hash, ''), COALESCE(failure_code, ''), COALESCE(client_request_id, ''), created_at, updated_at, filled_at, expired_at
		FROM orders
		WHERE user_id = $1 AND client_request_id = $2
	`, userID, clientRequestID)
}

func (s *PostgresStore) ListOrders(ctx context.Context, input SearchOrdersInput) ([]Order, error) {
	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, chain_type, wallet_address, order_type, side, input_token, output_token, input_amount::text, expected_output_amount::text, min_output_amount::text, slippage_bps, route_snapshot, status, COALESCE(tx_hash, ''), COALESCE(failure_code, ''), COALESCE(client_request_id, ''), created_at, updated_at, filled_at, expired_at
		FROM orders
		WHERE user_id = $1 AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3
	`, input.UserID, input.Status, limit)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]Order, 0)
	for rows.Next() {
		order, err := scanOrderRows(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (s *PostgresStore) UpdateOrderStatus(ctx context.Context, orderID string, update UpdateOrderStatusInput, now time.Time) (Order, error) {
	return s.scanOrder(ctx, `
		UPDATE orders
		SET status = $2,
		    tx_hash = NULLIF($3, ''),
		    failure_code = NULLIF($4, ''),
		    updated_at = $5,
		    filled_at = CASE WHEN $2 = 'confirmed' THEN $5 ELSE filled_at END
		WHERE id = $1
		RETURNING id, user_id, chain_type, wallet_address, order_type, side, input_token, output_token, input_amount::text, expected_output_amount::text, min_output_amount::text, slippage_bps, route_snapshot, status, COALESCE(tx_hash, ''), COALESCE(failure_code, ''), COALESCE(client_request_id, ''), created_at, updated_at, filled_at, expired_at
	`, orderID, update.Status, update.TxHash, update.FailureCode, now)
}

func (s *PostgresStore) AppendOrderEvent(ctx context.Context, orderID string, eventType string, payload map[string]any, now time.Time) error {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal order event: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO order_events (order_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4)
	`, orderID, eventType, rawPayload, now)
	if err != nil {
		return fmt.Errorf("append order event: %w", err)
	}
	return nil
}

func (s *PostgresStore) SaveNetworkFee(ctx context.Context, fee NetworkFee) error {
	payload, err := json.Marshal(fee)
	if err != nil {
		return fmt.Errorf("marshal network fee: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO network_fee_snapshots (chain_type, payload, source, created_at)
		VALUES ($1, $2, $3, $4)
	`, fee.ChainType, payload, fee.Source, fee.CreatedAt)
	if err != nil {
		return fmt.Errorf("save network fee: %w", err)
	}
	return nil
}

func (s *PostgresStore) LatestNetworkFee(ctx context.Context, chainType string) (NetworkFee, error) {
	var payload []byte
	err := s.pool.QueryRow(ctx, `
		SELECT payload
		FROM network_fee_snapshots
		WHERE chain_type = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, chainType).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return NetworkFee{}, ErrNotFound
	}
	if err != nil {
		return NetworkFee{}, fmt.Errorf("latest network fee: %w", err)
	}
	var fee NetworkFee
	if err := json.Unmarshal(payload, &fee); err != nil {
		return NetworkFee{}, fmt.Errorf("unmarshal network fee: %w", err)
	}
	return fee, nil
}

func (s *PostgresStore) scanOrder(ctx context.Context, sql string, args ...any) (Order, error) {
	row := s.pool.QueryRow(ctx, sql, args...)
	order, err := scanOrderRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}
	return order, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanOrderRow(row scanner) (Order, error) {
	var order Order
	var rawSnapshot []byte
	err := row.Scan(
		&order.ID,
		&order.UserID,
		&order.ChainType,
		&order.WalletAddress,
		&order.OrderType,
		&order.Side,
		&order.InputToken,
		&order.OutputToken,
		&order.InputAmount,
		&order.ExpectedOutputAmount,
		&order.MinOutputAmount,
		&order.SlippageBps,
		&rawSnapshot,
		&order.Status,
		&order.TxHash,
		&order.FailureCode,
		&order.ClientRequestID,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.FilledAt,
		&order.ExpiredAt,
	)
	if err != nil {
		return Order{}, err
	}
	_ = json.Unmarshal(rawSnapshot, &order.RouteSnapshot)
	return order, nil
}

func scanOrderRows(rows pgx.Rows) (Order, error) {
	order, err := scanOrderRow(rows)
	if err != nil {
		return Order{}, fmt.Errorf("scan order: %w", err)
	}
	return order, nil
}
