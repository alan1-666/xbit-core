package hypertrader

import (
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
	input := inputMap(variables)
	switch strings.ToLower(operation) {
	case "getsymbollist", "searchsymbol", "getpopularsymbol":
		symbols, err := h.service.ListSymbols(ctx, stringValue(input, "q", "keyword", "symbol"), stringValue(input, "category", "type"), intValue(input, 50, "limit"))
		if err != nil {
			return nil, err
		}
		return map[string]any{graphQLOperationKey(operation): map[string]any{"list": graphQLSymbols(symbols)}}, nil
	case "getfavoritesymbols":
		symbols, err := h.service.ListSymbols(ctx, "", "", 20)
		if err != nil {
			return nil, err
		}
		return map[string]any{"getFavoriteSymbols": map[string]any{"list": graphQLSymbols(symbols[:minInt(len(symbols), 3)])}}, nil
	case "getnewsymbol":
		symbols, err := h.service.ListSymbols(ctx, "", "", 10)
		if err != nil {
			return nil, err
		}
		list := make([]string, 0, len(symbols))
		for _, symbol := range symbols {
			list = append(list, symbol.Symbol)
		}
		return map[string]any{"getNewSymbol": map[string]any{"list": list}}, nil
	case "getcategory":
		return map[string]any{"getCategory": map[string]any{"categories": []string{"major", "defi", "meme", "ai"}}}, nil
	case "getusersymbolpreference":
		pref, err := h.service.GetSymbolPreference(ctx, stringValue(input, "userId", "userAddress"), stringValue(input, "symbol"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getUserSymbolPreference": graphQLPreference(pref)}, nil
	case "updateusersymbolpreference", "upsertfavoritesymbol":
		pref, err := h.service.UpdateSymbolPreference(ctx, SymbolPreference{
			UserID:     stringValue(input, "userId", "userAddress"),
			Symbol:     stringValue(input, "symbol"),
			IsFavorite: boolValue(input, true, "isFavorite"),
			Leverage:   intValue(input, 5, "leverage"),
			IsCross:    boolValue(input, true, "isCross"),
		})
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(operation, "UpsertFavoriteSymbol") {
			return map[string]any{"upsertFavoriteSymbol": map[string]any{"status": "ok", "error": nil}}, nil
		}
		return map[string]any{"updateUserSymbolPreference": graphQLPreference(pref)}, nil
	case "updatefavoritesymbolorder":
		return map[string]any{"updateFavoriteSymbolOrder": map[string]any{"status": "ok", "message": ""}}, nil
	case "generatecloid":
		return map[string]any{"generateCloid": h.service.GenerateCloid(intValue(input, 1, "count"))}, nil
	case "logtransaction":
		return map[string]any{"logTransaction": map[string]any{"status": "ok", "error": nil}}, nil
	case "getbanners":
		return map[string]any{"getBanners": map[string]any{"data": []map[string]any{}, "message": ""}}, nil
	case "routebanner":
		return map[string]any{"routeBanner": map[string]any{"message": "ok"}}, nil
	case "gethotsearchs", "gethotsearches":
		return map[string]any{"getHotSearches": map[string]any{"message": "", "data": []map[string]any{{"id": "btc", "board": "futures", "mode": "symbol", "symbol": "BTC", "chainId": 0, "tokenContract": "", "tokenName": "Bitcoin", "displayText": "BTC Perp", "showFlame": true}}}}, nil
	case "getuserbalance":
		account, err := h.service.Account(ctx, stringValue(input, "userAddress"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getUserBalance": map[string]any{"balance": account.Balance, "oneDayChange": account.OneDayChange, "oneDayPercentChange": account.OneDayPercentChange}}, nil
	case "getuserprevdaybalance":
		return map[string]any{"getUserPrevDayBalance": map[string]any{"Balance": "247820"}}, nil
	case "getuserposition":
		account, err := h.service.Account(ctx, stringValue(input, "userAddress"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getUserPosition": map[string]any{"rawUSD": account.RawUSD, "positions": graphQLPositions(account.Positions)}}, nil
	case "getusertradehistory":
		trades, err := h.service.TradeHistory(ctx, stringValue(input, "userAddress"), intValue(input, 100, "limit"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getUserTradeHistory": map[string]any{"histories": graphQLTrades(trades)}}, nil
	case "gethyperliquidorders", "getfutureorders":
		orders, err := h.service.Orders(ctx, OrderFilter{
			UserID:      stringValue(input, "userId"),
			UserAddress: stringValue(input, "userAddress"),
			Status:      stringValue(input, "status"),
			Symbol:      stringValue(input, "symbol"),
			Limit:       intValue(input, 50, "limit"),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{graphQLOperationKey(operation): map[string]any{"total": len(orders), "orders": graphQLFuturesOrders(orders)}}, nil
	case "gethyperliquidopenorders", "getopenorders":
		orders, err := h.service.OpenOrders(ctx, stringValue(input, "userAddress"))
		if err != nil {
			return nil, err
		}
		return map[string]any{graphQLOperationKey(operation): map[string]any{"total": len(orders), "orders": graphQLOpenOrders(orders)}}, nil
	case "createhyperliquidorder", "submithyperliquidorder":
		order, err := h.service.CreateOrder(ctx, createOrderInputFromMap(input))
		if err != nil {
			return nil, err
		}
		return map[string]any{graphQLOperationKey(operation): graphQLFuturesOrder(order)}, nil
	case "cancelhyperliquidorder":
		order, err := h.service.CancelOrder(ctx, CancelOrderInput{
			UserID:          stringValue(input, "userId"),
			UserAddress:     stringValue(input, "userAddress"),
			OrderID:         stringValue(input, "orderId", "id"),
			Cloid:           stringValue(input, "cloid"),
			Symbol:          stringValue(input, "symbol"),
			ExchangePayload: asMap(input["exchangePayload"]),
			ExchangeAction:  asMap(input["exchangeAction"]),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"cancelHyperLiquidOrder": graphQLFuturesOrder(order)}, nil
	case "gethyperliquidorderstatus", "synchyperliquidorderstatus":
		order, err := h.service.SyncOrderStatus(ctx, OrderStatusInput{
			UserID:          stringValue(input, "userId"),
			UserAddress:     stringValue(input, "userAddress"),
			OrderID:         stringValue(input, "orderId", "id"),
			ProviderOrderID: stringValue(input, "providerOrderId", "oid"),
			Cloid:           stringValue(input, "cloid", "clientOrderId"),
			Symbol:          stringValue(input, "symbol", "coin"),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{graphQLOperationKey(operation): graphQLFuturesOrder(order)}, nil
	case "updatehyperliquidleverage":
		result, err := h.service.UpdateLeverage(ctx, UpdateLeverageInput{
			UserID:          stringValue(input, "userId"),
			UserAddress:     stringValue(input, "userAddress"),
			Symbol:          stringValue(input, "symbol"),
			Leverage:        intValue(input, 5, "leverage"),
			IsCross:         boolValue(input, true, "isCross"),
			ExchangePayload: asMap(input["exchangePayload"]),
			ExchangeAction:  asMap(input["exchangeAction"]),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"updateHyperLiquidLeverage": graphQLProviderAction(result)}, nil
	case "getfundingrates":
		rates, err := h.service.FundingRates(ctx, stringValue(input, "symbol"), intValue(input, 8, "limit"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getFundingRates": graphQLFundingRates(rates)}, nil
	case "gethyperliquidauditevents":
		events, err := h.service.AuditEvents(ctx, stringValue(input, "userId"), intValue(input, 50, "limit"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"getHyperLiquidAuditEvents": graphQLAuditEvents(events)}, nil
	case "getfirstdepositusdc":
		return map[string]any{"getFirstDepositUSDC": map[string]any{"isFirst": false}}, nil
	case "checkuserdeprecatedasset":
		return map[string]any{"checkUserDeprecatedAsset": map[string]any{"deprecated": false, "confirmedBackup": true, "assets": []any{}}}, nil
	case "confirmassetbackup":
		return map[string]any{"confirmAssetBackup": true}, nil
	case "getactivesmartmoney", "getrecentactivesmartmoney", "getmyfollowedsmartmoney":
		traders, err := h.service.SmartMoney(ctx, intValue(variables, 20, "pageSize", "limit"))
		if err != nil {
			return nil, err
		}
		return map[string]any{graphQLOperationKey(operation): graphQLSmartMoneyResponse(traders)}, nil
	case "gettradertagdefinitions":
		return map[string]any{"getTraderTagDefinitions": graphQLTags(defaultTags())}, nil
	case "gettradertagsbyaddress":
		return map[string]any{"getTraderTagsByAddress": graphQLTags(defaultTags())}, nil
	case "analyzesmartmoneystrategy":
		return map[string]any{"analyzeSmartMoneyStrategy": map[string]any{"strategyCn": "趋势跟随，偏好高流动性主流合约。", "strategyEn": "Trend-following trader focused on liquid majors.", "analyzedAt": time.Now().UTC()}}, nil
	case "getfollowercount":
		return map[string]any{"getFollowerCount": map[string]any{"userAddress": stringValue(variables, "userAddress"), "count": 120}}, nil
	case "getsmartmoneyroi":
		return map[string]any{"getSmartMoneyRoi": map[string]any{"success": true, "message": "", "data": map[string]any{"userAddress": stringValue(variables, "userAddress"), "roi": "82.4", "periodDays": 30}}}, nil
	case "getsmartmoneymetrics30d":
		return map[string]any{"getSmartMoneyMetrics30d": map[string]any{"success": true, "message": "", "data": map[string]any{"userAddress": stringValue(variables, "userAddress"), "roe30d": "82.4", "winRate30d": "0.62", "sharpeRatio30d": "2.1", "maxDrawdown30d": "0.08", "totalPnl30d": "42100", "profitFactor": "2.4", "periodDays": 30}}}, nil
	case "getuserpositionholdingtime":
		return map[string]any{"getUserPositionHoldingTime": []map[string]any{{"id": "holding-1", "userAddress": stringValue(variables, "userAddress"), "coin": "BTC", "lastFillsId": "fill-1", "lastOpenTime": time.Now().Add(-2 * time.Hour), "timeSum": 7200, "totalHoldingTime": 7200, "status": "open", "updatedAt": time.Now().UTC(), "createdAt": time.Now().Add(-2 * time.Hour)}}}, nil
	case "gettradingsession":
		return map[string]any{"getTradingSession": map[string]any{"slot0004Count": 3, "slot0408Count": 2, "slot0812Count": 6, "slot1216Count": 4, "slot1620Count": 5, "slot2024Count": 7, "statDate": time.Now().UTC().Format("2006-01-02")}}, nil
	case "listaddressgroups":
		groups, err := h.service.Groups(ctx, stringValue(variables, "userId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"listAddressGroups": graphQLGroups(groups)}, nil
	case "createaddressgroup":
		group, err := h.service.CreateGroup(ctx, stringValue(input, "name"), stringValue(input, "userId"), boolValue(input, false, "isDefault"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"createAddressGroup": graphQLGroup(group)}, nil
	case "updateaddressgroup":
		group, err := h.service.UpdateGroup(ctx, stringValue(input, "id"), stringValue(input, "name"), boolValue(input, false, "isDefault"), intValue(input, 0, "order"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"updateAddressGroup": graphQLGroup(group)}, nil
	case "deleteaddressgroup":
		return map[string]any{"deleteAddressGroup": h.service.DeleteGroup(ctx, stringValue(variables, "id")) == nil}, nil
	case "listaddresses":
		addresses, err := h.service.Addresses(ctx, stringValue(variables, "groupId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"listAddresses": graphQLAddresses(addresses)}, nil
	case "createaddress":
		address, err := h.service.CreateAddress(ctx, stringValue(input, "address"), stringValue(input, "remarkName"), stringSliceValue(input, "groupIds"), stringValue(input, "userId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"createAddress": graphQLAddress(address)}, nil
	case "batchcreateaddresses", "importaddresses":
		addresses := stringSliceValue(input, "addresses")
		created := make([]map[string]any, 0, len(addresses))
		for _, raw := range addresses {
			address, err := h.service.CreateAddress(ctx, raw, "", stringSliceValue(input, "groupIds"), stringValue(input, "userId"))
			if err == nil {
				created = append(created, graphQLAddress(address))
			}
		}
		if strings.EqualFold(operation, "ImportAddresses") {
			return map[string]any{"importAddresses": map[string]any{"totalCount": len(addresses), "successCount": len(created), "failedCount": 0, "errors": []string{}, "addresses": created}}, nil
		}
		return map[string]any{"batchCreateAddresses": created}, nil
	case "updateaddress", "updateaddressgroupsforaddress":
		address, err := h.service.UpdateAddress(ctx, firstNonEmpty(stringValue(input, "id"), stringValue(input, "addressId")), stringValue(input, "remarkName"), stringSliceValue(input, "groupIds"))
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(operation, "UpdateAddressGroupsForAddress") {
			return map[string]any{"updateAddressGroupsForAddress": map[string]any{"addressId": address.ID, "groupIds": address.GroupIDs, "updatedAt": address.UpdatedAt}}, nil
		}
		return map[string]any{"updateAddress": graphQLAddress(address)}, nil
	case "batchupdateaddressgroups":
		groups, _ := h.service.Groups(ctx, stringValue(input, "userId"))
		return map[string]any{"batchUpdateAddressGroups": graphQLGroups(groups)}, nil
	case "deleteaddress":
		return map[string]any{"deleteAddress": h.service.DeleteAddress(ctx, stringValue(variables, "id")) == nil}, nil
	case "getflowaddressongroup":
		groups, _ := h.service.Groups(ctx, "")
		return map[string]any{"getFlowAddressOngroup": graphQLGroups(groups)}, nil
	case "getaddress":
		addresses, _ := h.service.Addresses(ctx, "")
		if len(addresses) == 0 {
			return map[string]any{"getAddress": nil}, nil
		}
		return map[string]any{"getAddress": graphQLAddress(addresses[0])}, nil
	case "getfollowedaddressespositions":
		account, _ := h.service.Account(ctx, "")
		return map[string]any{"getFollowedAddressesPositions": graphQLFollowedPositions(account.Positions)}, nil
	case "getfollowedaddresseslatestpositions":
		account, _ := h.service.Account(ctx, "")
		return map[string]any{"getFollowedAddressesLatestPositions": map[string]any{"totalCount": len(account.Positions), "positions": graphQLPositions(account.Positions)}}, nil
	case "checkhyperliquidwallet", "updatehyperliquidwallet":
		status, err := h.service.WalletStatus(ctx, stringValue(input, "userAddress"))
		if err != nil {
			return nil, err
		}
		return map[string]any{graphQLOperationKey(operation): status}, nil
	case "signhyperliquidcancelorder", "signhyperliquidcreateorder", "signhyperliquidupdateleverage", "approvehyperliquidapproveagent", "approvehyperliquidfeebuilder":
		result, err := h.service.Sign(ctx, stringValue(input, "userId"), stringValue(input, "userAddress"), operation, input)
		if err != nil {
			return nil, err
		}
		return map[string]any{graphQLOperationKey(operation): result}, nil
	case "listhyperliquidagentwallets":
		wallets, err := h.service.AgentWallets(ctx, stringValue(input, "userAddress"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"listHyperLiquidAgentWallets": wallets}, nil
	case "activatehyperliquidagentwallet":
		wallet, err := h.service.ActivateAgentWallet(ctx, ActivateAgentWalletInput{
			UserID:       stringValue(input, "userId"),
			UserAddress:  stringValue(input, "userAddress"),
			AgentAddress: stringValue(input, "agentAddress"),
			Status:       stringValue(input, "status"),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"activateHyperLiquidAgentWallet": wallet}, nil
	case "approvewithdrawhyperliquid":
		return map[string]any{"approveWithdrawHyperLiquid": map[string]any{"signedTransaction": "0xsigned-local-hyperliquid-withdraw"}}, nil
	case "createfundingswap":
		return map[string]any{"createFundingSwap": graphQLFundingSwap(input)}, nil
	case "createfuturetransaction":
		return map[string]any{"createFutureTransaction": graphQLFutureTransaction(input)}, nil
	default:
		return map[string]any{graphQLOperationKey(operation): map[string]any{"success": true, "message": "mvp placeholder", "data": []any{}}}, nil
	}
}

func (h *Handler) writeGraphQLError(w http.ResponseWriter, r *http.Request, status int, code string, message string) {
	meta := map[string]any{"retryable": false}
	if traceID := requestid.FromContext(r.Context()); traceID != "" {
		meta["traceId"] = traceID
	}
	httpx.WriteJSON(w, status, graphQLResponse{Errors: []graphQLError{{Message: message, Extensions: map[string]any{"code": code, "meta": meta}}}})
}

func inferOperationName(operationName string, query string) string {
	if strings.TrimSpace(operationName) != "" {
		return strings.TrimSpace(operationName)
	}
	known := []string{"GetFavoriteSymbols", "GetSymbolList", "UpsertFavoriteSymbol", "UpdateFavoriteSymbolOrder", "GetCategory", "GetUserSymbolPreference", "UpdateUserSymbolPreference", "SearchSymbol", "GetPopularSymbol", "GetNewSymbol", "GenerateCloid", "LogTransaction", "GetBanners", "RouteBanner", "GetHotSearchs", "GetUserBalance", "GetUserPrevDayBalance", "GetUserPosition", "GetUserTradeHistory", "GetHyperLiquidOrders", "GetFutureOrders", "GetHyperLiquidOpenOrders", "GetOpenOrders", "CreateHyperLiquidOrder", "SubmitHyperLiquidOrder", "CancelHyperLiquidOrder", "GetHyperLiquidOrderStatus", "SyncHyperLiquidOrderStatus", "UpdateHyperLiquidLeverage", "GetFundingRates", "GetHyperLiquidAuditEvents", "GetFirstDepositUSDC", "CheckUserDeprecatedAsset", "ConfirmAssetBackup", "GetActiveSmartMoney", "GetRecentActiveSmartMoney", "GetMyFollowedSmartMoney", "GetTraderTagDefinitions", "GetTraderTagsByAddress", "AnalyzeSmartMoneyStrategy", "GetFollowerCount", "GetSmartMoneyRoi", "GetSmartMoneyMetrics30d", "GetUserPositionHoldingTime", "GetTradingSession", "ListAddressGroups", "CreateAddressGroup", "UpdateAddressGroup", "DeleteAddressGroup", "ListAddresses", "CreateAddress", "BatchCreateAddresses", "ImportAddresses", "UpdateAddress", "UpdateAddressGroupsForAddress", "BatchUpdateAddressGroups", "DeleteAddress", "GetFlowAddressOngroup", "GetAddress", "GetFollowedAddressesPositions", "GetFollowedAddressesLatestPositions", "CheckHyperLiquidWallet", "updateHyperLiquidWallet", "signHyperLiquidCancelOrder", "signHyperLiquidCreateOrder", "signHyperLiquidUpdateLeverage", "approveHyperLiquidApproveAgent", "approveHyperLiquidFeeBuilder", "ListHyperLiquidAgentWallets", "ActivateHyperLiquidAgentWallet", "ApproveWithdrawHyperLiquid", "CreateFundingSwap", "CreateFutureTransaction"}
	for _, name := range known {
		if strings.Contains(query, name) {
			return name
		}
	}
	return ""
}

func graphQLOperationKey(operation string) string {
	switch strings.ToLower(operation) {
	case "getsymbollist":
		return "getSymbolList"
	case "searchsymbol":
		return "searchSymbol"
	case "getpopularsymbol":
		return "getPopularSymbol"
	case "gethyperliquidorders":
		return "getHyperLiquidOrders"
	case "getfutureorders":
		return "getFutureOrders"
	case "createhyperliquidorder":
		return "createHyperLiquidOrder"
	case "submithyperliquidorder":
		return "submitHyperLiquidOrder"
	case "getactivesmartmoney":
		return "getActiveSmartMoney"
	case "getrecentactivesmartmoney":
		return "getRecentActiveSmartMoney"
	case "getmyfollowedsmartmoney":
		return "getMyFollowedSmartMoney"
	case "checkhyperliquidwallet":
		return "checkHyperLiquidWallet"
	case "updatehyperliquidwallet":
		return "updateHyperLiquidWallet"
	case "signhyperliquidcancelorder":
		return "signHyperLiquidCancelOrder"
	case "signhyperliquidcreateorder":
		return "signHyperLiquidCreateOrder"
	case "signhyperliquidupdateleverage":
		return "signHyperLiquidUpdateLeverage"
	case "approvehyperliquidapproveagent":
		return "approveHyperLiquidApproveAgent"
	case "approvehyperliquidfeebuilder":
		return "approveHyperLiquidFeeBuilder"
	case "listhyperliquidagentwallets":
		return "listHyperLiquidAgentWallets"
	case "activatehyperliquidagentwallet":
		return "activateHyperLiquidAgentWallet"
	default:
		if operation == "" {
			return operation
		}
		return strings.ToLower(operation[:1]) + operation[1:]
	}
}

func graphQLSymbols(symbols []Symbol) []map[string]any {
	out := make([]map[string]any, 0, len(symbols))
	for _, symbol := range symbols {
		out = append(out, map[string]any{"symbol": symbol.Symbol, "aliasName": symbol.AliasName, "maxLeverage": symbol.MaxLeverage, "marketCap": symbol.MarketCap, "volume": symbol.Volume, "changPxPercent": symbol.ChangePercent, "openInterest": symbol.OpenInterest, "currentPrice": symbol.CurrentPrice, "type": symbol.Type, "quoteSymbol": symbol.QuoteSymbol})
	}
	return out
}

func graphQLPreference(pref SymbolPreference) map[string]any {
	return map[string]any{"isFavorite": pref.IsFavorite, "leverage": pref.Leverage, "isCross": pref.IsCross}
}

func graphQLPositions(positions []Position) []map[string]any {
	out := make([]map[string]any, 0, len(positions))
	for _, p := range positions {
		out = append(out, map[string]any{"address": p.Address, "coin": p.Coin, "createdAt": p.CreatedAt, "updatedAt": p.UpdatedAt, "positionType": p.PositionType, "szi": p.Szi, "leverageType": p.LeverageType, "leverageValue": p.LeverageValue, "entryPx": p.EntryPx, "positionValue": p.PositionValue, "unrealizedPnl": p.UnrealizedPnl, "returnOnEquity": p.ReturnOnEquity, "liquidationPx": p.LiquidationPx, "marginUsed": p.MarginUsed, "maxLeverage": p.MaxLeverage, "openTime": p.OpenTime, "cumFundingAllTime": p.CumFundingAllTime, "cumFundingSinceOpen": p.CumFundingSinceOpen, "cumFundingSinceChange": p.CumFundingSinceChange, "accountValue": p.AccountValue, "crossMaintenanceMarginUsed": p.CrossMaintenanceMarginUsed, "crossMarginRatio": p.CrossMarginRatio, "px": p.EntryPx, "side": p.Side, "time": p.Time, "startPosition": p.StartPosition, "dir": p.Dir, "closedPnl": p.ClosedPnl, "hash": p.Hash, "oid": p.Oid, "tid": p.Tid, "crossed": p.Crossed, "fee": p.Fee, "twapId": p.TwapID, "leverage": map[string]any{"type": p.LeverageType, "value": p.LeverageValue}, "symbol": p.Coin, "size": p.Szi, "fundingFee": p.CumFundingSinceOpen})
	}
	return out
}

func graphQLTrades(trades []TradeHistory) []map[string]any {
	out := make([]map[string]any, 0, len(trades))
	for _, t := range trades {
		out = append(out, map[string]any{"symbol": t.Symbol, "time": t.Time, "pnl": t.PnL, "pnlPercent": t.PnLPercent, "dir": t.Dir, "hash": t.Hash, "oid": t.Oid, "px": t.Px, "startPosition": t.StartPosition, "sz": t.Sz, "fee": t.Fee, "feeToken": t.FeeToken, "tid": t.Tid, "coin": t.Symbol, "side": t.Dir, "direction": t.Dir, "size": t.Sz, "price": t.Px, "tradeType": "perp"})
	}
	return out
}

func graphQLFuturesOrders(orders []FuturesOrder) []map[string]any {
	out := make([]map[string]any, 0, len(orders))
	for _, order := range orders {
		out = append(out, graphQLFuturesOrder(order))
	}
	return out
}

func graphQLFuturesOrder(order FuturesOrder) map[string]any {
	return map[string]any{
		"id":              order.ID,
		"userId":          order.UserID,
		"userAddress":     order.UserAddress,
		"symbol":          order.Symbol,
		"coin":            order.Symbol,
		"side":            order.Side,
		"orderType":       order.OrderType,
		"type":            order.OrderType,
		"price":           order.Price,
		"size":            order.Size,
		"sz":              order.Size,
		"status":          order.Status,
		"cloid":           order.Cloid,
		"provider":        order.Provider,
		"providerOrderId": order.ProviderOrderID,
		"clientRequestId": order.ClientRequestID,
		"reduceOnly":      order.ReduceOnly,
		"timeInForce":     order.TimeInForce,
		"rawPayload":      order.RawPayload,
		"responsePayload": order.ResponsePayload,
		"createdAt":       order.CreatedAt,
		"updatedAt":       order.UpdatedAt,
		"submittedAt":     order.SubmittedAt,
		"cancelledAt":     order.CancelledAt,
	}
}

func graphQLOpenOrders(orders []OpenOrder) []map[string]any {
	out := make([]map[string]any, 0, len(orders))
	for _, order := range orders {
		out = append(out, graphQLOpenOrder(order))
	}
	return out
}

func graphQLOpenOrder(order OpenOrder) map[string]any {
	return map[string]any{
		"id":              order.ID,
		"userAddress":     order.UserAddress,
		"symbol":          order.Symbol,
		"coin":            order.Symbol,
		"side":            order.Side,
		"orderType":       order.OrderType,
		"type":            order.OrderType,
		"price":           order.Price,
		"px":              order.Price,
		"size":            order.Size,
		"sz":              order.Size,
		"originalSize":    order.OriginalSize,
		"origSz":          order.OriginalSize,
		"status":          order.Status,
		"cloid":           order.Cloid,
		"provider":        order.Provider,
		"providerOrderId": order.ProviderOrderID,
		"oid":             order.ProviderOrderID,
		"reduceOnly":      order.ReduceOnly,
		"timeInForce":     order.TimeInForce,
		"tif":             order.TimeInForce,
		"timestamp":       order.Timestamp,
		"rawPayload":      order.RawPayload,
		"createdAt":       order.CreatedAt,
		"updatedAt":       order.UpdatedAt,
	}
}

func graphQLProviderAction(result ProviderActionResult) map[string]any {
	return map[string]any{
		"action":      result.Action,
		"provider":    result.Provider,
		"requestId":   result.RequestID,
		"status":      result.Status,
		"signature":   result.Signature,
		"rawPayload":  result.RawPayload,
		"submittedAt": result.SubmittedAt,
	}
}

func graphQLFundingRates(rates []FundingRate) []map[string]any {
	out := make([]map[string]any, 0, len(rates))
	for _, rate := range rates {
		out = append(out, map[string]any{"symbol": rate.Symbol, "fundingRate": rate.FundingRate, "premium": rate.Premium, "nextFundingTime": rate.NextFundingTime, "updatedAt": rate.UpdatedAt})
	}
	return out
}

func graphQLAuditEvents(events []AuditEvent) []map[string]any {
	out := make([]map[string]any, 0, len(events))
	for _, event := range events {
		out = append(out, map[string]any{"id": event.ID, "userId": event.UserID, "userAddress": event.UserAddress, "action": event.Action, "riskLevel": event.RiskLevel, "payload": event.Payload, "createdAt": event.CreatedAt})
	}
	return out
}

func graphQLSmartMoneyResponse(traders []SmartMoneyTrader) map[string]any {
	return map[string]any{"success": true, "message": "", "data": graphQLTraders(traders), "pagination": map[string]any{"page": 1, "pageSize": len(traders), "total": len(traders), "totalPages": 1}}
}

func graphQLTraders(traders []SmartMoneyTrader) []map[string]any {
	out := make([]map[string]any, 0, len(traders))
	for _, t := range traders {
		out = append(out, map[string]any{"userAddress": t.UserAddress, "roi": t.ROI, "netPnl": t.NetPnL, "avgWinRate": t.AvgWinRate, "maxDrawdown": t.MaxDrawdown, "periodDays": t.PeriodDays, "sharpeRatio": t.SharpeRatio, "profitLossRatio": t.ProfitLossRatio, "profitFactor": t.ProfitFactor, "totalVolume": t.TotalVolume, "avgDailyVolume": t.AvgDailyVolume, "tradingDays": t.TradingDays, "totalTrades": t.TotalTrades, "uniqueCoinsCount": t.UniqueCoinsCount, "avgTradesPerDay": t.AvgTradesPerDay, "totalLongPnl": t.TotalLongPnL, "totalShortPnl": t.TotalShortPnL, "winningPnlTotal": t.WinningPnLTotal, "losingPnlTotal": t.LosingPnLTotal, "kolLabels": t.KOLLabels, "kolLabelsDescription": t.KOLLabelsDescription, "followerCount": t.FollowerCount, "remarkName": t.RemarkName, "groupIds": t.GroupIDs, "portfolioData": t.PortfolioData, "lastOperation": graphQLTrades([]TradeHistory{t.LastOperation})[0], "tags": graphQLTags(t.Tags)})
	}
	return out
}

func defaultTags() []TraderTag {
	return []TraderTag{{ID: 1, Category: "style", Name: "trend", NameCN: "趋势", Color: "#46C2A9", Priority: 1, Description: "Trend follower", CreatedAt: time.Now().UTC()}}
}

func graphQLTags(tags []TraderTag) []map[string]any {
	out := make([]map[string]any, 0, len(tags))
	for _, tag := range tags {
		out = append(out, map[string]any{"id": tag.ID, "category": tag.Category, "name": tag.Name, "nameCn": tag.NameCN, "color": tag.Color, "priority": tag.Priority, "description": tag.Description, "createdAt": tag.CreatedAt})
	}
	return out
}

func graphQLGroups(groups []AddressGroup) []map[string]any {
	out := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		out = append(out, graphQLGroup(group))
	}
	return out
}

func graphQLGroup(group AddressGroup) map[string]any {
	return map[string]any{"id": group.ID, "name": group.Name, "userId": group.UserID, "isDefault": group.IsDefault, "order": group.Order, "createdAt": group.CreatedAt, "updatedAt": group.UpdatedAt}
}

func graphQLAddresses(addresses []Address) []map[string]any {
	out := make([]map[string]any, 0, len(addresses))
	for _, address := range addresses {
		out = append(out, graphQLAddress(address))
	}
	return out
}

func graphQLAddress(address Address) map[string]any {
	return map[string]any{"id": address.ID, "address": address.Address, "remarkName": address.RemarkName, "groupIds": address.GroupIDs, "ownerUserId": address.OwnerUserID, "userAddress": address.UserAddress, "profit1d": address.Profit1d, "profit7d": address.Profit7d, "profit30d": address.Profit30d, "createdAt": address.CreatedAt, "updatedAt": address.UpdatedAt}
}

func graphQLFollowedPositions(positions []Position) map[string]any {
	groups := make([]map[string]any, 0, len(positions))
	for _, p := range positions {
		groups = append(groups, map[string]any{"coin": p.Coin, "addressCount": 1, "positionCount": 1, "totalSzi": p.Szi, "totalPositionValue": p.PositionValue, "totalUnrealizedPnl": p.UnrealizedPnl, "totalMarginUsed": p.MarginUsed, "avgLeverage": p.LeverageValue, "longPositionValue": p.PositionValue, "shortPositionValue": "0", "longAvgLeverage": p.LeverageValue, "shortAvgLeverage": 0, "totalDiffPositionValue": p.PositionValue, "positions": graphQLPositions([]Position{p})})
	}
	return map[string]any{"totalAddresses": len(positions), "totalPositions": len(positions), "lastUpdated": time.Now().UTC(), "positionGroups": groups}
}

func graphQLFundingSwap(input map[string]any) map[string]any {
	return map[string]any{"id": "funding-local", "fromAddress": stringValue(input, "fromAddress"), "chainId": stringValue(input, "chainId"), "token": stringValue(input, "token"), "amount": stringValue(input, "amount"), "fee": "0", "crossChainFee": "0", "crossChainFeeUnit": "USDC", "toAddress": stringValue(input, "toAddress"), "toChainId": stringValue(input, "toChainId"), "toToken": stringValue(input, "toToken"), "toAmount": stringValue(input, "amount"), "route": "local-mvp", "createdAt": time.Now().UTC()}
}

func graphQLFutureTransaction(input map[string]any) map[string]any {
	return map[string]any{"type": stringValue(input, "type"), "fromAddress": stringValue(input, "fromAddress"), "chainId": stringValue(input, "chainId"), "token": stringValue(input, "token"), "amount": stringValue(input, "amount"), "fee": "0", "toAddress": stringValue(input, "toAddress"), "toChainId": stringValue(input, "toChainId"), "txHash": "0xfuture-local", "status": "submitted", "errorCode": "", "errorMessage": "", "nonce": "0", "createdAt": time.Now().UTC()}
}

func createOrderInputFromMap(input map[string]any) CreateOrderInput {
	return CreateOrderInput{
		UserID:          stringValue(input, "userId"),
		UserAddress:     stringValue(input, "userAddress", "address"),
		Symbol:          stringValue(input, "symbol", "coin"),
		Side:            stringValue(input, "side", "direction"),
		OrderType:       stringValue(input, "orderType", "type"),
		Price:           stringValue(input, "price", "px"),
		Size:            stringValue(input, "size", "sz"),
		Cloid:           stringValue(input, "cloid", "clientOrderId"),
		ClientRequestID: stringValue(input, "clientRequestId", "requestId"),
		ReduceOnly:      boolValue(input, false, "reduceOnly"),
		TimeInForce:     stringValue(input, "timeInForce", "tif"),
		ExchangePayload: asMap(input["exchangePayload"]),
		ExchangeAction:  asMap(input["exchangeAction"]),
		RawPayload:      input,
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
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			return strconv.Itoa(v)
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

func boolValue(m map[string]any, fallback bool, keys ...string) bool {
	for _, key := range keys {
		value, ok := m[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v
		case string:
			parsed, err := strconv.ParseBool(v)
			if err == nil {
				return parsed
			}
		}
	}
	return fallback
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
		case string:
			if v != "" {
				return []string{v}
			}
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
