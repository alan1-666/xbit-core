package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	data, err := h.executeGraphQL(r, operation, req.Variables)
	if err != nil {
		h.writeGraphQLError(w, r, http.StatusOK, errors.CodeValidation, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, graphQLResponse{Data: data})
}

func (h *Handler) executeGraphQL(r *http.Request, operation string, variables map[string]any) (map[string]any, error) {
	ctx := r.Context()
	switch strings.ToLower(operation) {
	case "getexchangemeta":
		return map[string]any{"getExchangeMeta": graphQLExchangeMeta()}, nil
	case "getexchangemetav2":
		return map[string]any{"getExchangeMetaV2": graphQLExchangeMetaV2()}, nil
	case "config":
		return map[string]any{"config": map[string]any{"platformFee": 30, "xstockPlatformFee": 30}}, nil
	case "getnetworkfee":
		return map[string]any{"getNetworkFee": h.graphQLNetworkFee(ctx)}, nil
	case "createorder":
		order, err := h.service.CreateOrder(ctx, graphQLCreateOrderInput(variables))
		if err != nil {
			return nil, err
		}
		return map[string]any{"createOrder": graphQLOrder(order)}, nil
	case "saveweb3order":
		input := inputMap(variables)
		order, err := h.service.CreateOrder(ctx, CreateOrderInput{
			UserID:          stringValue(input, "userAddress", "user", "userId"),
			ChainType:       stringValue(input, "chain", "chainId"),
			WalletAddress:   stringValue(input, "userAddress", "walletAddress"),
			OrderType:       OrderTypeMarket,
			Side:            SideBuy,
			InputToken:      "web3",
			OutputToken:     "web3",
			InputAmount:     "0",
			ClientRequestID: stringValue(input, "txid", "txId"),
			RouteSnapshot:   map[string]any{"source": "saveWeb3Order", "txid": stringValue(input, "txid", "txId")},
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"saveWeb3Order": graphQLOrder(order)}, nil
	case "cancelorder":
		id := stringValue(variables, "id", "orderId")
		order, err := h.service.CancelOrder(ctx, id)
		if err != nil {
			return nil, err
		}
		return map[string]any{"cancelOrder": graphQLOrder(order)}, nil
	case "getpendingorders":
		orders, err := h.service.ListOrders(ctx, graphQLSearchOrdersInput(variables, OrderStatusPending))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getPendingOrders": graphQLOrderList(orders)}, nil
	case "orders":
		orders, err := h.service.ListOrders(ctx, graphQLSearchOrdersInput(variables, ""))
		if err != nil {
			return nil, err
		}
		return map[string]any{"orders": graphQLOrders(orders)}, nil
	case "orderhistory", "gettransactions", "getalltransactions", "getuncompletedorders":
		orders, err := h.service.ListOrders(ctx, graphQLSearchOrdersInput(variables, ""))
		if err != nil {
			return nil, err
		}
		key := graphQLOperationKey(operation)
		if strings.EqualFold(operation, "GetUncompletedOrders") {
			return map[string]any{key: graphQLOrderList(orders)}, nil
		}
		return map[string]any{key: graphQLOrders(orders)}, nil
	case "historystatistic":
		return map[string]any{"historyStatistic": map[string]any{
			"totalOrder":     "0",
			"totalBuyQuote":  "0",
			"totalBuyUsd":    "0",
			"totalSellQuote": "0",
			"totalSellUsd":   "0",
		}}, nil
	case "getallpossibleroutes":
		return map[string]any{"getAllPossibleRoutes": h.graphQLAllPossibleRoutes(ctx, variables)}, nil
	case "confirmroute":
		input := inputMap(variables)
		return map[string]any{"confirmRoute": map[string]any{
			"requestId": stringValue(input, "requestId"),
			"status":    "Confirmed",
			"routeErr":  nil,
		}}, nil
	case "createtx":
		return map[string]any{"createTx": graphQLCreateTx(variables)}, nil
	case "signtx":
		return map[string]any{"signTx": graphQLSignTx(variables)}, nil
	case "checkstatus":
		return map[string]any{"checkStatus": graphQLCheckStatus(variables)}, nil
	case "getquotev2", "quoterelay":
		return map[string]any{graphQLOperationKey(operation): h.graphQLQuoteV2(ctx, variables)}, nil
	default:
		return nil, fmt.Errorf("unsupported operation %q", operation)
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
		"GetExchangeMetaV2", "GetExchangeMeta", "GetAllPossibleRoutes", "ConfirmRoute", "CreateTx", "SignTx", "CheckStatus", "GetQuoteV2", "QuoteRelay",
		"createOrder", "saveWeb3Order", "cancelOrder", "getNetworkFee", "getPendingOrders", "orders", "orderHistory", "getTransactions", "getAllTransactions", "GetUncompletedOrders", "historyStatistic", "config",
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
	case "getquotev2":
		return "getQuoteV2"
	case "quoterelay":
		return "quoteRelay"
	case "getalltransactions":
		return "getAllTransactions"
	case "gettransactions":
		return "getTransactions"
	case "getuncompletedorders":
		return "getUncompletedOrders"
	default:
		if operation == "" {
			return operation
		}
		return strings.ToLower(operation[:1]) + operation[1:]
	}
}

func graphQLCreateOrderInput(variables map[string]any) CreateOrderInput {
	input := inputMap(variables)
	transactionType := strings.ToLower(stringValue(input, "transactionType", "side"))
	side := SideBuy
	if transactionType == "sell" {
		side = SideSell
	}

	baseAddress := stringValue(input, "baseAddress", "baseToken")
	quoteAddress := stringValue(input, "quoteAddress", "quoteToken")
	baseAmount := stringValue(input, "baseAmount")
	quoteAmount := stringValue(input, "quoteAmount")
	orderType := strings.ToLower(stringValue(input, "type", "orderType"))
	if orderType == "" {
		orderType = OrderTypeMarket
	}
	if orderType != OrderTypeMarket && orderType != OrderTypeLimit {
		orderType = OrderTypeMarket
	}

	out := CreateOrderInput{
		UserID:          stringValue(input, "userAddress", "userId", "user"),
		ChainType:       stringValue(input, "chainId", "chain", "chainType"),
		WalletAddress:   stringValue(input, "walletAddress", "userAddress"),
		OrderType:       orderType,
		Side:            side,
		SlippageBps:     slippageBps(input),
		ClientRequestID: stringValue(input, "clientRequestId", "requestId", "cloid"),
		RouteSnapshot: map[string]any{
			"source":           "graphql-facade",
			"originalInput":    input,
			"transactionType":  transactionType,
			"submitEngineType": stringValue(input, "engine"),
		},
	}
	if side == SideSell {
		out.InputToken = baseAddress
		out.OutputToken = quoteAddress
		out.InputAmount = firstNonEmpty(baseAmount, quoteAmount)
	} else {
		out.InputToken = quoteAddress
		out.OutputToken = baseAddress
		out.InputAmount = firstNonEmpty(quoteAmount, baseAmount)
	}
	return out
}

func graphQLSearchOrdersInput(variables map[string]any, defaultStatus string) SearchOrdersInput {
	input := inputMap(variables)
	status := strings.ToLower(stringValue(input, "status"))
	if status == "" {
		status = defaultStatus
	}
	return SearchOrdersInput{
		UserID: stringValue(input, "userAddress", "userId", "user"),
		Status: graphQLStatusToInternal(status),
		Limit:  intValue(input, 50, "limit", "pageSize", "size"),
	}
}

func graphQLStatusToInternal(status string) string {
	switch strings.ToLower(status) {
	case "pending":
		return OrderStatusPending
	case "confirmed", "submitted", "filled":
		return OrderStatusSubmitted
	case "completed":
		return OrderStatusConfirmed
	case "canceled", "cancelled":
		return OrderStatusCancelled
	case "failed":
		return OrderStatusFailed
	default:
		return status
	}
}

func graphQLOrderList(orders []Order) map[string]any {
	return map[string]any{
		"total":  len(orders),
		"orders": graphQLOrders(orders),
	}
}

func graphQLOrders(orders []Order) []map[string]any {
	out := make([]map[string]any, 0, len(orders))
	for _, order := range orders {
		out = append(out, graphQLOrder(order))
	}
	return out
}

func graphQLOrder(order Order) map[string]any {
	baseAddress, quoteAddress := order.OutputToken, order.InputToken
	baseAmount, quoteAmount := order.ExpectedOutputAmount, order.InputAmount
	transactionType := "Buy"
	if order.Side == SideSell {
		baseAddress, quoteAddress = order.InputToken, order.OutputToken
		baseAmount, quoteAmount = order.InputAmount, order.ExpectedOutputAmount
		transactionType = "Sell"
	}

	return map[string]any{
		"id":                           order.ID,
		"createdAt":                    order.CreatedAt,
		"updatedAt":                    order.UpdatedAt,
		"deletedAt":                    nil,
		"filledAt":                     order.FilledAt,
		"exitAt":                       order.FilledAt,
		"transactionType":              transactionType,
		"type":                         graphQLOrderType(order.OrderType),
		"baseAddress":                  baseAddress,
		"quoteAddress":                 quoteAddress,
		"userAddress":                  order.WalletAddress,
		"limitPrice":                   "0",
		"baseAmount":                   baseAmount,
		"quoteAmount":                  quoteAmount,
		"exit":                         nil,
		"tp":                           nil,
		"tp2":                          nil,
		"sl":                           nil,
		"status":                       graphQLOrderStatus(order.Status),
		"txid":                         order.TxHash,
		"txId":                         order.TxHash,
		"openTxId":                     order.TxHash,
		"closeTxId":                    "",
		"chainId":                      order.ChainType,
		"baseDecimal":                  18,
		"baseSymbol":                   baseAddress,
		"quoteSymbol":                  quoteAddress,
		"slippage":                     bpsToPercent(order.SlippageBps),
		"marketCap":                    "0",
		"openPrice":                    "0",
		"openQuoteUsdRate":             "0",
		"triggerPrice":                 nil,
		"callbackRate":                 nil,
		"trailingOrderTriggered":       false,
		"triggerAt":                    nil,
		"mevProtect":                   false,
		"priorityFeePrice":             nil,
		"doublePrincipalAfterPurchase": false,
		"pnl":                          "0",
		"closePriceQuote":              "0",
		"closePriceUsd":                "0",
		"slippageLoss":                 "0",
		"slippageLossAmount":           "0",
		"platformFee":                  "0.003",
		"platformFeeAmount":            order.RouteSnapshot["platformFeeAmount"],
		"antiMevFee":                   "0",
		"antiMevFeeAmount":             "0",
		"gasFee":                       "0",
		"gasFeeAmount":                 "0",
		"pumpFee":                      "0",
		"pumpFeeAmount":                "0",
		"priorityFee":                  "0",
		"priorityFeeAmount":            "0",
		"submitCode":                   order.FailureCode,
		"isXStock":                     false,
	}
}

func graphQLOrderType(orderType string) string {
	if orderType == OrderTypeLimit {
		return "Limit"
	}
	return "Market"
}

func graphQLOrderStatus(status string) string {
	switch status {
	case OrderStatusPending:
		return "Pending"
	case OrderStatusSubmitted:
		return "Confirmed"
	case OrderStatusConfirmed:
		return "Completed"
	case OrderStatusCancelled:
		return "Canceled"
	default:
		return "Canceled"
	}
}

func (h *Handler) graphQLNetworkFee(ctx context.Context) map[string]any {
	solana, err := h.service.GetNetworkFee(ctx, "SOLANA")
	if err != nil {
		solana = defaultNetworkFee("SOLANA", time.Now().UTC())
	}
	return map[string]any{
		"solana":   solanaFeePayload(solana),
		"ethereum": evmFeePayload("1"),
		"bsc":      evmFeePayload("1"),
		"mon":      evmFeePayload("1"),
	}
}

func solanaFeePayload(fee NetworkFee) map[string]any {
	return map[string]any{
		"priorityFeePrice":  fee.PriorityFeePrice,
		"feeAccount":        "",
		"platformFee":       fee.PlatformFeeBps,
		"xstockPlatformFee": fee.PlatformFeeBps,
		"maxComputeUnits":   fee.MaxComputeUnits,
		"autoTipFee":        firstNonEmpty(fee.AutoTipFee, "0"),
		"minTipFee":         firstNonEmpty(fee.MinTipFee, "0"),
	}
}

func evmFeePayload(baseFee string) map[string]any {
	level := func(priority string, maxFee string) map[string]any {
		return map[string]any{
			"suggestedMaxPriorityFeePerGas": priority,
			"suggestedMaxFeePerGas":         maxFee,
			"minWaitTimeEstimate":           1,
			"maxWaitTimeEstimate":           30,
		}
	}
	return map[string]any{
		"low":              level("1", "2"),
		"medium":           level("2", "3"),
		"high":             level("3", "4"),
		"estimatedBaseFee": baseFee,
	}
}

func graphQLExchangeMeta() map[string]any {
	blockchains := []map[string]any{
		blockchainPayload("SOLANA", "Solana", "SOLANA", 9),
		blockchainPayload("EVM", "Ethereum", "EVM", 18),
		blockchainPayload("BSC", "BNB Chain", "BSC", 18),
		blockchainPayload("MON", "Monad", "MON", 18),
	}
	tokens := []map[string]any{
		tokenPayload("SOL", "So11111111111111111111111111111111111111112", "SOLANA", "Solana", 9),
		tokenPayload("ETH", "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "EVM", "Ethereum", 18),
		tokenPayload("USDC", "usdc", "SOLANA", "USD Coin", 6),
	}
	return map[string]any{
		"tokens":        tokens,
		"popularTokens": tokens,
		"swappers": []map[string]any{{
			"id":           "internal-mvp",
			"swapperId":    "internal-mvp",
			"title":        "Internal MVP",
			"logo":         "",
			"swapperGroup": "xbit",
			"types":        []string{"Swap"},
			"enabled":      true,
		}},
		"blockchains": blockchains,
	}
}

func graphQLExchangeMetaV2() map[string]any {
	return map[string]any{
		"chains": []map[string]any{
			{"chainId": "SOLANA", "chainImage": "", "chainName": "Solana", "tokens": []map[string]any{tokenV2Payload("SOL", "So11111111111111111111111111111111111111112", "Solana", 9), tokenV2Payload("USDC", "usdc", "USD Coin", 6)}},
			{"chainId": "EVM", "chainImage": "", "chainName": "Ethereum", "tokens": []map[string]any{tokenV2Payload("ETH", "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "Ethereum", 18), tokenV2Payload("USDC", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "USD Coin", 6)}},
		},
	}
}

func blockchainPayload(id string, name string, chainID string, decimals int) map[string]any {
	return map[string]any{
		"id":              id,
		"name":            name,
		"chainId":         chainID,
		"defaultDecimals": decimals,
		"addressPatterns": []string{},
		"feeAssets":       []string{},
		"logo":            "",
		"displayName":     name,
		"shortName":       id,
		"sort":            0,
		"color":           "",
		"enabled":         true,
		"type":            "mainnet",
		"info":            map[string]any{},
	}
}

func tokenPayload(symbol string, address string, chain string, name string, decimals int) map[string]any {
	return map[string]any{
		"id":                chain + ":" + address,
		"address":           address,
		"blockChain":        chain,
		"symbol":            symbol,
		"name":              name,
		"image":             "",
		"usdPrice":          "0",
		"decimals":          decimals,
		"isPopular":         true,
		"isSecondaryCoin":   false,
		"coinSource":        "internal",
		"coinSourceUrl":     "",
		"supportedSwappers": []string{"internal-mvp"},
		"chainDetail":       blockchainPayload(chain, chain, chain, decimals),
	}
}

func tokenV2Payload(symbol string, address string, name string, decimals int) map[string]any {
	return map[string]any{
		"address":    address,
		"symbol":     symbol,
		"name":       name,
		"image":      "",
		"decimals":   decimals,
		"usdPrice":   "0",
		"relayExtra": map[string]any{"id": address},
		"rangoExtra": map[string]any{"id": address},
	}
}

func (h *Handler) graphQLAllPossibleRoutes(ctx context.Context, variables map[string]any) map[string]any {
	input := inputMap(variables)
	quote, err := h.service.Quote(ctx, QuoteRequest{
		ChainType:   stringValue(input, "fromBlockchain", "chainType"),
		InputToken:  firstNonEmpty(stringValue(input, "fromTokenAddress"), stringValue(input, "fromSymbol")),
		OutputToken: firstNonEmpty(stringValue(input, "toTokenAddress"), stringValue(input, "toSymbol")),
		InputAmount: firstNonEmpty(stringValue(input, "amount"), "0"),
		SlippageBps: slippageBps(input),
	})
	if err != nil {
		return map[string]any{"error": err.Error(), "errorCode": 400, "results": []any{}, "diagnosisMessages": []string{err.Error()}, "traceId": 0}
	}
	return map[string]any{
		"from":              map[string]any{"blockchain": stringValue(input, "fromBlockchain"), "symbol": stringValue(input, "fromSymbol"), "address": stringValue(input, "fromTokenAddress")},
		"to":                map[string]any{"blockchain": stringValue(input, "toBlockchain"), "symbol": stringValue(input, "toSymbol"), "address": stringValue(input, "toTokenAddress")},
		"requestAmount":     quote.InputAmount,
		"routeId":           quote.RouteID,
		"diagnosisMessages": []string{},
		"error":             nil,
		"errorCode":         0,
		"traceId":           0,
		"results": []map[string]any{{
			"requestId":                         quote.ID,
			"outputAmount":                      quote.OutputAmount,
			"resultType":                        "OK",
			"walletNotSupportingFromBlockchain": false,
			"missingBlockchains":                []string{},
			"priceImpactUsd":                    "0",
			"priceImpactUsdPercent":             "0",
			"swaps":                             []map[string]any{},
			"scores":                            []map[string]any{},
			"tags":                              []map[string]any{{"label": "MVP", "value": "internal"}},
		}},
	}
}

func graphQLCreateTx(variables map[string]any) map[string]any {
	input := inputMap(variables)
	return map[string]any{
		"ok":        true,
		"error":     nil,
		"errorCode": 0,
		"traceId":   0,
		"transaction": map[string]any{
			"type":                 "EVM",
			"blockChain":           "EVM",
			"isApprovalTx":         false,
			"from":                 "",
			"to":                   "",
			"spender":              "",
			"data":                 "0x",
			"value":                "0",
			"gasLimit":             "0",
			"gasPrice":             "0",
			"maxPriorityFeePerGas": "0",
			"maxFeePerGas":         "0",
			"nonce":                "0",
			"identifier":           stringValue(input, "requestId"),
			"instructions":         []any{},
			"recentBlockhash":      "",
			"signatures":           []any{},
			"serializedMessage":    "",
			"txType":               "mvp",
		},
		"signUserTransactionEvm": map[string]any{
			"user_id":              "",
			"to":                   "",
			"data":                 "0x",
			"value":                "0",
			"chain":                "EVM",
			"maxFeePerGas":         "0",
			"maxPriorityFeePerGas": "0",
			"gasLimit":             "0",
			"gasPrice":             "0",
			"from":                 "",
			"signed_transaction":   "",
		},
	}
}

func graphQLSignTx(variables map[string]any) map[string]any {
	input := inputMap(variables)
	return map[string]any{
		"signature":            "mvp-signature-" + stringValue(input, "requestId"),
		"from":                 "",
		"to":                   "",
		"data":                 "0x",
		"value":                "0",
		"gasLimit":             "0",
		"gasPrice":             "0",
		"maxPriorityFeePerGas": "0",
		"maxFeePerGas":         "0",
		"nonce":                "0",
		"blockChain":           "EVM",
	}
}

func graphQLCheckStatus(variables map[string]any) map[string]any {
	input := inputMap(variables)
	return map[string]any{
		"status":       "success",
		"extraMessage": "",
		"failedType":   "",
		"timestamp":    time.Now().Unix(),
		"outputAmount": "0",
		"diagnosisUrl": "",
		"steps":        map[string]any{},
		"outputType":   "mvp",
		"error":        nil,
		"errorCode":    0,
		"traceId":      0,
		"explorerUrl":  []map[string]any{},
		"referrals":    []map[string]any{},
		"newTx":        nil,
		"outputToken":  map[string]any{"blockchain": "", "symbol": "", "image": "", "address": "", "usdPrice": "0", "decimals": 0, "name": "", "isPopular": false, "isSecondaryCoin": false, "coinSource": "internal", "coinSourceUrl": "", "supportedSwappers": []string{}},
		"bridgeExtra":  map[string]any{"requireRefundAction": false, "srcTx": stringValue(input, "txHash"), "destTx": ""},
	}
}

func (h *Handler) graphQLQuoteV2(ctx context.Context, variables map[string]any) map[string]any {
	input := inputMap(variables)
	quote, err := h.service.Quote(ctx, QuoteRequest{
		UserID:      stringValue(input, "user"),
		ChainType:   stringValue(input, "originId"),
		InputToken:  stringValue(input, "originId"),
		OutputToken: stringValue(input, "destinationId"),
		InputAmount: firstNonEmpty(stringValue(input, "amount"), "0"),
		SlippageBps: 100,
	})
	if err != nil {
		return map[string]any{"requestId": stringValue(input, "requestId"), "type": stringValue(input, "type"), "description": err.Error(), "errorCode": "400", "items": []any{}}
	}
	return map[string]any{
		"requestId":               firstNonEmpty(stringValue(input, "requestId"), quote.ID),
		"type":                    firstNonEmpty(stringValue(input, "type"), "swap"),
		"description":             "",
		"errorCode":               "0",
		"outPutAmountFormatted":   quote.OutputAmount,
		"gasTopupAmount":          "0",
		"gasTopupAmountFormatted": "0",
		"gasTopupAmountUsd":       "0",
		"gasAmountFormatted":      "0",
		"platformFeeAmountFormat": quote.PlatformFeeAmount,
		"platformFeeSymbol":       "",
		"thresholdCapacity":       quote.MinOutputAmount,
		"items":                   []map[string]any{},
	}
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

func slippageBps(input map[string]any) int {
	raw := stringValue(input, "slippage", "slippageBps")
	if raw == "" {
		return 100
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 100
	}
	if parsed <= 0 {
		return 100
	}
	if parsed <= 50 {
		return int(parsed * 100)
	}
	return int(parsed)
}

func bpsToPercent(bps int) string {
	if bps <= 0 {
		return "0"
	}
	value := float64(bps) / 100
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
