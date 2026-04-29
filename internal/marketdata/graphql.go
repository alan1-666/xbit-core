package marketdata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/xbit/xbit-backend/internal/httpx"
	"github.com/xbit/xbit-backend/pkg/errors"
	"github.com/xbit/xbit-backend/pkg/requestid"
)

type graphQLRequest struct {
	OperationName string         `json:"operationName"`
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables"`
}

type graphQLResponse struct {
	Data   map[string]any `json:"data,omitempty"`
	Errors []graphQLError `json:"errors,omitempty"`
}

type graphQLError struct {
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions"`
}

func (h *Handler) graphql(w http.ResponseWriter, r *http.Request) {
	var req graphQLRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		h.writeGraphQLError(w, r, http.StatusBadRequest, errors.CodeValidation, "invalid json body")
		return
	}
	if req.Variables == nil {
		req.Variables = map[string]any{}
	}
	operation := inferOperationName(req.OperationName, req.Query)
	if operation == "" {
		h.writeGraphQLError(w, r, http.StatusBadRequest, errors.CodeValidation, "operationName is required")
		return
	}
	data, err := h.executeGraphQL(r, operation, req.Query, req.Variables)
	if err != nil {
		h.writeGraphQLError(w, r, http.StatusOK, errors.CodeValidation, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, graphQLResponse{Data: data})
}

