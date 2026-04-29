package marketdata

import (
	"context"
	"fmt"
	"strings"
	"time"
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

func (s *Service) ListTokens(ctx context.Context, filter TokenFilter) (TokenList, error) {
	tokens, total, err := s.store.ListTokens(ctx, filter)
	if err != nil {
		return TokenList{}, err
	}
	filter = normalizeFilter(filter)
	return TokenList{Data: tokens, Page: filter.Page, Limit: filter.Limit, Total: total}, nil
}

func (s *Service) SearchTokens(ctx context.Context, query string, limit int) ([]Token, error) {
	list, err := s.ListTokens(ctx, TokenFilter{Query: query, Limit: limit})
	if err != nil {
		return nil, err
	}
	return list.Data, nil
}

func (s *Service) GetToken(ctx context.Context, chainID int, address string) (Token, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return Token{}, fmt.Errorf("token address is required")
	}
	return s.store.GetToken(ctx, chainID, address)
}

func (s *Service) Prices(ctx context.Context, chainID int, tokens []string) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(tokens))
	for _, tokenAddress := range tokens {
		token, err := s.store.GetToken(ctx, chainID, tokenAddress)
		if err != nil {
			out = append(out, map[string]any{"token": tokenAddress, "price": "0"})
			continue
		}
		out = append(out, map[string]any{"token": token.Address, "price": token.Price})
	}
	return out, nil
}

func (s *Service) OHLC(ctx context.Context, chainID int, token string, bucket string, limit int) ([]OHLC, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}
	return s.store.OHLC(ctx, chainID, token, bucket, limit)
}

func (s *Service) Transactions(ctx context.Context, chainID int, token string, limit int) ([]Transaction, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}
	return s.store.Transactions(ctx, chainID, token, limit)
}

func (s *Service) Pools(ctx context.Context, chainID int, token string) ([]Pool, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}
	return s.store.Pools(ctx, chainID, token)
}

func (s *Service) Categories(ctx context.Context, limit int) ([]Category, error) {
	list, err := s.ListTokens(ctx, TokenFilter{Limit: 100})
	if err != nil {
		return nil, err
	}
	byCategory := map[string][]Token{}
	for _, token := range list.Data {
		category := token.Category
		if category == "" {
			category = "uncategorized"
		}
		byCategory[category] = append(byCategory[category], token)
	}
	out := make([]Category, 0, len(byCategory))
	for category, tokens := range byCategory {
		top := tokens[0]
		out = append(out, Category{
			ID:               category,
			Name:             strings.Title(category),
			MarketCap:        top.MarketCap,
			Volume24h:        top.Volume24h,
			Price24hChange:   top.Price24hChange,
			PriceUpCount:     len(tokens),
			PriceDownCount:   0,
			TokensCount:      len(tokens),
			TopGainers:       tokens[:1],
			Top1TokenSymbol:  top.Symbol,
			Top1TokenAddress: top.Address,
			Top1TokenName:    top.Name,
			Top1TokenLogo:    top.LogoURL,
			Top1TokenChange:  top.Price24hChange,
		})
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Service) UpsertToken(ctx context.Context, input UpsertTokenInput) (Token, error) {
	if strings.TrimSpace(input.Address) == "" {
		return Token{}, fmt.Errorf("token address is required")
	}
	return s.store.UpsertToken(ctx, input.Token)
}

func (s *Service) AppendTransaction(ctx context.Context, input AppendTransactionInput) (Transaction, error) {
	if strings.TrimSpace(input.BaseToken) == "" {
		return Transaction{}, fmt.Errorf("base token is required")
	}
	return s.store.AppendTransaction(ctx, input.Transaction)
}

func (s *Service) SaveCheckpoint(ctx context.Context, checkpoint Checkpoint) (Checkpoint, error) {
	return s.store.SaveCheckpoint(ctx, checkpoint)
}

func (s *Service) GetCheckpoint(ctx context.Context, source string) (Checkpoint, error) {
	return s.store.GetCheckpoint(ctx, source)
}
