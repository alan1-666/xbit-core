package marketdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

func (s *PostgresStore) UpsertToken(ctx context.Context, token Token) (Token, error) {
	token = normalizeToken(token, time.Now().UTC())
	rawMetadata, err := json.Marshal(token.Metadata)
	if err != nil {
		return Token{}, fmt.Errorf("marshal token metadata: %w", err)
	}
	var out Token
	var metadata []byte
	err = s.pool.QueryRow(ctx, `
		INSERT INTO market_tokens (chain_id, address, symbol, name, decimals, logo_url, price, price_24h_change, market_cap, liquidity, volume_24h, holders, dexes, category, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NULLIF($14, ''), $15, $16, $17)
		ON CONFLICT (chain_id, lower(address))
		DO UPDATE SET
			symbol = EXCLUDED.symbol,
			name = EXCLUDED.name,
			decimals = EXCLUDED.decimals,
			logo_url = EXCLUDED.logo_url,
			price = EXCLUDED.price,
			price_24h_change = EXCLUDED.price_24h_change,
			market_cap = EXCLUDED.market_cap,
			liquidity = EXCLUDED.liquidity,
			volume_24h = EXCLUDED.volume_24h,
			holders = EXCLUDED.holders,
			dexes = EXCLUDED.dexes,
			category = EXCLUDED.category,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
		RETURNING address, chain_id, symbol, name, decimals, COALESCE(logo_url, ''), price::text, price_24h_change::text, market_cap::text, liquidity::text, volume_24h::text, holders, dexes, COALESCE(category, ''), metadata, created_at, updated_at
	`, token.ChainID, token.Address, token.Symbol, token.Name, token.Decimals, token.LogoURL, token.Price, zeroIfEmpty(token.Price24hChange), token.MarketCap, token.Liquidity, token.Volume24h, token.Holders, token.Dexes, token.Category, rawMetadata, token.CreatedAt, token.UpdatedAt).Scan(
		&out.Address,
		&out.ChainID,
		&out.Symbol,
		&out.Name,
		&out.Decimals,
		&out.LogoURL,
		&out.Price,
		&out.Price24hChange,
		&out.MarketCap,
		&out.Liquidity,
		&out.Volume24h,
		&out.Holders,
		&out.Dexes,
		&out.Category,
		&metadata,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return Token{}, fmt.Errorf("upsert market token: %w", err)
	}
	_ = json.Unmarshal(metadata, &out.Metadata)
	return out, nil
}

func (s *PostgresStore) ListTokens(ctx context.Context, filter TokenFilter) ([]Token, int, error) {
	filter = normalizeFilter(filter)
	query := "%" + strings.ToLower(strings.TrimSpace(filter.Query)) + "%"
	rows, err := s.pool.Query(ctx, `
		SELECT address, chain_id, symbol, name, decimals, COALESCE(logo_url, ''), price::text, price_24h_change::text, market_cap::text, liquidity::text, volume_24h::text, holders, dexes, COALESCE(category, ''), metadata, created_at, updated_at, count(*) OVER()
		FROM market_tokens
		WHERE ($1 = 0 OR chain_id = $1)
		  AND ($2 = '' OR lower(COALESCE(category, '')) = lower($2))
		  AND ($3 = '%%' OR lower(address) LIKE $3 OR lower(symbol) LIKE $3 OR lower(name) LIKE $3)
		ORDER BY volume_24h DESC, market_cap DESC, updated_at DESC
		LIMIT $4 OFFSET $5
	`, filter.ChainID, filter.Category, query, filter.Limit, (filter.Page-1)*filter.Limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list market tokens: %w", err)
	}
	defer rows.Close()

	tokens := make([]Token, 0)
	total := 0
	for rows.Next() {
		token, rowTotal, err := scanTokenWithTotal(rows)
		if err != nil {
			return nil, 0, err
		}
		total = rowTotal
		tokens = append(tokens, token)
	}
	return tokens, total, rows.Err()
}