func (h *Handler) executeGraphQL(r *http.Request, operation string, query string, variables map[string]any) (map[string]any, error) {
	ctx := r.Context()
	switch strings.ToLower(operation) {
	case "gettokentrending", "gettokentrendingwithdebug":
		list, err := h.service.ListTokens(ctx, graphQLTokenFilter(variables))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getTokenTrending": graphQLTokenList(list)}, nil
	case "getnewtoken":
		list, err := h.service.ListTokens(ctx, graphQLTokenFilter(variables))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getNewToken": graphQLTokenList(list)}, nil
	case "getmemetoken":
		filter := graphQLTokenFilter(variables)
		if filter.Category == "" {
			filter.Category = "meme"
		}
		list, err := h.service.ListTokens(ctx, filter)
		if err != nil {
			return nil, err
		}
		return map[string]any{"getMemeToken": graphQLTokenList(list)}, nil
	case "getfavoritetoken":
		list, err := h.service.ListTokens(ctx, graphQLTokenFilter(variables))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getFavoriteToken": graphQLTokenList(list)}, nil
	case "tokens":
		list, err := h.service.ListTokens(ctx, graphQLTokenFilter(variables))
		if err != nil {
			return nil, err
		}
		return map[string]any{"tokens": graphQLTokenList(list)}, nil
	case "getpopulartokens":
		list, err := h.service.ListTokens(ctx, TokenFilter{Limit: 20})
		if err != nil {
			return nil, err
		}
		return map[string]any{"getPopularTokens": graphQLPopularTokens(list.Data)}, nil
	case "searchtoken":
		tokens, err := h.service.SearchTokens(ctx, stringValue(variables, "input", "q", "query"), 20)
		if err != nil {
			return nil, err
		}
		return map[string]any{"searchToken": graphQLTokenPayloads(tokens)}, nil
	case "searchtokenlite":
		tokens, err := h.service.SearchTokens(ctx, stringValue(variables, "input", "q", "query"), 20)
		if err != nil {
			return nil, err
		}
		return map[string]any{"searchTokenLite": graphQLTokenPayloads(tokens)}, nil
	case "searchuniversal":
		tokens, err := h.service.SearchTokens(ctx, searchQuery(variables), 20)
		if err != nil {
			return nil, err
		}
		data := graphQLTokenPayloads(tokens)
		for _, item := range data {
			item["__typename"] = "SearchData"
		}
		return map[string]any{"searchUniversal": map[string]any{"data": data}}, nil
	case "gettokendetail", "gettokendetailmcandprice":
		token, err := h.service.GetToken(ctx, chainIDFromVariables(variables), tokenFromVariables(variables))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getTokenDetail": graphQLTokenPayload(token)}, nil
	case "gettokenmetadata":
		token, err := h.service.GetToken(ctx, chainIDFromVariables(variables), stringValue(variables, "input", "address", "token"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getTokenMetadata": graphQLTokenPayload(token)}, nil
	case "gettokensprices", "getprices":
		prices, err := h.service.Prices(ctx, chainIDFromVariables(variables), stringSliceValue(variables, "tokens"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getPrices": prices}, nil
	case "getmanytoken":
		tokens := stringSliceValue(inputMap(variables), "tokens", "addresses")
		out := make([]map[string]any, 0, len(tokens))
		for _, address := range tokens {
			token, err := h.service.GetToken(ctx, chainIDFromVariables(variables), address)
			if err == nil {
				out = append(out, graphQLTokenPayload(token))
			}
		}
		return map[string]any{"getManyToken": out}, nil
	case "getohlc":
		points, err := h.service.OHLC(ctx, chainIDFromVariables(variables), tokenFromVariables(variables), stringValue(inputMap(variables), "bucket", "interval"), intValue(inputMap(variables), 120, "limit"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getOHLC": graphQLOHLC(points)}, nil
	case "gettransactions":
		txs, err := h.service.Transactions(ctx, chainIDFromVariables(variables), tokenFromVariables(variables), intValue(inputMap(variables), 50, "limit"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getTransactions": map[string]any{"fromTimestamp": fromTimestamp(txs), "data": graphQLTransactions(txs)}}, nil
	case "gettradingtransactions":
		txs, err := h.service.Transactions(ctx, chainIDFromVariables(variables), tokenFromVariables(variables), intValue(inputMap(variables), 50, "limit"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getTradingTransactions": map[string]any{"cursor": strconv.FormatInt(fromTimestamp(txs), 10), "data": graphQLTransactions(txs)}}, nil
	case "getpooltransactions":
		txs, err := h.service.Transactions(ctx, chainIDFromVariables(variables), tokenFromVariables(variables), intValue(inputMap(variables), 50, "limit"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getPoolTransactions": map[string]any{"data": graphQLTransactions(txs), "numberOfPools": 1, "fromTimestamp": fromTimestamp(txs), "liquidity": "0"}}, nil
	case "gettokenpoolinfo":
		pools, err := h.service.Pools(ctx, chainIDFromVariables(variables), tokenFromVariables(variables))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getTokenPoolInfo": graphQLPools(pools)}, nil
	case "gettokensymbols":
		tokens := stringSliceValue(variables, "tokens")
		out := make([]map[string]any, 0, len(tokens))
		for _, address := range tokens {
			token, err := h.service.GetToken(ctx, chainIDFromVariables(variables), address)
			if err == nil {
				out = append(out, map[string]any{"token": token.Address, "symbol": token.Symbol, "chainId": token.ChainID})
			}
		}
		return map[string]any{"getTokenSymbols": out}, nil
	case "getallcategories":
		categories, err := h.service.Categories(ctx, intValue(inputMap(variables), 20, "limit"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getAllCategories": graphQLCategoryList(categories)}, nil
	case "getcategorystatistic":
		categories, err := h.service.Categories(ctx, 1)
		if err != nil {
			return nil, err
		}
		if len(categories) == 0 {
			return map[string]any{"getCategoryStatistic": graphQLEmptyCategoryStatistic()}, nil
		}
		return map[string]any{"getCategoryStatistic": graphQLCategoryStatistic(categories[0])}, nil
	case "tokensbycategory", "xstockstokens":
		list, err := h.service.ListTokens(ctx, graphQLTokenFilter(variables))
		if err != nil {
			return nil, err
		}
		return map[string]any{"tokensByCategory": graphQLTokenList(list)}, nil
	case "getcryptocurrencyprice":
		return map[string]any{"getCryptoCurrencyPrice": []map[string]any{{"symbol": "SOL", "usdPrice": "145.23"}, {"symbol": "ETH", "usdPrice": "3200.50"}}}, nil
	case "addtofavorite", "removetokenfavorite", "updatefavoritetokenorder":
		return map[string]any{graphQLOperationKey(operation): true}, nil
	case "gettokeninsight":
		return map[string]any{"getTokenInsight": map[string]any{"top10Holder": "0", "DevHold": "0", "Holders": "0", "Sniper": "0", "Insider": "0", "Bundler": "0", "LPBurned": "0", "LPMint": "0", "DexPaid": "0"}}, nil
	default:
		return h.defaultGraphQLResponse(operation, query, variables), nil
	}
}

func (h *Handler) defaultGraphQLResponse(operation string, query string, variables map[string]any) map[string]any {
	key := graphQLOperationKey(operation)
	switch {
	case strings.Contains(query, "data {"):
		return map[string]any{key: map[string]any{"data": []any{}, "page": 1, "limit": 50, "total": 0}}
	case strings.Contains(strings.ToLower(operation), "transaction"):
		return map[string]any{key: map[string]any{"data": []any{}, "fromTimestamp": 0, "cursor": ""}}
	default:
		return map[string]any{key: []any{}}
	}
}

func (h *Handler) writeGraphQLError(w http.ResponseWriter, r *http.Request, status int, code string, message string) {
	meta := map[string]any{"retryable": false}
	if traceID := requestid.FromContext(r.Context()); traceID != "" {
		meta["traceId"] = traceID
	}
	httpx.WriteJSON(w, status, graphQLResponse{Errors: []graphQLError{{
		Message: message,
		Extensions: map[string]any{
			"code": code,
			"meta": meta,
		},
	}}})
}

func inferOperationName(operationName string, query string) string {
	if strings.TrimSpace(operationName) != "" {
		return strings.TrimSpace(operationName)
	}
	known := []string{
		"GetTokenTrendingWithDebug", "GetTokenTrending", "GetNewToken", "GetMemeToken", "GetFavoriteToken", "Tokens", "GetTokenDetailMCAndPrice", "GetTokenDetail",
		"getTokenMetadata", "getTokensPrices", "GetPrices", "getPrices", "SearchTokenLite", "SearchToken", "SearchUniversal", "GetOHLC", "getOHLC", "GetPopularTokens",
		"GetAllCategories", "GetCategoryStatistic", "TokensByCategory", "XStocksTokens", "GetTransactions", "GetTradingTransactions", "GetPoolTransactions",
		"GetTokenPoolInfo", "GetTokenSymbols", "getManyToken", "getCryptoCurrencyPrice", "AddToFavorite", "RemoveTokenFavorite", "UpdateFavoriteTokenOrder",
	}
	for _, name := range known {
		if strings.Contains(query, name) {
			return name
		}
	}
	return ""
}

func graphQLOperationKey(operation string) string {
	switch strings.ToLower(operation) {
	case "gettokentrending", "gettokentrendingwithdebug":
		return "getTokenTrending"
	case "getnewtoken":
		return "getNewToken"
	case "getmemetoken":
		return "getMemeToken"
	case "getfavoritetoken":
		return "getFavoriteToken"
	case "gettokendetail", "gettokendetailmcandprice":
		return "getTokenDetail"
	case "gettokensprices", "getprices":
		return "getPrices"
	case "getohlc":
		return "getOHLC"
	case "getallcategories":
		return "getAllCategories"
	case "getcategorystatistic":
		return "getCategoryStatistic"
	case "tokensbycategory", "xstockstokens":
		return "tokensByCategory"
	case "addtofavorite":
		return "addToFavorite"
	case "removetokenfavorite":
		return "removeTokenFavorite"
	case "updatefavoritetokenorder":
		return "updateFavoriteTokenOrder"
	default:
		if operation == "" {
			return operation
		}
		return strings.ToLower(operation[:1]) + operation[1:]
	}
}

func graphQLTokenFilter(variables map[string]any) TokenFilter {
	input := inputMap(variables)
	return TokenFilter{
		Query:    stringValue(input, "q", "query", "keyword", "search"),
		Category: stringValue(input, "category", "categoryId"),
		ChainID:  chainIDFromVariables(variables),
		Page:     intValue(input, 1, "page", "offset"),
		Limit:    intValue(input, 50, "limit", "pageSize"),
	}
}

func graphQLTokenList(list TokenList) map[string]any {
	return map[string]any{
		"page":  list.Page,
		"limit": list.Limit,
		"total": list.Total,
		"data":  graphQLTokenPayloads(list.Data),
	}
}

func graphQLTokenPayloads(tokens []Token) []map[string]any {
	out := make([]map[string]any, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, graphQLTokenPayload(token))
	}
	return out
}

func graphQLTokenPayload(token Token) map[string]any {
	payload := map[string]any{
		"address":                  token.Address,
		"token":                    token.Address,
		"chainId":                  token.ChainID,
		"symbol":                   token.Symbol,
		"name":                     token.Name,
		"decimals":                 token.Decimals,
		"logo":                     token.LogoURL,
		"logoUrl":                  token.LogoURL,
		"image":                    token.LogoURL,
		"avatarUrl":                token.LogoURL,
		"thumbnailUrl":             token.LogoURL,
		"price":                    token.Price,
		"usdPrice":                 token.Price,
		"firstPrice":               token.Price,
		"openPrice":                token.Price,
		"openPrice24h":             token.Price,
		"athPrice":                 token.Price,
		"atlPrice":                 token.Price,
		"price1mAgo":               token.Price,
		"price5mAgo":               token.Price,
		"price1hAgo":               token.Price,
		"price24hAgo":              token.Price,
		"price1mChange":            "0",
		"price5mChange":            "0",
		"price1hChange":            token.Price24hChange,
		"price6hChange":            token.Price24hChange,
		"price24hChange":           token.Price24hChange,
		"marketcap":                token.MarketCap,
		"marketCap":                token.MarketCap,
		"marketCap5mChangeUsd":     "0",
		"liquidity":                token.Liquidity,
		"initLiquidity":            token.Liquidity,
		"volume1m":                 "0",
		"volume5m":                 "0",
		"volume1h":                 token.Volume24h,
		"volume6h":                 token.Volume24h,
		"volume24h":                token.Volume24h,
		"txs1m":                    0,
		"txs5m":                    0,
		"txs1h":                    10,
		"txs6h":                    30,
		"txs24h":                   120,
		"buyTxs1m":                 0,
		"buyTxs5m":                 0,
		"buyTxs1h":                 6,
		"buyTxs6h":                 18,
		"buyTxs24h":                72,
		"sellTxs1m":                0,
		"sellTxs5m":                0,
		"sellTxs1h":                4,
		"sellTxs6h":                12,
		"sellTxs24h":               48,
		"numberOfHolder":           token.Holders,
		"holders":                  token.Holders,
		"dexes":                    token.Dexes,
		"createdTime":              token.CreatedAt.Unix(),
		"createdTimeRaw":           token.CreatedAt,
		"totalSupply":              "1000000000",
		"circulatingSupply":        "1000000000",
		"tags":                     []string{token.Category},
		"isBlacklisted":            false,
		"isHoneypot":               false,
		"mintDisable":              true,
		"ownershipRenounced":       true,
		"isFavorite":               false,
		"isHotToken":               token.Category == "meme",
		"isOG":                     false,
		"isExclusive":              false,
		"isXStock":                 false,
		"burnRatio":                "0",
		"burnStatus":               "unknown",
		"top10HolderRate":          "0",
		"ratTraderAmountRate":      "0",
		"internalMarketProgress":   "0",
		"turnoverRate24h":          "0",
		"creator":                  "",
		"contractCreator":          "",
		"contractOwner":            "",
		"metadataCustom":           token.Metadata,
		"isMigrated":               false,
		"totalFee":                 "0",
		"numberMigratedTokenByDev": 0,
		"topTrending":              0,
		"numberProTrader":          0,
		"tweetId":                  "",
		"twitterNameChangeCount":   0,
		"twitterUrl":               stringFromMetadata(token.Metadata, "twitterUrl"),
		"telegramUrl":              stringFromMetadata(token.Metadata, "telegramUrl"),
		"website":                  stringFromMetadata(token.Metadata, "website"),
		"advertisesOnDex":          false,
		"bundlerHoldingPercent":    "0",
		"txBySniperPct":            "0",
		"devHold":                  "0",
		"sameSourceWallet":         0,
		"smartMoneyPct":            "0",
		"smartMoneyHolder":         0,
		"devLaunched":              0,
		"devMigrated":              0,
		"insider":                  "0",
		"top10Holder":              "0",
		"sniperHoldPct":            "0",
		"sniperHoldAmount":         "0",
		"sniperCount":              0,
		"botHolder":                "0",
		"favoriteAt":               nil,
		"memeTooltip":              map[string]any{"bundlerCount": 0, "bundlerHoldAmount": "0", "devHoldAmount": "0", "insiderCount": 0, "insiderHoldAmount": "0", "top10HolderAmount": "0"},
		"info":                     map[string]any{"logoUrl": token.LogoURL, "avatarUrl": token.LogoURL, "bannerUrl": "", "dexScreenerBoosts": nil, "socials": []any{}, "websites": []any{}},
		"health":                   map[string]any{"noBlackListWhiteListFunction": true, "notMint": true, "burnt": false, "top10": "0"},
		"security":                 map[string]any{"buyTax": "0", "sellTax": "0"},
		"totalTransactions":        map[string]any{"numberOfPurchases5m": 0, "numberOfPurchases1h": 6, "numberOfPurchases6h": 18, "numberOfPurchases24h": 72, "numberOfSales5m": 0, "numberOfSales1h": 4, "numberOfSales6h": 12, "numberOfSales24h": 48},
		"totalAmount":              map[string]any{"totalBuyAmount5m": "0", "totalBuyAmount1h": token.Volume24h, "totalBuyAmount6h": token.Volume24h, "totalBuyAmount24h": token.Volume24h, "totalSellAmount5m": "0", "totalSellAmount1h": "0", "totalSellAmount6h": "0", "totalSellAmount24h": "0"},
		"numberUniqueAddresses":    map[string]any{"numberOfBuyAddress5m": 0, "numberOfBuyAddress1h": 6, "numberOfBuyAddress6h": 18, "numberOfBuyAddress24h": 72, "numberOfSellAddress5m": 0, "numberOfSellAddress1h": 4, "numberOfSellAddress6h": 12, "numberOfSellAddress24h": 48},
		"ohlc":                     []map[string]any{{"ts": token.UpdatedAt.Unix(), "token": token.Address, "open": token.Price, "usdVolume": token.Volume24h}},
		"debug":                    map[string]any{"calculatedAt": token.UpdatedAt.Unix(), "score": 0, "debugData": map[string]any{}},
	}
	return payload
}

func graphQLPopularTokens(tokens []Token) []map[string]any {
	out := make([]map[string]any, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, map[string]any{"token": token.Address, "chainId": token.ChainID, "symbol": token.Symbol, "name": token.Name, "logoUrl": token.LogoURL, "hot": token.Category == "meme"})
	}
	return out
}

func graphQLOHLC(points []OHLC) []map[string]any {
	out := make([]map[string]any, 0, len(points))
	for _, point := range points {
		out = append(out, map[string]any{"ts": point.TS, "token": point.Token, "open": point.Open, "high": point.High, "low": point.Low, "close": point.Close, "price": point.Price, "tokenVolume": point.TokenVolume, "usdVolume": point.USDVolume})
	}
	return out
}

func graphQLTransactions(txs []Transaction) []map[string]any {
	out := make([]map[string]any, 0, len(txs))
	for _, tx := range txs {
		out = append(out, map[string]any{
			"timestamp": tx.Timestamp, "chainId": tx.ChainID, "txHash": tx.TxHash, "logIndex": tx.LogIndex, "eventIndex": tx.EventIndex,
			"baseToken": tx.BaseToken, "quoteToken": tx.QuoteToken, "pair": tx.Pair, "type": tx.Type, "maker": tx.Maker,
			"baseAmount": tx.BaseAmount, "quoteAmount": tx.QuoteAmount, "price": tx.Price, "usdAmount": tx.USDAmount, "usdPrice": tx.USDPrice,
			"liquidity": tx.Liquidity, "holderPct": tx.HolderPct, "isInsider": tx.IsInsider, "isNativeWallet": tx.IsNativeWallet,
			"isHugeValue": tx.IsHugeValue, "isWhale": tx.IsWhale, "isDev": tx.IsDev, "isFreshWallet": tx.IsFreshWallet, "isNewActivity": tx.IsNewActivity,
			"isPoolContract": tx.IsPoolContract, "isKOL": tx.IsKOL, "isSmartMoney": tx.IsSmartMoney, "isTopTrader": tx.IsTopTrader, "isSniper": tx.IsSniper, "isBundler": tx.IsBundler,
			"totalSupply": tx.TotalSupply, "decimals": tx.Decimals, "tx24h": tx.Tx24h, "holdingProgress": tx.HoldingProgress,
			"nativeAmount": tx.NativeAmount, "nativePrice": tx.NativePrice, "isSingleSideTransaction": tx.IsSingleSideTransaction, "dex": tx.Dex,
			"MakerAlias": tx.MakerAlias, "totalFee": tx.TotalFee, "totalFeeUSD": tx.TotalFeeUSD, "isKlineTx": tx.IsKlineTx, "reasonFiltering": tx.ReasonFiltering,
			"totalAddBaseLiq": "0", "totalAddQuoteLiq": "0",
		})
	}
	return out
}

func graphQLPools(pools []Pool) []map[string]any {
	out := make([]map[string]any, 0, len(pools))
	for _, pool := range pools {
		out = append(out, map[string]any{"address": pool.Address, "chainId": pool.ChainID, "baseToken": pool.BaseToken, "createdAt": pool.CreatedAt, "quoteToken": pool.QuoteToken, "quoteTokenPrice": pool.QuoteTokenPrice, "quoteSymbol": pool.QuoteSymbol, "baseSymbol": pool.BaseSymbol, "baseTokenLiquidity": pool.BaseTokenLiquidity, "quoteLiquidity": pool.QuoteLiquidity, "usdLiquidity": pool.USDLiquidity, "dex": pool.Dex})
	}
	return out
}

func graphQLCategoryList(categories []Category) map[string]any {
	data := make([]map[string]any, 0, len(categories))
	for _, category := range categories {
		data = append(data, graphQLCategory(category))
	}
	return map[string]any{"data": data, "limit": 50, "page": 1}
}

func graphQLCategory(category Category) map[string]any {
	return map[string]any{
		"categoryId": category.ID, "name": category.Name, "marketCap": category.MarketCap, "volume24h": category.Volume24h, "price24hChange": category.Price24hChange,
		"priceUpCount": category.PriceUpCount, "priceDownCount": category.PriceDownCount, "tokensCount": category.TokensCount, "topGainers": graphQLTokenPayloads(category.TopGainers),
		"top1TokenSymbol": category.Top1TokenSymbol, "top1TokenAddress": category.Top1TokenAddress, "top1TokenName": category.Top1TokenName, "top1TokenLogo": category.Top1TokenLogo,
		"top1TokenP24hChange": category.Top1TokenChange, "volume1hHistory": []string{}, "price1hChangeHistory": []string{},
	}
}

func graphQLCategoryStatistic(category Category) map[string]any {
	return map[string]any{"marketCap": category.MarketCap, "volume24h": category.Volume24h, "price24hChange": category.Price24hChange, "priceUpCount": category.PriceUpCount, "priceDownCount": category.PriceDownCount, "tokensCount": category.TokensCount}
}

func graphQLEmptyCategoryStatistic() map[string]any {
	return map[string]any{"marketCap": "0", "volume24h": "0", "price24hChange": "0", "priceUpCount": 0, "priceDownCount": 0, "tokensCount": 0}
}

func inputMap(variables map[string]any) map[string]any {
	if input, ok := variables["input"]; ok {
		if m := asMap(input); len(m) > 0 {
			return m
		}
	}
	return variables
}

func asMap(value any) map[string]any {
	switch v := value.(type) {
	case map[string]any:
		return v
	case json.RawMessage:
		var out map[string]any
		_ = json.Unmarshal(v, &out)
		return out
	default:
		return map[string]any{}
	}
}

func tokenFromVariables(variables map[string]any) string {
	input := inputMap(variables)
	return firstNonEmpty(stringValue(input, "token", "address", "baseAddress"), stringValue(variables, "token", "address", "input"))
}

func searchQuery(variables map[string]any) string {
	input := inputMap(variables)
	return firstNonEmpty(stringValue(input, "keyword", "query", "q"), stringValue(variables, "input"))
}

func chainIDFromVariables(variables map[string]any) int {
	input := inputMap(variables)
	if value := intValue(input, 0, "chainId"); value != 0 {
		return value
	}
	if value := intValue(variables, 0, "chainId"); value != 0 {
		return value
	}
	chain := strings.ToUpper(firstNonEmpty(stringValue(input, "chain", "chainType"), stringValue(variables, "chain")))
	switch chain {
	case "SOLANA", "SOL":
		return 501
	case "BSC":
		return 56
	case "MON":
		return 10143
	case "EVM", "ETH", "ETHEREUM":
		return 1
	default:
		return 0
	}
}

func fromTimestamp(txs []Transaction) int64 {
	if len(txs) == 0 {
		return 0
	}
	return txs[len(txs)-1].Timestamp
}

func stringValue(m map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := m[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		case json.Number:
			return v.String()
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			return strconv.Itoa(v)
		case int64:
			return strconv.FormatInt(v, 10)
		default:
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func stringSliceValue(m map[string]any, keys ...string) []string {
	for _, key := range keys {
		value, ok := m[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case []string:
			return v
		case []any:
			out := make([]string, 0, len(v))
			for _, item := range v {
				out = append(out, strings.TrimSpace(fmt.Sprint(item)))
			}
			return out
		}
	}
	return nil
}

func intValue(m map[string]any, fallback int, keys ...string) int {
	for _, key := range keys {
		value, ok := m[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			parsed, err := strconv.Atoi(v)
			if err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringFromMetadata(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, _ := metadata[key].(string)
	return value
}
