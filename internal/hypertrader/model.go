package hypertrader

import "time"

type Symbol struct {
	Symbol        string    `json:"symbol"`
	AliasName     string    `json:"aliasName"`
	MaxLeverage   int       `json:"maxLeverage"`
	MarketCap     string    `json:"marketCap"`
	Volume        string    `json:"volume"`
	ChangePercent string    `json:"changPxPercent"`
	OpenInterest  string    `json:"openInterest"`
	CurrentPrice  string    `json:"currentPrice"`
	Type          string    `json:"type"`
	QuoteSymbol   string    `json:"quoteSymbol"`
	Category      string    `json:"category"`
	CreatedAt     time.Time `json:"createdAt"`
}

type SymbolPreference struct {
	Symbol     string `json:"symbol"`
	UserID     string `json:"userId"`
	IsFavorite bool   `json:"isFavorite"`
	Leverage   int    `json:"leverage"`
	IsCross    bool   `json:"isCross"`
}

type Position struct {
	Address                    string    `json:"address,omitempty"`
	Coin                       string    `json:"coin"`
	CreatedAt                  time.Time `json:"createdAt"`
	UpdatedAt                  time.Time `json:"updatedAt"`
	PositionType               string    `json:"positionType"`
	Szi                        string    `json:"szi"`
	LeverageType               string    `json:"leverageType"`
	LeverageValue              int       `json:"leverageValue"`
	EntryPx                    string    `json:"entryPx"`
	PositionValue              string    `json:"positionValue"`
	UnrealizedPnl              string    `json:"unrealizedPnl"`
	ReturnOnEquity             string    `json:"returnOnEquity"`
	LiquidationPx              string    `json:"liquidationPx"`
	MarginUsed                 string    `json:"marginUsed"`
	MaxLeverage                int       `json:"maxLeverage"`
	OpenTime                   int64     `json:"openTime"`
	CumFundingAllTime          string    `json:"cumFundingAllTime"`
	CumFundingSinceOpen        string    `json:"cumFundingSinceOpen"`
	CumFundingSinceChange      string    `json:"cumFundingSinceChange"`
	AccountValue               string    `json:"accountValue"`
	CrossMaintenanceMarginUsed string    `json:"crossMaintenanceMarginUsed"`
	CrossMarginRatio           string    `json:"crossMarginRatio"`
	Side                       string    `json:"side,omitempty"`
	Time                       int64     `json:"time,omitempty"`
	StartPosition              string    `json:"startPosition,omitempty"`
	Dir                        string    `json:"dir,omitempty"`
	ClosedPnl                  string    `json:"closedPnl,omitempty"`
	Hash                       string    `json:"hash,omitempty"`
	Oid                        int64     `json:"oid,omitempty"`
	Tid                        int64     `json:"tid,omitempty"`
	Crossed                    bool      `json:"crossed,omitempty"`
	Fee                        string    `json:"fee,omitempty"`
	TwapID                     string    `json:"twapId,omitempty"`
}

type AccountBalance struct {
	Balance             string     `json:"balance"`
	OneDayChange        string     `json:"oneDayChange"`
	OneDayPercentChange string     `json:"oneDayPercentChange"`
	RawUSD              string     `json:"rawUSD"`
	Positions           []Position `json:"positions"`
}

type TradeHistory struct {
	Symbol        string `json:"symbol"`
	Time          int64  `json:"time"`
	PnL           string `json:"pnl"`
	PnLPercent    string `json:"pnlPercent"`
	Dir           string `json:"dir"`
	Hash          string `json:"hash"`
	Oid           int64  `json:"oid"`
	Px            string `json:"px"`
	StartPosition string `json:"startPosition"`
	Sz            string `json:"sz"`
	Fee           string `json:"fee"`
	FeeToken      string `json:"feeToken"`
	Tid           int64  `json:"tid"`
}

