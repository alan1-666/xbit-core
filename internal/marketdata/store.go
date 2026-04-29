package marketdata

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
	UpsertToken(ctx context.Context, token Token) (Token, error)
	ListTokens(ctx context.Context, filter TokenFilter) ([]Token, int, error)
	GetToken(ctx context.Context, chainID int, address string) (Token, error)
	OHLC(ctx context.Context, chainID int, token string, bucket string, limit int) ([]OHLC, error)
	Transactions(ctx context.Context, chainID int, token string, limit int) ([]Transaction, error)
	Pools(ctx context.Context, chainID int, token string) ([]Pool, error)
	AppendTransaction(ctx context.Context, tx Transaction) (Transaction, error)
	SaveCheckpoint(ctx context.Context, checkpoint Checkpoint) (Checkpoint, error)
	GetCheckpoint(ctx context.Context, source string) (Checkpoint, error)
}

type MemoryStore struct {
	mu           sync.RWMutex
	tokens       map[string]Token
	ohlc         map[string][]OHLC
	transactions map[string][]Transaction
	pools        map[string][]Pool
	checkpoints  map[string]Checkpoint
}

func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		tokens:       map[string]Token{},
		ohlc:         map[string][]OHLC{},
		transactions: map[string][]Transaction{},
		pools:        map[string][]Pool{},
		checkpoints:  map[string]Checkpoint{},
	}
	store.seed()
	return store
}

func (s *MemoryStore) seed() {
	now := time.Now().UTC().Truncate(time.Second)
	tokens := []Token{
		{
			Address:        "So11111111111111111111111111111111111111112",
			ChainID:        501,
			Symbol:         "SOL",
			Name:           "Solana",
			Decimals:       9,
			LogoURL:        "",
			Price:          "145.23",
			Price24hChange: "2.41",
			MarketCap:      "68000000000",
			Liquidity:      "210000000",
			Volume24h:      "3200000000",
			Holders:        1200000,
			Dexes:          []string{"Jupiter", "Raydium"},
			Category:       "layer1",
			Metadata:       map[string]any{"source": "seed", "website": "https://solana.com"},
			CreatedAt:      now.Add(-6 * 365 * 24 * time.Hour),
			UpdatedAt:      now,
		},
		{
			Address:        "xbit-demo-token",
			ChainID:        501,
			Symbol:         "XBIT",
			Name:           "XBIT Demo Token",
			Decimals:       6,
			LogoURL:        "",
			Price:          "0.042",
			Price24hChange: "18.7",
			MarketCap:      "4200000",
			Liquidity:      "760000",
			Volume24h:      "1250000",
			Holders:        18420,
			Dexes:          []string{"Raydium", "Meteora"},
			Category:       "meme",
			Metadata:       map[string]any{"source": "seed", "twitterUrl": "https://x.com/xbit"},
			CreatedAt:      now.Add(-72 * time.Hour),
			UpdatedAt:      now,
		},
		{
			Address:        "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			ChainID:        1,
			Symbol:         "ETH",
			Name:           "Ethereum",
			Decimals:       18,
			LogoURL:        "",
			Price:          "3200.50",
			Price24hChange: "1.15",
			MarketCap:      "385000000000",
			Liquidity:      "500000000",
			Volume24h:      "12800000000",
			Holders:        9000000,
			Dexes:          []string{"Uniswap"},
			Category:       "layer1",
			Metadata:       map[string]any{"source": "seed", "website": "https://ethereum.org"},
			CreatedAt:      now.Add(-9 * 365 * 24 * time.Hour),
			UpdatedAt:      now,
		},
		{
			Address:        "0x6982508145454ce325ddbe47a25d4ec3d2311933",
			ChainID:        1,
			Symbol:         "PEPE",
			Name:           "Pepe",
			Decimals:       18,
			LogoURL:        "",
			Price:          "0.0000082",
			Price24hChange: "-3.21",
			MarketCap:      "3450000000",
			Liquidity:      "86000000",
			Volume24h:      "540000000",
			Holders:        330000,
			Dexes:          []string{"Uniswap"},
			Category:       "meme",
			Metadata:       map[string]any{"source": "seed"},
			CreatedAt:      now.Add(-2 * 365 * 24 * time.Hour),
			UpdatedAt:      now,
		},
	}
	for _, token := range tokens {
		key := tokenKey(token.ChainID, token.Address)
		s.tokens[key] = token
		s.ohlc[key+":1m"] = generatedOHLC(token, 60, now)
		s.transactions[key] = generatedTransactions(token, now)
		s.pools[key] = []Pool{{
			Address:            "pool-" + strings.ToLower(token.Symbol),
			ChainID:            token.ChainID,
			BaseToken:          token.Address,
			QuoteToken:         "USDC",
			QuoteTokenPrice:    "1",
			QuoteSymbol:        "USDC",
			BaseSymbol:         token.Symbol,
			BaseTokenLiquidity: token.Liquidity,
			QuoteLiquidity:     token.Liquidity,
			USDLiquidity:       token.Liquidity,
			Dex:                firstDex(token.Dexes),
			CreatedAt:          now.Add(-24 * time.Hour),
		}}
	}
}