func (s *PostgresStore) GetToken(ctx context.Context, chainID int, address string) (Token, error) {
	var token Token
	var metadata []byte
	err := s.pool.QueryRow(ctx, `
		SELECT address, chain_id, symbol, name, decimals, COALESCE(logo_url, ''), price::text, price_24h_change::text, market_cap::text, liquidity::text, volume_24h::text, holders, dexes, COALESCE(category, ''), metadata, created_at, updated_at
		FROM market_tokens
		WHERE ($1 = 0 OR chain_id = $1) AND (lower(address) = lower($2) OR lower(symbol) = lower($2))
		ORDER BY updated_at DESC
		LIMIT 1
	`, chainID, address).Scan(
		&token.Address,
		&token.ChainID,
		&token.Symbol,
		&token.Name,
		&token.Decimals,
		&token.LogoURL,
		&token.Price,
		&token.Price24hChange,
		&token.MarketCap,
		&token.Liquidity,
		&token.Volume24h,
		&token.Holders,
		&token.Dexes,
		&token.Category,
		&metadata,
		&token.CreatedAt,
		&token.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Token{}, ErrNotFound
	}
	if err != nil {
		return Token{}, fmt.Errorf("get market token: %w", err)
	}
	_ = json.Unmarshal(metadata, &token.Metadata)
	return token, nil
}

func (s *PostgresStore) OHLC(ctx context.Context, chainID int, token string, bucket string, limit int) ([]OHLC, error) {
	if bucket == "" {
		bucket = "1m"
	}
	if limit <= 0 || limit > 500 {
		limit = 120
	}
	rows, err := s.pool.Query(ctx, `
		SELECT ts, token, open::text, high::text, low::text, close::text, close::text, token_volume::text, usd_volume::text
		FROM token_ohlc
		WHERE chain_id = $1 AND lower(token) = lower($2) AND bucket = $3
		ORDER BY ts DESC
		LIMIT $4
	`, chainID, token, bucket, limit)
	if err != nil {
		return nil, fmt.Errorf("query ohlc: %w", err)
	}
	defer rows.Close()

	points := make([]OHLC, 0)
	for rows.Next() {
		var point OHLC
		if err := rows.Scan(&point.TS, &point.Token, &point.Open, &point.High, &point.Low, &point.Close, &point.Price, &point.TokenVolume, &point.USDVolume); err != nil {
			return nil, fmt.Errorf("scan ohlc: %w", err)
		}
		points = append(points, point)
	}
	reverseOHLC(points)
	return points, rows.Err()
}

func (s *PostgresStore) Transactions(ctx context.Context, chainID int, token string, limit int) ([]Transaction, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, timestamp, chain_id, tx_hash, log_index, event_index, base_token, quote_token, pair, side, maker, base_amount::text, quote_amount::text, price::text, usd_amount::text, usd_price::text, liquidity::text, dex, metadata
		FROM token_transactions
		WHERE chain_id = $1 AND lower(base_token) = lower($2)
		ORDER BY timestamp DESC, log_index DESC
		LIMIT $3
	`, chainID, token, limit)
	if err != nil {
		return nil, fmt.Errorf("query token transactions: %w", err)
	}
	defer rows.Close()

	txs := make([]Transaction, 0)
	for rows.Next() {
		var tx Transaction
		var metadata []byte
		if err := rows.Scan(&tx.ID, &tx.Timestamp, &tx.ChainID, &tx.TxHash, &tx.LogIndex, &tx.EventIndex, &tx.BaseToken, &tx.QuoteToken, &tx.Pair, &tx.Type, &tx.Maker, &tx.BaseAmount, &tx.QuoteAmount, &tx.Price, &tx.USDAmount, &tx.USDPrice, &tx.Liquidity, &tx.Dex, &metadata); err != nil {
			return nil, fmt.Errorf("scan token transaction: %w", err)
		}
		_ = json.Unmarshal(metadata, &tx.Metadata)
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

func (s *PostgresStore) Pools(ctx context.Context, chainID int, token string) ([]Pool, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT address, chain_id, base_token, quote_token, quote_token_price::text, quote_symbol, base_symbol, base_token_liquidity::text, quote_liquidity::text, usd_liquidity::text, dex, created_at
		FROM token_pools
		WHERE chain_id = $1 AND lower(base_token) = lower($2)
		ORDER BY usd_liquidity DESC
	`, chainID, token)
	if err != nil {
		return nil, fmt.Errorf("query token pools: %w", err)
	}
	defer rows.Close()

	pools := make([]Pool, 0)
	for rows.Next() {
		var pool Pool
		if err := rows.Scan(&pool.Address, &pool.ChainID, &pool.BaseToken, &pool.QuoteToken, &pool.QuoteTokenPrice, &pool.QuoteSymbol, &pool.BaseSymbol, &pool.BaseTokenLiquidity, &pool.QuoteLiquidity, &pool.USDLiquidity, &pool.Dex, &pool.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan token pool: %w", err)
		}
		pools = append(pools, pool)
	}
	return pools, rows.Err()
}