type FuturesOrder struct {
	ID              string         `json:"id"`
	UserID          string         `json:"userId,omitempty"`
	UserAddress     string         `json:"userAddress,omitempty"`
	Symbol          string         `json:"symbol"`
	Side            string         `json:"side"`
	OrderType       string         `json:"orderType"`
	Price           string         `json:"price,omitempty"`
	Size            string         `json:"size"`
	Status          string         `json:"status"`
	Cloid           string         `json:"cloid,omitempty"`
	Provider        string         `json:"provider"`
	ProviderOrderID string         `json:"providerOrderId,omitempty"`
	ClientRequestID string         `json:"clientRequestId,omitempty"`
	ReduceOnly      bool           `json:"reduceOnly"`
	TimeInForce     string         `json:"timeInForce,omitempty"`
	RawPayload      map[string]any `json:"rawPayload,omitempty"`
	ResponsePayload map[string]any `json:"responsePayload,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	SubmittedAt     *time.Time     `json:"submittedAt,omitempty"`
	CancelledAt     *time.Time     `json:"cancelledAt,omitempty"`
}

type CreateOrderInput struct {
	UserID          string         `json:"userId"`
	UserAddress     string         `json:"userAddress"`
	Symbol          string         `json:"symbol"`
	Side            string         `json:"side"`
	OrderType       string         `json:"orderType"`
	Price           string         `json:"price"`
	Size            string         `json:"size"`
	Cloid           string         `json:"cloid"`
	ClientRequestID string         `json:"clientRequestId"`
	ReduceOnly      bool           `json:"reduceOnly"`
	TimeInForce     string         `json:"timeInForce"`
	ExchangePayload map[string]any `json:"exchangePayload,omitempty"`
	RawPayload      map[string]any `json:"rawPayload"`
}

type CancelOrderInput struct {
	UserID          string         `json:"userId"`
	UserAddress     string         `json:"userAddress"`
	OrderID         string         `json:"orderId"`
	Cloid           string         `json:"cloid"`
	Symbol          string         `json:"symbol"`
	ExchangePayload map[string]any `json:"exchangePayload,omitempty"`
}

type OrderStatusInput struct {
	UserID          string `json:"userId"`
	UserAddress     string `json:"userAddress"`
	OrderID         string `json:"orderId"`
	ProviderOrderID string `json:"providerOrderId"`
	Cloid           string `json:"cloid"`
	Symbol          string `json:"symbol"`
}

type OrderStatus struct {
	OrderID         string         `json:"orderId"`
	ProviderOrderID string         `json:"providerOrderId"`
	Cloid           string         `json:"cloid,omitempty"`
	Symbol          string         `json:"symbol,omitempty"`
	Status          string         `json:"status"`
	FilledSize      string         `json:"filledSize,omitempty"`
	RemainingSize   string         `json:"remainingSize,omitempty"`
	AveragePrice    string         `json:"averagePrice,omitempty"`
	RawPayload      map[string]any `json:"rawPayload,omitempty"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type UpdateLeverageInput struct {
	UserID          string         `json:"userId"`
	UserAddress     string         `json:"userAddress"`
	Symbol          string         `json:"symbol"`
	Leverage        int            `json:"leverage"`
	IsCross         bool           `json:"isCross"`
	ExchangePayload map[string]any `json:"exchangePayload,omitempty"`
}

type OrderFilter struct {
	UserID      string
	UserAddress string
	Status      string
	Symbol      string
	Limit       int
}

type ProviderActionResult struct {
	Action      string         `json:"action"`
	Provider    string         `json:"provider"`
	RequestID   string         `json:"requestId"`
	Status      string         `json:"status"`
	Signature   Signature      `json:"signature"`
	RawPayload  map[string]any `json:"rawPayload,omitempty"`
	SubmittedAt time.Time      `json:"submittedAt"`
}

type AuditEvent struct {
	ID          string         `json:"id"`
	UserID      string         `json:"userId,omitempty"`
	UserAddress string         `json:"userAddress,omitempty"`
	Action      string         `json:"action"`
	RiskLevel   string         `json:"riskLevel"`
	Payload     map[string]any `json:"payload"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type FundingRate struct {
	Symbol          string    `json:"symbol"`
	FundingRate     string    `json:"fundingRate"`
	Premium         string    `json:"premium"`
	NextFundingTime int64     `json:"nextFundingTime"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type SmartMoneyTrader struct {
	UserAddress          string         `json:"userAddress"`
	ROI                  string         `json:"roi"`
	NetPnL               string         `json:"netPnl"`
	AvgWinRate           string         `json:"avgWinRate"`
	MaxDrawdown          string         `json:"maxDrawdown"`
	PeriodDays           int            `json:"periodDays"`
	SharpeRatio          string         `json:"sharpeRatio"`
	ProfitLossRatio      string         `json:"profitLossRatio"`
	ProfitFactor         string         `json:"profitFactor"`
	TotalVolume          string         `json:"totalVolume"`
	AvgDailyVolume       string         `json:"avgDailyVolume"`
	TradingDays          int            `json:"tradingDays"`
	TotalTrades          int            `json:"totalTrades"`
	UniqueCoinsCount     int            `json:"uniqueCoinsCount"`
	AvgTradesPerDay      string         `json:"avgTradesPerDay"`
	TotalLongPnL         string         `json:"totalLongPnl"`
	TotalShortPnL        string         `json:"totalShortPnl"`
	WinningPnLTotal      string         `json:"winningPnlTotal"`
	LosingPnLTotal       string         `json:"losingPnlTotal"`
	KOLLabels            []string       `json:"kolLabels"`
	KOLLabelsDescription []string       `json:"kolLabelsDescription"`
	FollowerCount        int            `json:"followerCount"`
	RemarkName           string         `json:"remarkName"`
	GroupIDs             []string       `json:"groupIds"`
	PortfolioData        map[string]any `json:"portfolioData"`
	LastOperation        TradeHistory   `json:"lastOperation"`
	Tags                 []TraderTag    `json:"tags"`
}

type TraderTag struct {
	ID          int       `json:"id,omitempty"`
	Category    string    `json:"category"`
	Name        string    `json:"name"`
	NameCN      string    `json:"nameCn,omitempty"`
	Color       string    `json:"color,omitempty"`
	Priority    int       `json:"priority,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
}

type AddressGroup struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	UserID    string    `json:"userId,omitempty"`
	IsDefault bool      `json:"isDefault"`
	Order     int       `json:"order,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Address struct {
	ID          string    `json:"id"`
	Address     string    `json:"address"`
	RemarkName  string    `json:"remarkName"`
	GroupIDs    []string  `json:"groupIds"`
	OwnerUserID string    `json:"ownerUserId"`
	UserAddress string    `json:"userAddress"`
	Profit1d    string    `json:"profit1d"`
	Profit7d    string    `json:"profit7d"`
	Profit30d   string    `json:"profit30d"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type HyperliquidWalletStatus struct {
	ApprovedAgent     bool   `json:"approvedAgent"`
	SetReferral       bool   `json:"setReferral"`
	SetFeeBuilder     bool   `json:"setFeeBuilder"`
	Agent             string `json:"agent"`
	AgentName         string `json:"agentName"`
	FeeBuilderAddress string `json:"feeBuilderAddress"`
	FeeBuilderPercent string `json:"feeBuilderPercent"`
	ReferralCode      string `json:"referralCode"`
}

type Signature struct {
	R string `json:"r"`
	S string `json:"s"`
	V int    `json:"v"`
}