func (s *MemoryStore) UpsertToken(_ context.Context, token Token) (Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token = normalizeToken(token, time.Now().UTC())
	s.tokens[tokenKey(token.ChainID, token.Address)] = token
	return token, nil
}

func (s *MemoryStore) ListTokens(_ context.Context, filter TokenFilter) ([]Token, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	filter = normalizeFilter(filter)
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	category := strings.ToLower(strings.TrimSpace(filter.Category))
	out := make([]Token, 0, len(s.tokens))
	for _, token := range s.tokens {
		if filter.ChainID != 0 && token.ChainID != filter.ChainID {
			continue
		}
		if category != "" && strings.ToLower(token.Category) != category {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(token.Address+" "+token.Symbol+" "+token.Name), query) {
			continue
		}
		out = append(out, token)
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := strconv.ParseFloat(out[i].Volume24h, 64)
		right, _ := strconv.ParseFloat(out[j].Volume24h, 64)
		return left > right
	})
	total := len(out)
	start := (filter.Page - 1) * filter.Limit
	if start >= len(out) {
		return []Token{}, total, nil
	}
	end := start + filter.Limit
	if end > len(out) {
		end = len(out)
	}
	return out[start:end], total, nil
}

func (s *MemoryStore) GetToken(_ context.Context, chainID int, address string) (Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if chainID != 0 {
		token, ok := s.tokens[tokenKey(chainID, address)]
		if ok {
			return token, nil
		}
	}
	address = strings.ToLower(strings.TrimSpace(address))
	for _, token := range s.tokens {
		if strings.EqualFold(token.Address, address) || strings.EqualFold(token.Symbol, address) {
			return token, nil
		}
	}
	return Token{}, ErrNotFound
}

func (s *MemoryStore) OHLC(_ context.Context, chainID int, token string, bucket string, limit int) ([]OHLC, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 500 {
		limit = 120
	}
	if bucket == "" {
		bucket = "1m"
	}
	key := tokenKey(chainID, token) + ":" + bucket
	points := append([]OHLC(nil), s.ohlc[key]...)
	if len(points) == 0 {
		if found, err := s.GetToken(context.Background(), chainID, token); err == nil {
			points = generatedOHLC(found, limit, time.Now().UTC())
		}
	}
	if len(points) > limit {
		points = points[len(points)-limit:]
	}
	return points, nil
}

func (s *MemoryStore) Transactions(_ context.Context, chainID int, token string, limit int) ([]Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	txs := append([]Transaction(nil), s.transactions[tokenKey(chainID, token)]...)
	if len(txs) == 0 {
		if found, err := s.GetToken(context.Background(), chainID, token); err == nil {
			txs = generatedTransactions(found, time.Now().UTC())
		}
	}
	if len(txs) > limit {
		txs = txs[:limit]
	}
	return txs, nil
}

func (s *MemoryStore) Pools(_ context.Context, chainID int, token string) ([]Pool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pools := append([]Pool(nil), s.pools[tokenKey(chainID, token)]...)
	return pools, nil
}

func (s *MemoryStore) AppendTransaction(_ context.Context, tx Transaction) (Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tx.ID == "" {
		tx.ID = uuid.NewString()
	}
	if tx.Timestamp == 0 {
		tx.Timestamp = time.Now().Unix()
	}
	key := tokenKey(tx.ChainID, tx.BaseToken)
	s.transactions[key] = append([]Transaction{tx}, s.transactions[key]...)
	return tx, nil
}