func (s *PostgresStore) AppendTransaction(ctx context.Context, tx Transaction) (Transaction, error) {
	rawMetadata, err := json.Marshal(tx.Metadata)
	if err != nil {
		return Transaction{}, fmt.Errorf("marshal transaction metadata: %w", err)
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO token_transactions (chain_id, token, tx_hash, log_index, event_index, base_token, quote_token, pair, side, maker, base_amount, quote_amount, price, usd_amount, usd_price, liquidity, dex, timestamp, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id::text
	`, tx.ChainID, tx.BaseToken, tx.TxHash, tx.LogIndex, tx.EventIndex, tx.BaseToken, tx.QuoteToken, tx.Pair, tx.Type, tx.Maker, zeroIfEmpty(tx.BaseAmount), zeroIfEmpty(tx.QuoteAmount), zeroIfEmpty(tx.Price), zeroIfEmpty(tx.USDAmount), zeroIfEmpty(tx.USDPrice), zeroIfEmpty(tx.Liquidity), tx.Dex, tx.Timestamp, rawMetadata).Scan(&tx.ID)
	if err != nil {
		return Transaction{}, fmt.Errorf("append token transaction: %w", err)
	}
	return tx, nil
}

func (s *PostgresStore) SaveCheckpoint(ctx context.Context, checkpoint Checkpoint) (Checkpoint, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO indexer_checkpoints (source, cursor, block_number, event_ts, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (source)
		DO UPDATE SET cursor = EXCLUDED.cursor, block_number = EXCLUDED.block_number, event_ts = EXCLUDED.event_ts, updated_at = EXCLUDED.updated_at
		RETURNING source, cursor, block_number, event_ts, updated_at
	`, checkpoint.Source, checkpoint.Cursor, checkpoint.BlockNumber, checkpoint.EventTS).Scan(&checkpoint.Source, &checkpoint.Cursor, &checkpoint.BlockNumber, &checkpoint.EventTS, &checkpoint.UpdatedAt)
	if err != nil {
		return Checkpoint{}, fmt.Errorf("save indexer checkpoint: %w", err)
	}
	return checkpoint, nil
}

func (s *PostgresStore) GetCheckpoint(ctx context.Context, source string) (Checkpoint, error) {
	var checkpoint Checkpoint
	err := s.pool.QueryRow(ctx, `
		SELECT source, cursor, block_number, event_ts, updated_at
		FROM indexer_checkpoints
		WHERE source = $1
	`, source).Scan(&checkpoint.Source, &checkpoint.Cursor, &checkpoint.BlockNumber, &checkpoint.EventTS, &checkpoint.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checkpoint{}, ErrNotFound
	}
	if err != nil {
		return Checkpoint{}, fmt.Errorf("get indexer checkpoint: %w", err)
	}
	return checkpoint, nil
}

type tokenScanner interface {
	Scan(dest ...any) error
}

func scanTokenWithTotal(row tokenScanner) (Token, int, error) {
	var token Token
	var metadata []byte
	total := 0
	err := row.Scan(
		&token.Address,
		&token.ChainID,
		&token.Symbol,
		&token.Name,
		&token.Decimals,
		&token.LogoURL,
		&token.Price,
		&token.Price24hChange,
		&token.MarketCap,
		&token.Liquidity,
		&token.Volume24h,
		&token.Holders,
		&token.Dexes,
		&token.Category,
		&metadata,
		&token.CreatedAt,
		&token.UpdatedAt,
		&total,
	)
	if err != nil {
		return Token{}, 0, fmt.Errorf("scan market token: %w", err)
	}
	_ = json.Unmarshal(metadata, &token.Metadata)
	return token, total, nil
}

func reverseOHLC(points []OHLC) {
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}
}

func zeroIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "0"
	}
	return value
}