func (s *MemoryStore) SaveCheckpoint(_ context.Context, checkpoint Checkpoint) (Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	checkpoint.Source = strings.TrimSpace(checkpoint.Source)
	if checkpoint.Source == "" {
		return Checkpoint{}, errors.New("source is required")
	}
	checkpoint.UpdatedAt = time.Now().UTC()
	s.checkpoints[checkpoint.Source] = checkpoint
	return checkpoint, nil
}

func (s *MemoryStore) GetCheckpoint(_ context.Context, source string) (Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	checkpoint, ok := s.checkpoints[strings.TrimSpace(source)]
	if !ok {
		return Checkpoint{}, ErrNotFound
	}
	return checkpoint, nil
}

func normalizeFilter(filter TokenFilter) TokenFilter {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return filter
}

func normalizeToken(token Token, now time.Time) Token {
	token.Address = strings.TrimSpace(token.Address)
	token.Symbol = strings.TrimSpace(token.Symbol)
	token.Name = strings.TrimSpace(token.Name)
	if token.ChainID == 0 {
		token.ChainID = 501
	}
	if token.Decimals == 0 {
		token.Decimals = 18
	}
	if token.Price == "" {
		token.Price = "0"
	}
	if token.MarketCap == "" {
		token.MarketCap = "0"
	}
	if token.Liquidity == "" {
		token.Liquidity = "0"
	}
	if token.Volume24h == "" {
		token.Volume24h = "0"
	}
	if token.Metadata == nil {
		token.Metadata = map[string]any{}
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = now
	}
	token.UpdatedAt = now
	return token
}

func tokenKey(chainID int, address string) string {
	return strconv.Itoa(chainID) + ":" + strings.ToLower(strings.TrimSpace(address))
}

func firstDex(dexes []string) string {
	if len(dexes) == 0 {
		return "Internal"
	}
	return dexes[0]
}

func generatedOHLC(token Token, limit int, now time.Time) []OHLC {
	base, _ := strconv.ParseFloat(token.Price, 64)
	if base <= 0 {
		base = 1
	}
	out := make([]OHLC, 0, limit)
	for i := limit - 1; i >= 0; i-- {
		ts := now.Add(-time.Duration(i) * time.Minute).Unix()
		delta := float64((i%9)-4) / 1000
		open := base * (1 + delta)
		close := base * (1 + delta/2)
		high := maxFloat(open, close) * 1.002
		low := minFloat(open, close) * 0.998
		out = append(out, OHLC{
			TS:          ts,
			Token:       token.Address,
			Open:        formatFloat(open),
			High:        formatFloat(high),
			Low:         formatFloat(low),
			Close:       formatFloat(close),
			Price:       formatFloat(close),
			TokenVolume: formatFloat(1000 + float64(i*7)),
			USDVolume:   formatFloat(10000 + float64(i*300)),
		})
	}
	return out
}

func generatedTransactions(token Token, now time.Time) []Transaction {
	out := make([]Transaction, 0, 12)
	for i := 0; i < 12; i++ {
		side := "buy"
		if i%2 == 1 {
			side = "sell"
		}
		out = append(out, Transaction{
			ID:              uuid.NewString(),
			Timestamp:       now.Add(-time.Duration(i) * 2 * time.Minute).Unix(),
			ChainID:         token.ChainID,
			TxHash:          "0xseed" + strconv.Itoa(i),
			LogIndex:        i,
			EventIndex:      i,
			BaseToken:       token.Address,
			QuoteToken:      "USDC",
			Pair:            "pool-" + strings.ToLower(token.Symbol),
			Type:            side,
			Maker:           "wallet-" + strconv.Itoa(i),
			BaseAmount:      formatFloat(10 + float64(i)),
			QuoteAmount:     formatFloat(10 * float64(i+1)),
			Price:           token.Price,
			USDAmount:       formatFloat(100 + float64(i*20)),
			USDPrice:        token.Price,
			Liquidity:       token.Liquidity,
			HolderPct:       "0",
			Decimals:        token.Decimals,
			Tx24h:           100 + i,
			NativeAmount:    "0",
			NativePrice:     "0",
			Dex:             firstDex(token.Dexes),
			MakerAlias:      "",
			TotalFee:        "0",
			TotalFeeUSD:     "0",
			IsKlineTx:       true,
			ReasonFiltering: "",
			Metadata:        map[string]any{"source": "seed"},
		})
	}
	return out
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 10, 64)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
